package service

import (
	"errors"
	"log/slog"

	"github.com/EwanGreer/chatatui/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatService struct {
	rooms        RoomStore
	messages     MessageStore
	historyLimit int
}

func NewChatService(rooms RoomStore, messages MessageStore, historyLimit int) *ChatService {
	return &ChatService{rooms: rooms, messages: messages, historyLimit: historyLimit}
}

func (s *ChatService) GetRoom(id uuid.UUID) (*domain.Room, error) {
	room, err := s.rooms.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	return room, err
}

// JoinRoom records the user's membership and returns their message history as wire messages.
// A membership error is logged but does not prevent history from being returned.
func (s *ChatService) JoinRoom(roomID, userID uuid.UUID) ([]domain.WireMessage, error) {
	if err := s.rooms.AddMember(roomID, userID); err != nil {
		slog.Error("failed to record room membership", "error", err, "room_id", roomID, "user_id", userID)
	}
	msgs, err := s.messages.GetByRoom(roomID, s.historyLimit, 0)
	if err != nil {
		return nil, err
	}
	wire := make([]domain.WireMessage, len(msgs))
	for i, m := range msgs {
		wire[i] = m.ToWireMessage()
	}
	return wire, nil
}

// PublishMessage persists a message and returns it as a wire message ready to broadcast.
func (s *ChatService) PublishMessage(content []byte, senderID uuid.UUID, senderName string, roomID uuid.UUID) (*domain.WireMessage, error) {
	msg := &domain.Message{
		Content:  string(content),
		SenderID: senderID,
		Author:   senderName,
		RoomID:   roomID,
	}
	if err := s.messages.Create(msg); err != nil {
		return nil, err
	}
	wire := msg.ToWireMessage()
	return &wire, nil
}
