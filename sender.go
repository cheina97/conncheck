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
	clusterID         string
	ctx               context.Context
	cancel            func()
	conn              *net.UDPConn
	raddr             net.UDPAddr
	started           bool
	buff              []byte
	ch                chan Msg
	updateCallback    func(connected bool) error
	lastLatency       time.Duration
	connected         bool
	lastSeq           uint64
	consecutiveErrors int
}

// NewSender creates a new conncheck sender.
func NewSender(clusterID string, ctx context.Context, cancel func(), conn *net.UDPConn, ip string, ch chan Msg, updateCallback func(connected bool) error) *Sender {
	return &Sender{
		clusterID:         clusterID,
		ctx:               ctx,
		cancel:            cancel,
		raddr:             net.UDPAddr{IP: net.ParseIP(ip), Port: port},
		conn:              conn,
		started:           false,
		buff:              make([]byte, buffSize),
		ch:                ch,
		updateCallback:    updateCallback,
		lastSeq:           0,
		consecutiveErrors: 0,
	}
}

// Start starts the conncheck sender periodic ping.
func (s *Sender) SendPing(ctx context.Context) (time.Duration, error) {
	ctxPing, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	s.lastSeq++
	msgOut := Msg{ClusterID: s.clusterID, Seq: s.lastSeq, MsgType: PING}
	klog.V(8).Infof("conncheck sender: sending a msg -> %s", msgOut)
	b, err := MarshalMsg(msgOut)
	if err != nil {
		return 0, fmt.Errorf("conncheck sender: failed to marshal msg: %v", err)
	}
	start := time.Now()
	_, err = s.conn.WriteToUDP(b, &s.raddr)
	if err != nil {
		return 0, fmt.Errorf("conncheck sender: failed to write to %s: %v", s.raddr.String(), err)
	}
	for {
		select {
		case msgIn := <-s.ch:
			end := time.Now()
			if msgIn.Seq == msgOut.Seq {
				klog.V(9).Infof("Conncheck sender: received a msg -> %s", msgIn)
				return end.Sub(start), nil
			}
		case <-ctxPing.Done():
			return 0, fmt.Errorf("conncheck sender: context cancelled")
		}
	}
}

// Stop stops the conncheck sender.
func (s *Sender) Stop() error {
	if !s.started {
		return fmt.Errorf("sender not started")
	}
	s.cancel()
	return nil
}
