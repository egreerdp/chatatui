package repository

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/EwanGreer/chatatui/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func HashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

type User struct {
	BaseModel
	Name   string
	APIKey string `gorm:"uniqueIndex"`
	Rooms  []Room `gorm:"many2many:room_members;"`
}

func userToDomain(row *User) *domain.User {
	return &domain.User{
		ID:     row.ID,
		Name:   row.Name,
		APIKey: row.APIKey,
	}
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(u *domain.User) error {
	row := &User{Name: u.Name, APIKey: u.APIKey}
	if err := r.db.Create(row).Error; err != nil {
		return err
	}
	u.ID = row.ID
	return nil
}

func (r *UserRepository) GetByAPIKey(apiKey string) (*domain.User, error) {
	var row User
	err := r.db.Where("api_key = ?", HashAPIKey(apiKey)).First(&row).Error
	if err != nil {
		return nil, err
	}
	return userToDomain(&row), nil
}

func (r *UserRepository) GetByID(id uuid.UUID) (*domain.User, error) {
	var row User
	err := r.db.First(&row, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return userToDomain(&row), nil
}

func (r *UserRepository) List(limit, offset int) ([]User, error) {
	var users []User
	err := r.db.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error
	return users, err
}

func (r *UserRepository) Update(user *User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&User{}, "id = ?", id).Error
}
