package domain

import (
	"encoding/json"
	"time"
)

type WireMessageType string

const (
	WireMessageTypeChat   WireMessageType = "chat"
	WireMessageTypeSystem WireMessageType = "system"
	WireMessageTypeTyping WireMessageType = "typing"
	WireMessageTypeError  WireMessageType = "error"
)

func (t WireMessageType) String() string { return string(t) }

type WireMessage struct {
	Type      WireMessageType `json:"type"`
	ID        string          `json:"id"`
	Author    string          `json:"author"`
	Content   string          `json:"content"`
	Timestamp time.Time       `json:"timestamp"`
}

func (m *WireMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}
