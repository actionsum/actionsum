package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/actionsum/actionsum/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	defaultDBName = "actionsum.db"
	defaultDBDir  = ".config/actionsum"
)

type DB struct {
	*gorm.DB
}

func GetDefaultDBPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	dbDir := filepath.Join(homeDir, defaultDBDir)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create database directory: %w", err)
	}

	return filepath.Join(dbDir, defaultDBName), nil
}

func Connect(dbPath string) (*DB, error) {
	if dbPath == "" {
		var err error
		dbPath, err = GetDefaultDBPath()
		if err != nil {
			return nil, err
		}
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Initialize() error {
	err := db.AutoMigrate(&models.FocusEvent{}, &models.ErrorLog{})
	if err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return nil
}

func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
}
