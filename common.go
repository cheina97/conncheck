package conncheck

import (
	"encoding/json"
	"fmt"
	"time"
)

func (msg Msg) String() string {
	return fmt.Sprintf("ClusterID: %s, MsgType: %s, Timestamp: %d:%d:%d.%d",
		msg.ClusterID,
		msg.MsgType,
		msg.TimeStamp.Hour(), msg.TimeStamp.Minute(), msg.TimeStamp.Second(), msg.TimeStamp.Nanosecond())
}

type Msg struct {
	ClusterID string    `json:"clusterID"`
	MsgType   MsgTypes  `json:"msgType"`
	TimeStamp time.Time `json:"timeStamp"`
}

type MsgTypes string

const (
	PING MsgTypes = "PING"
	PONG MsgTypes = "PONG"
)

type UpdateFunc func(connected bool) error

func UnmarshalMsg(bytes []byte) (*Msg, error) {
	msg := &Msg{}
	err := json.Unmarshal(bytes, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func MarshalMsg(msg Msg) ([]byte, error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return b, nil
}
