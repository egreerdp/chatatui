package repository

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SQLiteDB struct {
	*gorm.DB
}

func NewSQLiteDB(dbName string) *SQLiteDB {
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// db.AutoMigrate(&Product{})

	return &SQLiteDB{
		DB: db,
	}
}
