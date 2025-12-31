package models

import (
	"time"

	"gorm.io/gorm"
)

type FocusEvent struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Timestamp     time.Time      `gorm:"not null;index" json:"timestamp"`
	AppName       string         `gorm:"not null;index" json:"app_name"`
	WindowTitle   string         `gorm:"not null" json:"window_title"`
	Duration      int64          `gorm:"not null;default:0" json:"duration"` // Duration in seconds
	IsIdle        bool           `gorm:"not null;default:false" json:"is_idle"`
	IsLocked      bool           `gorm:"not null;default:false" json:"is_locked"`
	DisplayServer string         `gorm:"not null" json:"display_server"` // "x11" or "wayland"
	CreatedAt     time.Time      `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type AppSummary struct {
	AppName      string  `json:"app_name"`
	TotalSeconds int64   `json:"total_seconds"`
	TotalMinutes float64 `json:"total_minutes"`
	TotalHours   float64 `json:"total_hours"`
	EventCount   int     `json:"event_count"`
	Percentage   float64 `json:"percentage,omitempty"`
}

type ReportPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Type  string    `json:"type"` // "day", "week", "month"
}

type Report struct {
	Period       ReportPeriod `json:"period"`
	Apps         []AppSummary `json:"apps"`
	TotalSeconds int64        `json:"total_seconds"`
	TotalMinutes float64      `json:"total_minutes"`
	TotalHours   float64      `json:"total_hours"`
	GeneratedAt  time.Time    `json:"generated_at"`
}
