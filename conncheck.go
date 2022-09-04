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
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

// ConnChecker is a struct that holds the receiver and senders.
type ConnChecker struct {
	receiver *Receiver
	// key is the target cluster ID.
	senders map[string]*Sender
	conn    *net.UDPConn
}

// NewConnChecker creates a new ConnChecker.
func NewConnChecker() (*ConnChecker, error) {
	addr := &net.UDPAddr{
		Port: port,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", addr)
	klog.V(9).Infof("conncheck socket: listening on %s", addr.String())
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP: %w", err)
	}
	ctxReceiver, cancelReceiver := context.WithCancel(context.Background())
	connChecker := ConnChecker{
		receiver: NewReceiver(ctxReceiver, conn, cancelReceiver),
		senders:  make(map[string]*Sender),
		conn:     conn,
	}
	return &connChecker, nil
}

// RunReceiver runs the receiver.
func (c *ConnChecker) RunReceiver() error {
	if err := c.receiver.Run(c.receiver.ctx); err != nil {
		return fmt.Errorf("failed to run receiver: %w", err)
	}
	return nil
}

// RunReceiverDisconnectObserver runs the receiver disconnect observer.
func (c *ConnChecker) RunReceiverDisconnectObserver() error {
	ctxErrorChecker, cancel := context.WithCancel(c.receiver.ctx)
	defer cancel()
	if err := c.receiver.RunDisconnectObserver(ctxErrorChecker); err != nil {
		return fmt.Errorf("failed to run receiver disconnect checker: %w", err)
	}
	return nil
}

// AddAndRunSender create a new sender and runs it.
func (c *ConnChecker) AddAndRunSender(clusterID, ip string, updateCallback UpdateFunc) error {
	if _, ok := c.senders[clusterID]; ok {
		return fmt.Errorf("sender %s already exists", clusterID)
	}
	var err error
	ctxSender, cancelSender := context.WithCancel(context.Background())
	c.senders[clusterID] = NewSender(ctxSender, clusterID, cancelSender, c.conn, ip)

	err = c.receiver.InitPeer(clusterID, updateCallback)
	if err != nil {
		return fmt.Errorf("failed to add redirect chan: %w", err)
	}

	pingCallback := func(ctx context.Context) (done bool, err error) {
		err = c.senders[clusterID].SendPing(ctx)
		if err != nil {
			klog.Warningf("failed to send ping: %w", err)
		}
		return false, nil
	}
	_ = wait.PollImmediateInfiniteWithContext(ctxSender, PeriodicPingInterval, pingCallback)
	klog.Infof("conncheck sender %s stopped", clusterID)
	return nil
}

// DelAndStopSender stops and deletes a sender. If sender has been already stoped and deleted is a no-op function.
func (c *ConnChecker) DelAndStopSender(clusterID string) {
	if _, ok := c.senders[clusterID]; ok {
		c.senders[clusterID].cancel()
		delete(c.senders, clusterID)
	}
	if _, ok := c.receiver.peers[clusterID]; !ok {
		c.receiver.m.Lock()
		delete(c.receiver.peers, clusterID)
		c.receiver.m.Unlock()
	}
}

// GetLatency returns the latency with clusterID.
func (c *ConnChecker) GetLatency(clusterID string) (time.Duration, error) {
	c.receiver.m.RLock()
	defer c.receiver.m.RUnlock()
	if peer, ok := c.receiver.peers[clusterID]; ok {
		return peer.latency, nil
	}
	return 0, fmt.Errorf("sender %s not found", clusterID)
}

// GetConnected returns the connection status with clusterID.
func (c *ConnChecker) GetConnected(clusterID string) (bool, error) {
	c.receiver.m.RLock()
	defer c.receiver.m.RUnlock()
	if peer, ok := c.receiver.peers[clusterID]; ok {
		return peer.connected, nil
	}
	return false, fmt.Errorf("sender %s not found", clusterID)
}
