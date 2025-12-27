package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Room struct {
	gorm.Model
	Name    []byte
	UUID    uuid.UUID
	Members []User `gorm:"foreignKey:UUID"`
}
