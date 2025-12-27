package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Room struct {
	gorm.Model
	UUID    uuid.UUID `gorm:"type:uuid;uniqueIndex"`
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
	if room.UUID == uuid.Nil {
		room.UUID = uuid.New()
	}
	return r.db.Create(room).Error
}

func (r *RoomRepository) GetByID(id uint) (*Room, error) {
	var room Room
	err := r.db.Preload("Members").First(&room, id).Error
	return &room, err
}

func (r *RoomRepository) GetByUUID(uid uuid.UUID) (*Room, error) {
	var room Room
	err := r.db.Preload("Members").Where("uuid = ?", uid).First(&room).Error
	return &room, err
}

func (r *RoomRepository) GetOrCreateByUUID(uid uuid.UUID) (*Room, error) {
	var room Room
	err := r.db.Where("uuid = ?", uid).First(&room).Error
	if err == gorm.ErrRecordNotFound {
		room = Room{UUID: uid}
		if err := r.db.Create(&room).Error; err != nil {
			return nil, err
		}
		return &room, nil
	}
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

func (r *RoomRepository) Delete(id uint) error {
	return r.db.Delete(&Room{}, id).Error
}

func (r *RoomRepository) AddMember(roomID, userID uint) error {
	return r.db.Exec("INSERT OR IGNORE INTO room_members (room_id, user_id) VALUES (?, ?)", roomID, userID).Error
}

func (r *RoomRepository) RemoveMember(roomID, userID uint) error {
	return r.db.Exec("DELETE FROM room_members WHERE room_id = ? AND user_id = ?", roomID, userID).Error
}
