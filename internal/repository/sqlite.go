package repository

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SQLiteDB struct {
	*gorm.DB
	users    *UserRepository
	rooms    *RoomRepository
	messages *MessageRepository
}

func NewSQLiteDB(dbName string) *SQLiteDB {
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	if err := db.AutoMigrate(&User{}, &Room{}, &Message{}); err != nil {
		panic("failed to migrate database")
	}

	return &SQLiteDB{
		DB:       db,
		users:    NewUserRepository(db),
		rooms:    NewRoomRepository(db),
		messages: NewMessageRepository(db),
	}
}

func (s *SQLiteDB) Users() *UserRepository {
	return s.users
}

func (s *SQLiteDB) Rooms() *RoomRepository {
	return s.rooms
}

func (s *SQLiteDB) Messages() *MessageRepository {
	return s.messages
}
