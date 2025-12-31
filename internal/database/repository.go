package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/actionsum/actionsum/internal/models"

	"github.com/pkg/errors"

	"gorm.io/gorm"
)

type Repository struct {
	db *DB
}

func NewRepository(db *DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(event *models.FocusEvent) error {
	event.AppName = strings.ToLower(event.AppName)
	result := r.db.Create(event)
	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to insert focus event")
	}
	return nil
}

func (r *Repository) GetByID(id uint) (*models.FocusEvent, error) {
	var event models.FocusEvent
	result := r.db.First(&event, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, errors.Wrap(result.Error, "failed to get focus event")
	}
	return &event, nil
}

func (r *Repository) GetEventsSince(since time.Time) ([]*models.FocusEvent, error) {
	var events []*models.FocusEvent
	result := r.db.Where("timestamp >= ?", since).Order("timestamp ASC").Find(&events)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, "failed to query focus events")
	}

	return events, nil
}

func (r *Repository) GetAppSummarySince(since time.Time) ([]models.AppSummary, error) {
	var summaries []models.AppSummary

	result := r.db.Model(&models.FocusEvent{}).
		Select("app_name, SUM(duration) as total_seconds, COUNT(*) as event_count").
		Where("timestamp >= ?", since).
		Group("app_name").
		Order("total_seconds DESC").
		Scan(&summaries)

	if result.Error != nil {
		return nil, errors.Wrap(result.Error, "failed to query app summary")
	}

	return summaries, nil
}

func (r *Repository) DeleteOldEvents(before time.Time) (int64, error) {
	result := r.db.Where("timestamp < ?", before).Delete(&models.FocusEvent{})
	if result.Error != nil {
		return 0, errors.Wrap(result.Error, "failed to delete old events")
	}
	return result.RowsAffected, nil
}

func (r *Repository) GetLatest() (*models.FocusEvent, error) {
	var event models.FocusEvent
	result := r.db.Order("timestamp DESC").First(&event)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.Wrap(result.Error, "failed to get latest event")
	}
	return &event, nil
}

func (r *Repository) Update(event *models.FocusEvent) error {
	event.AppName = strings.ToLower(event.AppName)
	result := r.db.Save(event)
	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to update event")
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("event not found")
	}
	return nil
}

func (r *Repository) UpdateDuration(id uint, duration int64) error {
	result := r.db.Model(&models.FocusEvent{}).Where("id = ?", id).Update("duration", duration)
	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to update event duration")
	}
	return nil
}

func (r *Repository) CreateErrorLog(errorLog *models.ErrorLog) error {
	result := r.db.Create(errorLog)
	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to insert error log")
	}
	return nil
}

func (r *Repository) Clear() error {
	result := r.db.Exec("DELETE FROM focus_events")
	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to clear focus events")
	}
	return nil
}

// NormalizeAppNames updates all app_name values to lowercase
func (r *Repository) NormalizeAppNames() (int64, error) {
	result := r.db.Exec("UPDATE focus_events SET app_name = LOWER(app_name)")
	if result.Error != nil {
		return 0, errors.Wrap(result.Error, "failed to normalize app names")
	}
	return result.RowsAffected, nil
}
