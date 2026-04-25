package domain

import "github.com/google/uuid"

type User struct {
	ID     uuid.UUID
	Name   string
	APIKey string
}
