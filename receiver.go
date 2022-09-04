// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

// Peer represents a peer.
type Peer struct {
	connected             bool
	latency               time.Duration
	lastReceivedTimestamp time.Time
	updateCallback        UpdateFunc
}

// Receiver is a receiver for conncheck messages.
type Receiver struct {
	peers  map[string]*Peer
	m      sync.RWMutex
	buff   []byte
	conn   *net.UDPConn
	ctx    context.Context
	cancel context.CancelFunc
}

// NewReceiver creates a new conncheck receiver.
func NewReceiver(ctx context.Context, conn *net.UDPConn, cancel context.CancelFunc) *Receiver {
	return &Receiver{
		peers:  make(map[string]*Peer),
		buff:   make([]byte, buffSize),
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}
}

// SendPong sends a PONG message to the given address.
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
	klog.V(8).Infof("conncheck receiver: sent a PONG -> %s", msg)
	return nil
}

// ReceivePong receives a PONG message.
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
	}
	return fmt.Errorf("%s is not in the peersInfo map", msg.ClusterID)
}

// InitPeer initializes a peer.
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

// Run starts the receiver.
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
				klog.V(8).Infof("conncheck receiver: received a PING -> %s", msgr)
				raddr.Port = port
				err = r.SendPong(raddr, msgr)
			case PONG:
				klog.V(8).Infof("conncheck receiver: received a PONG -> %s", msgr)
				err = r.ReceivePong(msgr)
			}
			if err != nil {
				klog.Errorf("conncheck receiver: %v", err)
			}
		}
	}
}

// RunDisconnectObserver starts the disconnect observer.
func (r *Receiver) RunDisconnectObserver(ctx context.Context) error {
	klog.V(9).Infof("conncheck receiver disconnect checker: starting")
	err := wait.PollImmediateInfiniteWithContext(ctx, ConnectionCheckInterval, func(ctx context.Context) (done bool, err error) {
		r.m.Lock()
		defer r.m.Unlock()
		for id, peer := range r.peers {
			if time.Since(peer.lastReceivedTimestamp) <= ExceedingTime {
				continue
			}
			klog.V(8).Infof("conncheck receiver: %s unreachable", id)
			peer.connected = false
			peer.latency = 0
			err := peer.updateCallback(false)
			if err != nil {
				klog.Errorf("conncheck receiver: failed to update peer %s: %w", peer.lastReceivedTimestamp, err)
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("conncheck receiver: failed to run error checker: %w", err)
	}
	return nil
}
