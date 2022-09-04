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
	"encoding/json"
	"fmt"
	"time"
)

// Msg represents a message sent between two nodes.
type Msg struct {
	ClusterID string    `json:"clusterID"`
	MsgType   MsgTypes  `json:"msgType"`
	TimeStamp time.Time `json:"timeStamp"`
}

func (msg Msg) String() string {
	return fmt.Sprintf("ClusterID: %s, MsgType: %s, Timestamp: %d:%d:%d.%d",
		msg.ClusterID,
		msg.MsgType,
		msg.TimeStamp.Hour(), msg.TimeStamp.Minute(), msg.TimeStamp.Second(), msg.TimeStamp.Nanosecond())
}

// MsgTypes represents the type of a message.
type MsgTypes string

const (
	// PING is the type of a ping message.
	PING MsgTypes = "PING"
	// PONG is the type of a pong message.
	PONG MsgTypes = "PONG"
)

// UpdateFunc is a callback function.
type UpdateFunc func(connected bool) error

// UnmarshalMsg unmarshals a message.
func UnmarshalMsg(bytes []byte) (*Msg, error) {
	msg := &Msg{}
	err := json.Unmarshal(bytes, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// MarshalMsg marshals a message.
func MarshalMsg(msg Msg) ([]byte, error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return b, nil
}
