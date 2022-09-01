package conncheck

import (
	"context"
	"fmt"
	"net"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

type ConnChecker struct {
	Receiver *Receiver
	//key is the target cluster ID
	senders map[string]*Sender
	conn    *net.UDPConn
}

func NewConnChecker() (*ConnChecker, error) {
	addr := &net.UDPAddr{
		Port: port,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", addr)
	klog.V(9).Infof("conncheck socket: listening on %s", addr.String())
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP: %v", err)
	}
	ctxReceiver, cancelReceiver := context.WithCancel(context.Background())
	connChecker := ConnChecker{
		Receiver: NewReceiver(conn, ctxReceiver, cancelReceiver),
		senders:  make(map[string]*Sender),
		conn:     conn,
	}
	return &connChecker, nil
}

func (c *ConnChecker) RunReceiver() error {

	if err := c.Receiver.Run(c.Receiver.ctx); err != nil {
		return fmt.Errorf("failed to run receiver: %v", err)
	}
	return nil
}

func (c *ConnChecker) AddAndRunSender(clusterID, ip string, updateCallback func(connected bool) error) error {
	if _, ok := c.senders[clusterID]; ok {
		return fmt.Errorf("sender %s already exists", clusterID)
	}
	var err error

	ctxSender, cancelSender := context.WithCancel(context.Background())
	ch := make(chan Msg)
	c.senders[clusterID] = NewSender(clusterID, ctxSender, cancelSender, c.conn, ip, ch, updateCallback)

	err = c.Receiver.AddRedirectChan(clusterID, ch)
	if err != nil {
		return fmt.Errorf("failed to add redirect chan: %v", err)
	}

	err = wait.PollImmediateInfiniteWithContext(ctxSender, PeriodicPingInterval, func(ctx context.Context) (done bool, err error) {
		var connected, update bool
		latency, err := c.senders[clusterID].SendPing(ctxSender)
		if err != nil {
			c.senders[clusterID].consecutiveErrors++
			if c.senders[clusterID].consecutiveErrors == MaxConsecutiveErrors {
				c.senders[clusterID].lastLatency = 0
				c.senders[clusterID].connected = false
				connected = false
				update = true
				klog.Errorf("cluster %s unreachable", clusterID)
			}
		} else {
			c.senders[clusterID].consecutiveErrors = 0
			c.senders[clusterID].lastLatency = latency
			c.senders[clusterID].connected = true
			connected = true
			update = true
		}
		if update {
			err = updateCallback(connected)
			if err != nil {
				return false, fmt.Errorf("conncheck sender: failed to update status: %v", err)
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to run sender: %v", err)
	}
	c.clean(clusterID)
	klog.Infof("conncheck sender %s stopped", clusterID)
	return nil
	/* for {
		select {
		case <-ctxSender.Done():
			c.clean(clusterID)
			klog.Infof("conncheck sender %s stopped", clusterID)
			return nil
		default:
			var status netv1alpha1.ConnectionStatus
			latency, err := c.senders[clusterID].SendPing(ctxSender)
			if err != nil {
				c.senders[clusterID].consecutiveErrors++
				if c.senders[clusterID].consecutiveErrors == MaxConsecutiveErrors {
					c.senders[clusterID].lastLatency = 0
					c.senders[clusterID].connected = false
					status = netv1alpha1.ConnectionError
					klog.Errorf("cluster %s unreachable", clusterID)
				}
			} else {
				c.senders[clusterID].consecutiveErrors = 0
				c.senders[clusterID].lastLatency = latency
				c.senders[clusterID].connected = true
				status = netv1alpha1.Connected
			}
			if status != "" {
				err = c.senders[clusterID].updateCallback(status)
				if err != nil {
					return fmt.Errorf("conncheck sender: failed to update status: %v", err)
				}
			}
			time.Sleep(PeriodicPingInterval)
		}
	} */
}

func (c *ConnChecker) DelAndStopSender(clusterID string) error {
	if _, ok := c.senders[clusterID]; !ok {
		return fmt.Errorf("sender %s does not exist", clusterID)
	}
	c.senders[clusterID].cancel()
	return nil
}

func (c *ConnChecker) GetLatency(clusterID string) (time.Duration, error) {
	if _, ok := c.senders[clusterID]; !ok {
		return 0, fmt.Errorf("sender %s not found", clusterID)
	}
	return c.senders[clusterID].lastLatency, nil
}

func (c *ConnChecker) GetConnected(clusterID string) (bool, error) {
	if _, ok := c.senders[clusterID]; !ok {
		return false, fmt.Errorf("sender %s not found", clusterID)
	}
	return c.senders[clusterID].connected, nil
}

func (c *ConnChecker) clean(clusterID string) {
	if _, ok := c.Receiver.redirectChan[clusterID]; !ok {
		klog.Warning("redirect chan %s not found", clusterID)
	}
	if _, ok := c.senders[clusterID]; !ok {
		klog.Warning("sender %s not found", clusterID)
	}
	close(c.Receiver.redirectChan[clusterID])
	delete(c.Receiver.redirectChan, clusterID)
	delete(c.senders, clusterID)
}
