package repository

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresDB struct {
	*gorm.DB
	users    *UserRepository
	rooms    *RoomRepository
	messages *MessageRepository
}

func NewPostgresDB(dsn string) *PostgresDB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	// Drop old tables to allow clean migration from integer PKs to UUID PKs.
	// This is acceptable during development; a proper migration will be needed for production.
	db.Exec("DROP TABLE IF EXISTS room_members, messages, rooms, users CASCADE")

	if err := db.AutoMigrate(&User{}, &Room{}, &Message{}); err != nil {
		panic("failed to migrate database: " + err.Error())
	}

	return &PostgresDB{
		DB:       db,
		users:    NewUserRepository(db),
		rooms:    NewRoomRepository(db),
		messages: NewMessageRepository(db),
	}
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
