package domain

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID        uuid.UUID
	SenderID  uuid.UUID
	Author    string
	Content   string
	RoomID    uuid.UUID
	CreatedAt time.Time
}

func (m Message) ToWireMessage() WireMessage {
	return WireMessage{
		Type:      WireMessageTypeChat,
		ID:        m.ID.String(),
		Author:    m.Author,
		Content:   m.Content,
		Timestamp: m.CreatedAt,
	}
}
