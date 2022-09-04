package conncheck

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

// Receiver is a conncheck receiver.

type Peer struct {
	connected             bool
	latency               time.Duration
	lastReceivedTimestamp time.Time
	updateCallback        UpdateFunc
}
type Receiver struct {
	peers  map[string]*Peer
	m      sync.RWMutex
	buff   []byte
	conn   *net.UDPConn
	ctx    context.Context
	cancel context.CancelFunc
}

// NewReceiver creates a new conncheck receiver.
func NewReceiver(conn *net.UDPConn, ctx context.Context, cancel context.CancelFunc) *Receiver {
	return &Receiver{
		peers:  make(map[string]*Peer),
		buff:   make([]byte, buffSize),
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (r *Receiver) SendPong(raddr *net.UDPAddr, msg *Msg) error {
	msg.MsgType = PONG
	b, err := MarshalMsg(*msg)
	if err != nil {
		return fmt.Errorf("failed to marshal msg: %w", err)
	}
	_, err = r.conn.WriteToUDP(b, raddr)
	if err != nil {
		return fmt.Errorf("failed to write to %s: %w", raddr.String(), err)
	}
	klog.V(8).Infof("conncheck sender: sent a msg -> %s", msg)
	return nil
}

func (r *Receiver) ReceivePong(msg *Msg) error {
	r.m.Lock()
	defer r.m.Unlock()
	if peer, ok := r.peers[msg.ClusterID]; ok {
		now := time.Now()
		peer.lastReceivedTimestamp = now
		peer.latency = now.Sub(msg.TimeStamp)
		peer.connected = true

		err := peer.updateCallback(true)
		if err != nil {
			return fmt.Errorf("failed to update peer %s: %w", msg.ClusterID, err)
		}
		return nil
	} else {
		return fmt.Errorf("%s is not in the peersInfo map", msg.ClusterID)
	}
}

func (r *Receiver) InitPeer(clusterID string, updateCallback UpdateFunc) error {
	r.m.Lock()
	defer r.m.Unlock()
	if _, ok := r.peers[clusterID]; ok {
		return fmt.Errorf("conncheck receiver: %s is already in the peersInfo map", clusterID)
	}
	r.peers[clusterID] = &Peer{
		connected:             false,
		latency:               0,
		lastReceivedTimestamp: time.Now(),
		updateCallback:        updateCallback,
	}
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
			klog.V(8).Infof("conncheck receiver: received a msg -> %s", msgr)
			switch msgr.MsgType {
			case PING:
				raddr.Port = port
				err = r.SendPong(raddr, msgr)
			case PONG:
				err = r.ReceivePong(msgr)
			}
			if err != nil {
				klog.Errorf("conncheck receiver: %v", err)
			}
		}
	}
}

func (r *Receiver) RunDisconnectChecker(ctx context.Context) error {
	klog.V(9).Infof("conncheck receiver disconnect checker: starting")
	err := wait.PollImmediateInfiniteWithContext(ctx, TimeExceededCheckInterval, func(ctx context.Context) (done bool, err error) {
		r.m.Lock()
		defer r.m.Unlock()
		for id, peer := range r.peers {
			if time.Since(peer.lastReceivedTimestamp) > ExceedingTime {
				peer.connected = false
				peer.latency = 0
				err := peer.updateCallback(false)
				klog.V(8).Infof("conncheck receiver: %s unreachable", id)
				if err != nil {
					klog.Errorf("conncheck receiver: failed to update peer %s: %w", peer.lastReceivedTimestamp, err)
				}
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("conncheck receiver: failed to run error checker: %v", err)
	}
	return nil
}
