package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	UUID     uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	Content  []byte
	SenderID uint
	Sender   User `gorm:"foreignKey:SenderID"`
	RoomID   uint
	Room     Room `gorm:"foreignKey:RoomID"`
}

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(msg *Message) error {
	if msg.UUID == uuid.Nil {
		msg.UUID = uuid.New()
	}
	return r.db.Create(msg).Error
}

func (r *MessageRepository) GetByID(id uint) (*Message, error) {
	var msg Message
	err := r.db.Preload("Sender").Preload("Room").First(&msg, id).Error
	return &msg, err
}

func (r *MessageRepository) GetByUUID(uid uuid.UUID) (*Message, error) {
	var msg Message
	err := r.db.Preload("Sender").Preload("Room").Where("uuid = ?", uid).First(&msg).Error
	return &msg, err
}

func (r *MessageRepository) GetByRoom(roomID uint, limit, offset int) ([]Message, error) {
	var messages []Message
	err := r.db.Preload("Sender").
		Where("room_id = ?", roomID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, err
}

func (r *MessageRepository) GetByRoomUUID(roomUUID uuid.UUID, limit, offset int) ([]Message, error) {
	var room Room
	if err := r.db.Where("uuid = ?", roomUUID).First(&room).Error; err != nil {
		return nil, err
	}
	return r.GetByRoom(room.ID, limit, offset)
}

func (r *MessageRepository) Delete(id uint) error {
	return r.db.Delete(&Message{}, id).Error
}
