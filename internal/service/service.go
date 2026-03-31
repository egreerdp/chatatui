package service

import (
	"time"

	"github.com/google/uuid"
)

type RoomInfo struct {
	ID   uuid.UUID
	Name string
}

type MessageInfo struct {
	ID        uuid.UUID
	Author    string
	Content   string
	CreatedAt time.Time
}
