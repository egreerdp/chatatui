package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Room struct {
	BaseModel
	Name    string
	Members []User `gorm:"many2many:room_members;"`
}

type RoomRepository struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) Create(room *Room) error {
	return r.db.Create(room).Error
}

func (r *RoomRepository) GetByID(id uuid.UUID) (*Room, error) {
	var room Room
	err := r.db.Preload("Members").First(&room, "id = ?", id).Error
	return &room, err
}

func (r *RoomRepository) List(limit, offset int) ([]Room, error) {
	var rooms []Room
	err := r.db.Preload("Members").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rooms).Error
	return rooms, err
}

func (r *RoomRepository) Update(room *Room) error {
	return r.db.Save(room).Error
}

func (r *RoomRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&Room{}, "id = ?", id).Error
}

func (r *RoomRepository) AddMember(roomID, userID uuid.UUID) error {
	return r.db.Exec("INSERT INTO room_members (room_id, user_id) VALUES (?, ?) ON CONFLICT DO NOTHING", roomID, userID).Error
}

func (r *RoomRepository) RemoveMember(roomID, userID uuid.UUID) error {
	return r.db.Exec("DELETE FROM room_members WHERE room_id = ? AND user_id = ?", roomID, userID).Error
}
