package bus

import (
	"encoding/json"
)

type Message struct {
	AppID    string          `json:"app_id"`
	Event    string          `json:"event"`
	Data     json.RawMessage `json:"data"`
	Channel  string          `json:"channel"`
	SocketID string          `json:"socket_id"`
}

func (m Message) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

type Bus interface {
	Publish(Message) error
	Channel(appID string) (chan Message, error)
	Close()
}
