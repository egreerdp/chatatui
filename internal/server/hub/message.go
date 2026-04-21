package hub

import (
	"encoding/json"
	"time"
)

type MessageType string

const (
	MessageTypeChat   MessageType = "chat"
	MessageTypeSystem MessageType = "system"
	MessageTypeTyping MessageType = "typing"
	MessageTypeError  MessageType = "error"
)

func (m MessageType) String() string {
	return string(m)
}

type Message struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id"`
	Author    string      `json:"author"`
	Content   string      `json:"content"`
	Timestamp time.Time   `json:"timestamp"`
}

func (m *Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}
