package tracker

import (
	"context"
	"fmt"
	"log"
	"time"

	"actionsum/internal/config"
	"actionsum/internal/database"
	"actionsum/internal/models"
	"actionsum/pkg/window"
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

	if err := s.trackOnce(); err != nil {
		log.Printf("Warning: initial tracking failed: %v", err)
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
			if err := s.trackOnce(); err != nil {
				log.Printf("Tracking error: %v", err)
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

func (s *Service) trackOnce() error {

	idleInfo, err := s.detector.GetIdleInfo()
	if err != nil {

		log.Printf("Warning: failed to get idle info: %v", err)
		idleInfo = &window.IdleInfo{
			IsIdle:   false,
			IsLocked: false,
			IdleTime: 0,
		}
	}

	if idleInfo.IsIdle || idleInfo.IsLocked {
		log.Printf("Skipping tracking: idle=%v, locked=%v", idleInfo.IsIdle, idleInfo.IsLocked)
		return nil
	}

	windowInfo, err := s.detector.GetFocusedWindow()
	if err != nil {

		log.Printf("Warning: failed to get focused window: %v", err)
		return nil
	}

	if windowInfo == nil || windowInfo.AppName == "" {
		log.Printf("Warning: no valid window information available")
		return nil
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
		return fmt.Errorf("failed to save event: %w", err)
	}

	log.Printf("Tracked: %s - %s", event.AppName, event.WindowTitle)
	return nil
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
