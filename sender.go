package conncheck

import (
	"context"
	"fmt"
	"net"
	"time"

	"k8s.io/klog/v2"
)

// Sender is a sender for the conncheck server.
type Sender struct {
	clusterID string
	ctx       context.Context
	cancel    func()
	conn      *net.UDPConn
	raddr     net.UDPAddr
	started   bool
	buff      []byte
}

// NewSender creates a new conncheck sender.
func NewSender(clusterID string, ctx context.Context, cancel func(), conn *net.UDPConn, ip string) *Sender {
	return &Sender{
		clusterID: clusterID,
		ctx:       ctx,
		cancel:    cancel,
		raddr:     net.UDPAddr{IP: net.ParseIP(ip), Port: port},
		conn:      conn,
		started:   false,
		buff:      make([]byte, buffSize),
	}
}

// Start starts the conncheck sender periodic ping.
func (s *Sender) SendPing(ctx context.Context) error {
	msgOut := Msg{ClusterID: s.clusterID, MsgType: PING, TimeStamp: time.Now()}
	b, err := MarshalMsg(msgOut)
	if err != nil {
		return fmt.Errorf("conncheck sender: failed to marshal msg: %v", err)
	}
	_, err = s.conn.WriteToUDP(b, &s.raddr)
	if err != nil {
		return fmt.Errorf("conncheck sender: failed to write to %s: %v", s.raddr.String(), err)
	}
	klog.V(8).Infof("conncheck sender: sent a msg -> %s", msgOut)
	return nil
}

// Stop stops the conncheck sender.
func (s *Sender) Stop() error {
	if !s.started {
		return fmt.Errorf("sender not started")
	}
	s.cancel()
	return nil
}
