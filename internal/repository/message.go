package repository

import (
	"github.com/EwanGreer/chatatui/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	BaseModel
	Content  []byte
	SenderID uuid.UUID `gorm:"type:uuid"`
	Sender   User      `gorm:"foreignKey:SenderID"`
	RoomID   uuid.UUID `gorm:"type:uuid"`
	Room     Room      `gorm:"foreignKey:RoomID"`
}

func messageToDomain(row *Message) domain.Message {
	return domain.Message{
		ID:        row.ID,
		SenderID:  row.SenderID,
		Author:    row.Sender.Name,
		Content:   string(row.Content),
		RoomID:    row.RoomID,
		CreatedAt: row.CreatedAt,
	}
}

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(msg *domain.Message) error {
	row := &Message{
		Content:  []byte(msg.Content),
		SenderID: msg.SenderID,
		RoomID:   msg.RoomID,
	}
	if err := r.db.Create(row).Error; err != nil {
		return err
	}
	msg.ID = row.ID
	msg.CreatedAt = row.CreatedAt
	return nil
}

func (r *MessageRepository) GetByRoom(roomID uuid.UUID, limit, offset int) ([]domain.Message, error) {
	var rows []Message
	err := r.db.Preload("Sender").
		Where("room_id = ?", roomID).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	messages := make([]domain.Message, len(rows))
	for i := range rows {
		messages[i] = messageToDomain(&rows[i])
	}
	return messages, nil
}

func (r *MessageRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&Message{}, "id = ?", id).Error
}
