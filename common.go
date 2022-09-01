package conncheck

import (
	"encoding/json"
	"fmt"
)

func (msg Msg) String() string {
	return fmt.Sprintf("ClusterID: %s, Seq: %d, MsgType: %s", msg.ClusterID, msg.Seq, msg.MsgType)
}

type Msg struct {
	ClusterID string   `json:"clusterID"`
	Seq       uint64   `json:"seq"`
	MsgType   MsgTypes `json:"msgType"`
}

type MsgTypes string

const (
	PING MsgTypes = "PING"
	PONG MsgTypes = "PONG"
)

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
