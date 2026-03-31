package repository

import (
	"crypto/sha256"
	"encoding/hex"

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

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) GetByAPIKey(apiKey string) (*User, error) {
	var user User
	err := r.db.Where("api_key = ?", HashAPIKey(apiKey)).First(&user).Error
	return &user, err
}

func (r *UserRepository) GetByID(id uuid.UUID) (*User, error) {
	var user User
	err := r.db.First(&user, "id = ?", id).Error
	return &user, err
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
