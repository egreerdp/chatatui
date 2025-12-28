package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	UUID   uuid.UUID `gorm:"type:uuid;uniqueIndex"`
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
	if user.UUID == uuid.Nil {
		user.UUID = uuid.New()
	}
	return r.db.Create(user).Error
}

func (r *UserRepository) GetByAPIKey(apiKey string) (*User, error) {
	var user User
	err := r.db.Where("api_key = ?", apiKey).First(&user).Error
	return &user, err
}

func (r *UserRepository) GetOrCreateByUUID(uid uuid.UUID, name string) (*User, error) {
	var user User
	err := r.db.Where("uuid = ?", uid).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		user = User{UUID: uid, Name: name}
		if err := r.db.Create(&user).Error; err != nil {
			return nil, err
		}
		return &user, nil
	}
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

func (r *UserRepository) Delete(id uint) error {
	return r.db.Delete(&User{}, id).Error
}
