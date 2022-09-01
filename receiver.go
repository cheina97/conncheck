package conncheck

import (
	"context"
	"fmt"
	"net"
	"time"

	"k8s.io/klog/v2"
)

// Receiver is a conncheck receiver.
type Receiver struct {
	buff         []byte
	redirectChan map[string]chan Msg
	conn         *net.UDPConn
	ctx          context.Context
	Cancel       context.CancelFunc
}

// NewReceiver creates a new conncheck receiver.
func NewReceiver(conn *net.UDPConn, ctx context.Context, cancel context.CancelFunc) *Receiver {
	return &Receiver{
		buff:         make([]byte, buffSize),
		redirectChan: make(map[string]chan Msg),
		conn:         conn,
		ctx:          ctx,
		Cancel:       cancel,
	}
}

func (r *Receiver) SendPong(raddr *net.UDPAddr, msg *Msg) error {
	time.Sleep(time.Second * 3)
	msg.MsgType = PONG
	b, err := MarshalMsg(*msg)
	if err != nil {
		return fmt.Errorf("conncheck receiver: failed to marshal msg: %w", err)
	}
	_, err = r.conn.WriteToUDP(b, raddr)
	if err != nil {
		return fmt.Errorf("conncheck receiver: failed to write to %s: %w", raddr.String(), err)
	}
	klog.V(8).Infof("conncheck receiver: sent a msg -> %s", msg)
	return nil
}

func (r *Receiver) RedirectPong(msg *Msg) error {
	ch, ok := r.redirectChan[msg.ClusterID]
	if ch == nil || !ok {
		return fmt.Errorf("conncheck receiver: channel closed for %s", msg.ClusterID)
	}
	klog.V(8).Infof("conncheck receiver: redirecting a msg -> %s", msg)
	ch <- *msg
	return nil
}

// Run starts the conncheck receiver inside a network namespace.
// The Do() function has to be called inside the goroutine because when a go routine start,
// it chooses a random thread and this thread could run in a netns which is not the gateway netns.
func (r *Receiver) Run(ctx context.Context) error {
	klog.V(9).Infof("conncheck receiver: starting")
	for {
		select {
		case <-ctx.Done():
			klog.V(9).Infof("conncheck receiver: stopped")
			return nil
		default:
			n, raddr, err := r.conn.ReadFromUDP(r.buff)
			if err != nil {
				return fmt.Errorf("conncheck receiver: failed to read from %s: %w", raddr.String(), err)
			}
			msgr, err := UnmarshalMsg(r.buff[:n])
			if err != nil {
				return fmt.Errorf("conncheck receiver: failed to unmarshal msg: %w", err)
			}
			klog.V(9).Infof("conncheck receiver: received a msg -> %s", msgr)
			switch msgr.MsgType {
			case PING:
				raddr.Port = port
				err = r.SendPong(raddr, msgr)
			case PONG:
				err = r.RedirectPong(msgr)
			}
			if err != nil {
				klog.Errorf("conncheck receiver: %v", err)
			}
		}
	}
}

func (r *Receiver) AddRedirectChan(clusterID string, ch chan Msg) error {
	if _, ok := r.redirectChan[clusterID]; ok {
		return fmt.Errorf("conncheck receiver: channel already exists for %s", clusterID)
	}
	r.redirectChan[clusterID] = ch
	return nil
}
