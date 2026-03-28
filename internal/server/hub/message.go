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
)

type WireMessage struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id"`
	Author    string      `json:"author"`
	Content   string      `json:"content"`
	Timestamp time.Time   `json:"timestamp"`
}

func (m *WireMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}
