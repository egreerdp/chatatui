package domain

import (
	"time"

	"github.com/google/uuid"
)

type RoomMember struct {
	UserID          uuid.UUID
	Name            string
	LastConnectedAt time.Time
}
