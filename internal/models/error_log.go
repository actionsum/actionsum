package models

import (
	"time"

	"gorm.io/gorm"
)

type ErrorLog struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Timestamp time.Time      `gorm:"not null;index" json:"timestamp"`
	ErrorMsg  string         `gorm:"not null" json:"error_msg"`
	CreatedAt time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
