package repository

import "gorm.io/gorm"

type Message struct {
	gorm.Model

	Content []byte
	Sender  string // Foreign key
	Room    string // Foreign key
}
