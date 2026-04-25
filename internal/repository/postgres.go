package repository

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresDB struct {
	*gorm.DB
	users    *UserRepository
	rooms    *RoomRepository
	messages *MessageRepository
}

func NewPostgresDB(dsn string) (*PostgresDB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	if err := db.SetupJoinTable(&Room{}, "Members", &RoomMember{}); err != nil {
		return nil, fmt.Errorf("setting up join table: %w", err)
	}

	if err := db.AutoMigrate(&User{}, &Room{}, &Message{}, &RoomMember{}); err != nil {
		return nil, fmt.Errorf("migrating database: %w", err)
	}

	return &PostgresDB{
		DB:       db,
		users:    NewUserRepository(db),
		rooms:    NewRoomRepository(db),
		messages: NewMessageRepository(db),
	}, nil
}

func (s *PostgresDB) Users() *UserRepository {
	return s.users
}

func (s *PostgresDB) Rooms() *RoomRepository {
	return s.rooms
}

func (s *PostgresDB) Messages() *MessageRepository {
	return s.messages
}
