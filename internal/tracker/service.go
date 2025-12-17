package tracker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hugo/actionsum/internal/config"
	"github.com/hugo/actionsum/internal/database"
	"github.com/hugo/actionsum/internal/models"
	"github.com/hugo/actionsum/pkg/window"
)

type Service struct {
	config   *config.Config
	repo     *database.Repository
	detector window.Detector
	stopChan chan struct{}
	running  bool
}

func NewService(cfg *config.Config, repo *database.Repository, detector window.Detector) *Service {
	return &Service{
		config:   cfg,
		repo:     repo,
		detector: detector,
		stopChan: make(chan struct{}),
		running:  false,
	}
}

func (s *Service) Start(ctx context.Context) error {
	if s.running {
		return fmt.Errorf("tracker is already running")
	}

	s.running = true
	log.Printf("Starting tracker with %v poll interval", s.config.Tracker.PollInterval)

	ticker := time.NewTicker(s.config.Tracker.PollInterval)
	defer ticker.Stop()

	appName, isIdle, isLocked, err := s.trackOnce()
	if err != nil {
		s.storeError(err)
	}
	if appName != "" {
		log.Printf("Initial track: %s (idle: %v, locked: %v)", appName, isIdle, isLocked)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Tracker stopped by context")
			s.running = false
			return ctx.Err()

		case <-s.stopChan:
			log.Println("Tracker stopped")
			s.running = false
			return nil

		case <-ticker.C:
			appName, isIdle, isLocked, err := s.trackOnce()
			if err != nil {
				s.storeError(err)
			}
			if appName != "" {
				log.Printf("Tracked: %s (idle: %v, locked: %v)", appName, isIdle, isLocked)
			}
		}
	}
}

func (s *Service) Stop() {
	if s.running {
		close(s.stopChan)
	}
}

func (s *Service) IsRunning() bool {
	return s.running
}

func (s *Service) trackOnce() (string, bool, bool, error) {

	idleInfo, err := s.detector.GetIdleInfo()
	if err != nil {
		return "", false, false, fmt.Errorf("failed to get idle info: %w", err)
	}

	if idleInfo.IsIdle || idleInfo.IsLocked {
		log.Printf("Skipping tracking: idle=%v, locked=%v", idleInfo.IsIdle, idleInfo.IsLocked)
		return "", idleInfo.IsIdle, idleInfo.IsLocked, nil
	}

	windowInfo, err := s.detector.GetFocusedWindow()
	if err != nil {
		return "", idleInfo.IsIdle, idleInfo.IsLocked, fmt.Errorf("failed to get focused window: %w", err)
	}

	if windowInfo == nil || windowInfo.AppName == "" {
		return "", idleInfo.IsIdle, idleInfo.IsLocked, fmt.Errorf("no valid window information available")
	}

	event := &models.FocusEvent{
		Timestamp:     time.Now(),
		AppName:       windowInfo.AppName,
		WindowTitle:   windowInfo.WindowTitle,
		Duration:      s.config.GetPollIntervalSeconds(),
		IsIdle:        idleInfo.IsIdle,
		IsLocked:      idleInfo.IsLocked,
		DisplayServer: windowInfo.DisplayServer,
		CreatedAt:     time.Now(),
	}

	if err := s.repo.Create(event); err != nil {
		return "", idleInfo.IsIdle, idleInfo.IsLocked, fmt.Errorf("failed to save event: %w", err)
	}

	return event.AppName, idleInfo.IsIdle, idleInfo.IsLocked, nil
}

func (s *Service) storeError(err error) {
	errorLog := &models.ErrorLog{
		Timestamp: time.Now(),
		ErrorMsg:  err.Error(),
		CreatedAt: time.Now(),
	}

	if dbErr := s.repo.CreateErrorLog(errorLog); dbErr != nil {
		log.Printf("Failed to store error in database: %v (original error: %v)", dbErr, err)
	} else {
		log.Printf("Error logged to database: %v", err)
	}
}

func (s *Service) IsScreenLocked() (bool, error) {
	idleInfo, err := s.detector.GetIdleInfo()
	if err != nil {
		return false, fmt.Errorf("failed to get idle info: %w", err)
	}

	return idleInfo.IsLocked, nil
}

func (s *Service) GetCurrentWindow() (*window.WindowInfo, *window.IdleInfo, error) {
	windowInfo, err := s.detector.GetFocusedWindow()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get focused window: %w", err)
	}

	idleInfo, err := s.detector.GetIdleInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get idle info: %w", err)
	}

	return windowInfo, idleInfo, nil
}
