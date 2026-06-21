package models

import (
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	dataDir := filepath.Join(".", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}
	dbPath := filepath.Join(dataDir, "repos.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}
	DB = db
	if err := DB.AutoMigrate(&Repo{}); err != nil {
		return err
	}
	log.Printf("Database initialized at %s", dbPath)
	return nil
}
