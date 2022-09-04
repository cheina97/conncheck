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
	receiver *Receiver
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
		receiver: NewReceiver(conn, ctxReceiver, cancelReceiver),
		senders:  make(map[string]*Sender),
		conn:     conn,
	}
	return &connChecker, nil
}

func (c *ConnChecker) RunReceiver() error {
	if err := c.receiver.Run(c.receiver.ctx); err != nil {
		return fmt.Errorf("failed to run receiver: %v", err)
	}
	return nil
}

func (c *ConnChecker) RunReceiverDisconnectObserver() error {
	ctxErrorChecker, cancel := context.WithCancel(c.receiver.ctx)
	defer cancel()
	if err := c.receiver.RunDisconnectChecker(ctxErrorChecker); err != nil {
		return fmt.Errorf("failed to run receiver disconnect checker: %v", err)
	}
	return nil
}

func (c *ConnChecker) AddAndRunSender(clusterID, ip string, updateCallback UpdateFunc) error {
	if _, ok := c.senders[clusterID]; ok {
		return fmt.Errorf("sender %s already exists", clusterID)
	}
	var err error
	ctxSender, cancelSender := context.WithCancel(context.Background())
	c.senders[clusterID] = NewSender(clusterID, ctxSender, cancelSender, c.conn, ip)

	err = c.receiver.InitPeer(clusterID, updateCallback)
	if err != nil {
		return fmt.Errorf("failed to add redirect chan: %v", err)
	}

	pingCallback := func(ctx context.Context) (done bool, err error) {
		err = c.senders[clusterID].SendPing(ctx)
		if err != nil {
			klog.Warningf("failed to send ping: %v", err)
		}
		return false, nil
	}
	_ = wait.PollImmediateInfiniteWithContext(ctxSender, PeriodicPingInterval, pingCallback)
	klog.Infof("conncheck sender %s stopped", clusterID)
	return nil
}

func (c *ConnChecker) DelAndStopSender(clusterID string) error {
	if _, ok := c.senders[clusterID]; !ok {
		return fmt.Errorf("sender %s does not exist", clusterID)
	}
	c.senders[clusterID].cancel()
	delete(c.senders, clusterID)
	if _, ok := c.receiver.peers[clusterID]; !ok {
		return fmt.Errorf("peer %s not found", clusterID)
	}
	c.receiver.m.Lock()
	delete(c.receiver.peers, clusterID)
	c.receiver.m.Unlock()
	return nil
}

func (c *ConnChecker) GetLatency(clusterID string) (time.Duration, error) {
	c.receiver.m.RLock()
	defer c.receiver.m.RUnlock()
	if peer, ok := c.receiver.peers[clusterID]; ok {
		return peer.latency, nil
	} else {
		return 0, fmt.Errorf("sender %s not found", clusterID)
	}
}

func (c *ConnChecker) GetConnected(clusterID string) (bool, error) {
	c.receiver.m.RLock()
	defer c.receiver.m.RUnlock()
	if peer, ok := c.receiver.peers[clusterID]; ok {
		return peer.connected, nil
	} else {
		return false, fmt.Errorf("sender %s not found", clusterID)
	}
}
