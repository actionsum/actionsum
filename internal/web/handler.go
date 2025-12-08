package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"actionsum/internal/config"
	"actionsum/internal/database"
	"actionsum/internal/models"
	"actionsum/internal/reporter"
)

// Handler manages HTTP requests
type Handler struct {
	config   *config.Config
	repo     *database.Repository
	reporter *reporter.Reporter
}

// NewHandler creates a new web handler
func NewHandler(cfg *config.Config, repo *database.Repository) *Handler {
	return &Handler{
		config:   cfg,
		repo:     repo,
		reporter: reporter.New(cfg, repo),
	}
}

// SetupRoutes configures all HTTP routes
func (h *Handler) SetupRoutes(mux *http.ServeMux) {
	// API routes
	mux.HandleFunc("/api/events", h.handleEvents)
	mux.HandleFunc("/api/events/latest", h.handleLatestEvent)
	mux.HandleFunc("/api/report", h.handleReport)
	mux.HandleFunc("/api/summary", h.handleSummary)
	mux.HandleFunc("/api/status", h.handleStatus)

	// Health check
	mux.HandleFunc("/health", h.handleHealth)

	// Root
	mux.HandleFunc("/", h.handleIndex)
}

// handleEvents returns focus events with optional filtering
func (h *Handler) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	limitStr := query.Get("limit")
	periodType := query.Get("period") // day, week, month

	var events []*models.FocusEvent

	if periodType != "" {
		// Get events for a specific period
		period, err := h.getPeriod(periodType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		events, err = h.repo.GetEventsSince(period.Start)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to fetch events: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		// Get recent events (last 24 hours by default)
		start := time.Now().Add(-24 * time.Hour)
		allEvents, err := h.repo.GetEventsSince(start)
		if err == nil {
			// Apply limit in runtime
			limit := 100 // default
			if limitStr != "" {
				if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
					limit = l
				}
			}

			if len(allEvents) > limit {
				events = allEvents[len(allEvents)-limit:]
			} else {
				events = allEvents
			}
		}
	}

	respondJSON(w, events)
}

// handleLatestEvent returns the most recent focus event
func (h *Handler) handleLatestEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	event, err := h.repo.GetLatest()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch latest event: %v", err), http.StatusInternalServerError)
		return
	}

	if event == nil {
		http.Error(w, "No events found", http.StatusNotFound)
		return
	}

	respondJSON(w, event)
}

// handleReport generates a report for the specified period
func (h *Handler) handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	periodType := r.URL.Query().Get("period")
	if periodType == "" {
		periodType = "day"
	}

	report, err := h.reporter.GenerateReport(periodType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate report: %v", err), http.StatusInternalServerError)
		return
	}

	respondJSON(w, report)
}

// handleSummary returns aggregated app usage summary
func (h *Handler) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	periodType := r.URL.Query().Get("period")
	if periodType == "" {
		periodType = "day"
	}

	period, err := h.getPeriod(periodType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get raw summaries from database (SQL does the SUM)
	summaries, err := h.repo.GetAppSummarySince(period.Start)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get summary: %v", err), http.StatusInternalServerError)
		return
	}

	// Runtime calculates derived fields and totals
	var totalSeconds int64
	for i := range summaries {
		summaries[i].TotalMinutes = float64(summaries[i].TotalSeconds) / 60.0
		summaries[i].TotalHours = float64(summaries[i].TotalSeconds) / 3600.0
		totalSeconds += summaries[i].TotalSeconds
	}

	// Calculate percentages
	if totalSeconds > 0 {
		for i := range summaries {
			summaries[i].Percentage = (float64(summaries[i].TotalSeconds) / float64(totalSeconds)) * 100.0
		}
	}

	response := map[string]interface{}{
		"period":        period,
		"apps":          summaries,
		"total_seconds": totalSeconds,
		"total_minutes": float64(totalSeconds) / 60.0,
		"total_hours":   float64(totalSeconds) / 3600.0,
	}

	respondJSON(w, response)
}

// handleStatus returns current daemon status
func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	latestEvent, _ := h.repo.GetLatest()

	status := map[string]interface{}{
		"running":       true,
		"poll_interval": h.config.Tracker.PollInterval.String(),
		"database_path": h.config.Database.Path,
		"exclude_idle":  h.config.Report.ExcludeIdle,
	}

	if latestEvent != nil {
		status["latest_event"] = map[string]interface{}{
			"app_name":       latestEvent.AppName,
			"window_title":   latestEvent.WindowTitle,
			"timestamp":      latestEvent.Timestamp,
			"display_server": latestEvent.DisplayServer,
		}
	}

	respondJSON(w, status)
}

// handleHealth returns health check status
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleIndex returns basic API information
func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	info := map[string]interface{}{
		"name":    "actionsum API",
		"version": "0.1.0",
		"endpoints": []map[string]string{
			{"path": "/api/events", "description": "Get focus events (query: limit, period)"},
			{"path": "/api/events/latest", "description": "Get latest focus event"},
			{"path": "/api/report", "description": "Get time report (query: period=day|week|month)"},
			{"path": "/api/summary", "description": "Get app usage summary (query: period=day|week|month)"},
			{"path": "/api/status", "description": "Get daemon status"},
			{"path": "/health", "description": "Health check"},
		},
	}

	respondJSON(w, info)
}

// getPeriod calculates the time range for a period type
func (h *Handler) getPeriod(periodType string) (*models.ReportPeriod, error) {
	now := time.Now()
	var start, end time.Time

	switch periodType {
	case "day", "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.Add(24 * time.Hour)
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(weekday - 1))
		end = start.AddDate(0, 0, 7)
	case "month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)
	default:
		return nil, fmt.Errorf("invalid period type: %s", periodType)
	}

	return &models.ReportPeriod{
		Start: start,
		End:   end,
		Type:  periodType,
	}, nil
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
