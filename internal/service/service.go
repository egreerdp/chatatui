package service

import (
	"github.com/EwanGreer/chatatui/internal/domain"
	"github.com/google/uuid"
)

type RoomStore interface {
	GetByID(id uuid.UUID) (*domain.Room, error)
	AddMember(roomID, userID uuid.UUID) error
}

type MessageStore interface {
	Create(msg *domain.Message) error
	GetByRoom(roomID uuid.UUID, limit, offset int) ([]domain.Message, error)
}
