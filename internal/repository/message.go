package repository

import (
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

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(msg *Message) error {
	return r.db.Create(msg).Error
}

func (r *MessageRepository) GetByID(id uuid.UUID) (*Message, error) {
	var msg Message
	err := r.db.Preload("Sender").Preload("Room").First(&msg, "id = ?", id).Error
	return &msg, err
}

func (r *MessageRepository) GetByRoom(roomID uuid.UUID, limit, offset int) ([]Message, error) {
	var messages []Message
	err := r.db.Preload("Sender").
		Where("room_id = ?", roomID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, err
}

func (r *MessageRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&Message{}, "id = ?", id).Error
}
