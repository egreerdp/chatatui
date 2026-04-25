package repository

import (
	"time"

	"github.com/EwanGreer/chatatui/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoomMember struct {
	RoomID          uuid.UUID `gorm:"primaryKey"`
	UserID          uuid.UUID `gorm:"primaryKey"`
	LastConnectedAt time.Time
	User            User
}

type Room struct {
	BaseModel
	Name    string
	Members []User `gorm:"many2many:room_members;"`
}

func memberToDomain(row *RoomMember) domain.RoomMember {
	return domain.RoomMember{
		UserID:          row.UserID,
		Name:            row.User.Name,
		LastConnectedAt: row.LastConnectedAt,
	}
}

func roomToDomain(row *Room) *domain.Room {
	members := make([]domain.RoomMember, len(row.Members))
	for i, m := range row.Members {
		members[i] = domain.RoomMember{
			UserID: m.ID,
			Name:   m.Name,
		}
	}
	return &domain.Room{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Members:   members,
	}
}

type RoomRepository struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

func (r *RoomRepository) Create(room *domain.Room) error {
	row := &Room{Name: room.Name}
	if err := r.db.Create(row).Error; err != nil {
		return err
	}
	room.ID = row.ID
	room.CreatedAt = row.CreatedAt
	room.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *RoomRepository) GetByID(id uuid.UUID) (*domain.Room, error) {
	var row Room
	err := r.db.Preload("Members").First(&row, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return roomToDomain(&row), nil
}

func (r *RoomRepository) List(limit, offset int) ([]domain.Room, error) {
	var rows []Room
	err := r.db.Preload("Members").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	rooms := make([]domain.Room, len(rows))
	for i := range rows {
		rooms[i] = *roomToDomain(&rows[i])
	}
	return rooms, nil
}

func (r *RoomRepository) Update(room *Room) error {
	return r.db.Save(room).Error
}

func (r *RoomRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&Room{}, "id = ?", id).Error
}

func (r *RoomRepository) ListRoomMembers(roomID uuid.UUID) ([]domain.RoomMember, error) {
	var rows []RoomMember
	err := r.db.Preload("User").Where("room_id = ?", roomID).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	members := make([]domain.RoomMember, len(rows))
	for i := range rows {
		members[i] = memberToDomain(&rows[i])
	}
	return members, nil
}

func (r *RoomRepository) AddMember(roomID, userID uuid.UUID) error {
	return r.db.Exec(
		"INSERT INTO room_members (room_id, user_id, last_connected_at) VALUES (?, ?, NOW()) ON CONFLICT (room_id, user_id) DO UPDATE SET last_connected_at = NOW()",
		roomID, userID,
	).Error
}

func (r *RoomRepository) RemoveMember(roomID, userID uuid.UUID) error {
	return r.db.Exec("DELETE FROM room_members WHERE room_id = ? AND user_id = ?", roomID, userID).Error
}
