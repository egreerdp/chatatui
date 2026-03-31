package service

import (
	"time"

	"github.com/egreerdp/chatatui/internal/repository"
	"github.com/google/uuid"
)

type ChatService struct {
	rooms    *repository.RoomRepository
	messages *repository.MessageRepository
}

func NewChatService(rooms *repository.RoomRepository, messages *repository.MessageRepository) *ChatService {
	return &ChatService{rooms: rooms, messages: messages}
}

func (s *ChatService) GetRoom(id uuid.UUID) (*RoomInfo, error) {
	room, err := s.rooms.GetByID(id)
	if err != nil {
		return nil, err
	}
	return &RoomInfo{ID: room.ID, Name: room.Name}, nil
}

func (s *ChatService) AddRoomMember(roomID, userID uuid.UUID) error {
	return s.rooms.AddMember(roomID, userID)
}

func (s *ChatService) GetMessageHistory(roomID uuid.UUID, limit, offset int) ([]MessageInfo, error) {
	messages, err := s.messages.GetByRoom(roomID, limit, offset)
	if err != nil {
		return nil, err
	}

	infos := make([]MessageInfo, len(messages))
	for i, m := range messages {
		infos[i] = MessageInfo{
			ID:        m.ID,
			Author:    m.Sender.Name,
			Content:   string(m.Content),
			CreatedAt: m.CreatedAt,
		}
	}
	return infos, nil
}

func (s *ChatService) PersistMessage(content []byte, senderID, roomID uuid.UUID) (uuid.UUID, time.Time, error) {
	msg := &repository.Message{
		Content:  content,
		SenderID: senderID,
		RoomID:   roomID,
	}
	if err := s.messages.Create(msg); err != nil {
		return uuid.Nil, time.Time{}, err
	}
	return msg.ID, msg.CreatedAt, nil
}
