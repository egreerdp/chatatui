package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name []byte
	UUID uuid.UUID
}
