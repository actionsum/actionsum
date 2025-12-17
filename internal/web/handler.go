package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/actionsum/actionsum/internal/config"
	"github.com/actionsum/actionsum/internal/database"
	"github.com/actionsum/actionsum/internal/models"
	"github.com/actionsum/actionsum/internal/reporter"
	"github.com/actionsum/actionsum/pkg/utils"
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

	// Check if request is from HTMX
	if r.Header.Get("HX-Request") == "true" {
		h.respondSummaryHTML(w, summaries, totalSeconds)
		return
	}

	// Default JSON response
	response := map[string]interface{}{
		"period":        period,
		"apps":          summaries,
		"total_seconds": totalSeconds,
		"total_minutes": float64(totalSeconds) / 60.0,
		"total_hours":   float64(totalSeconds) / 3600.0,
	}

	respondJSON(w, response)
}

// respondSummaryHTML renders summary data as HTML
func (h *Handler) respondSummaryHTML(w http.ResponseWriter, summaries []models.AppSummary, totalSeconds int64) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if len(summaries) == 0 {
		w.Write([]byte(`<div class="loading">No data available</div>`))
		return
	}

	html := ""
	for _, app := range summaries {
		// Show the highest round unit (h, m, or s)
		timeStr := utils.FormatRoundedUnit(app.TotalSeconds)

		// Format percentage with padding for XX.X% format
		percentStr := fmt.Sprintf("%.1f%%", app.Percentage)
		if app.Percentage < 10 {
			percentStr = "&nbsp;&nbsp;" + percentStr
		} else if app.Percentage < 100 {
			percentStr = "&nbsp;" + percentStr
		}

		html += fmt.Sprintf(`
		<div class="app-item">
			<span class="app-name">%s</span>
			<div>
				<span class="app-time">%s</span>
				<span class="app-percentage">%s</span>
			</div>
		</div>`, app.AppName, timeStr, percentStr)
	}

	totalStr := utils.FormatRoundedUnit(totalSeconds)

	html += fmt.Sprintf(`<div class="total">Total: %s</div>`, totalStr)

	w.Write([]byte(html))
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

// handleIndex serves the main dashboard page
func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Actionsum Dashboard</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        :root {
            --bg-primary: #f5f5f5;
            --bg-secondary: white;
            --text-primary: #333;
            --text-secondary: #1a1a1a;
            --text-muted: #7f8c8d;
            --border-color: #eee;
            --border-strong: #ecf0f1;
            --accent-color: #3498db;
            --heading-color: #2c3e50;
            --shadow: rgba(0,0,0,0.1);
        }
        
        [data-theme="dark"] {
            --bg-primary: #1a1a1a;
            --bg-secondary: #2d2d2d;
            --text-primary: #e0e0e0;
            --text-secondary: #ffffff;
            --text-muted: #a0a0a0;
            --border-color: #404040;
            --border-strong: #4a4a4a;
            --accent-color: #5dade2;
            --heading-color: #5dade2;
            --shadow: rgba(0,0,0,0.3);
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: var(--bg-primary);
            padding: 20px;
            color: var(--text-primary);
            transition: background-color 0.3s ease, color 0.3s ease;
        }
        
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 30px;
        }
        
        h1 {
            color: var(--text-secondary);
            font-size: 2rem;
            margin: 0;
        }
        
        .theme-toggle {
            background: var(--bg-secondary);
            border: 2px solid var(--border-color);
            border-radius: 50px;
            padding: 8px 16px;
            cursor: pointer;
            font-size: 1.2rem;
            transition: all 0.3s ease;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        .theme-toggle:hover {
            border-color: var(--accent-color);
            transform: scale(1.05);
        }
        
        .dashboard {
            display: flex;
            gap: 20px;
            flex-wrap: wrap;
        }
        
        .report-box {
            flex: 1;
            min-width: 300px;
            background: var(--bg-secondary);
            border-radius: 8px;
            box-shadow: 0 2px 4px var(--shadow);
            padding: 24px;
            transition: background-color 0.3s ease, box-shadow 0.3s ease;
        }
        
        .report-box h2 {
            font-size: 1.5rem;
            margin-bottom: 20px;
            color: var(--heading-color);
            border-bottom: 2px solid var(--accent-color);
            padding-bottom: 10px;
        }
        
        .app-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 12px 0;
            border-bottom: 1px solid var(--border-color);
        }
        
        .app-item:last-child {
            border-bottom: none;
        }
        
        .app-name {
            font-weight: 500;
            color: var(--text-primary);
        }
        
        .app-time {
            color: var(--text-muted);
            font-size: 0.9rem;
        }
        
        .app-percentage {
            color: var(--accent-color);
            font-weight: 600;
            margin-left: 10px;
            display: inline-block;
            min-width: 5em;
            text-align: right;
			margin: 1px
        }
        
        .loading {
            color: var(--text-muted);
            font-style: italic;
        }
        
        .total {
            margin-top: 20px;
            padding-top: 15px;
            border-top: 2px solid var(--border-strong);
            font-weight: 600;
            font-size: 1.1rem;
            color: var(--heading-color);
        }
        
        @media (max-width: 1024px) {
            .dashboard {
                flex-direction: column;
            }
            
            .report-box {
                min-width: 100%;
            }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>Actionsum Dashboard</h1>
        <button class="theme-toggle" onclick="toggleTheme()">
            <span id="theme-icon">🌙</span>
        </button>
    </div>
    <div class="dashboard">
        <div class="report-box">
            <h2>Today</h2>
            <div hx-get="/api/summary?period=today" hx-trigger="load, every 30s" hx-swap="innerHTML">
                <div class="loading">Loading...</div>
            </div>
        </div>
        
        <div class="report-box">
            <h2>This Week</h2>
            <div hx-get="/api/summary?period=week" hx-trigger="load, every 30s" hx-swap="innerHTML">
                <div class="loading">Loading...</div>
            </div>
        </div>
        
        <div class="report-box">
            <h2>This Month</h2>
            <div hx-get="/api/summary?period=month" hx-trigger="load, every 30s" hx-swap="innerHTML">
                <div class="loading">Loading...</div>
            </div>
        </div>
    </div>
    <script>
        // Initialize theme from localStorage or system preference
        function initTheme() {
            const savedTheme = localStorage.getItem('theme');
            const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
            const theme = savedTheme || (prefersDark ? 'dark' : 'light');
            setTheme(theme);
        }
        
        function setTheme(theme) {
            document.documentElement.setAttribute('data-theme', theme);
            document.getElementById('theme-icon').textContent = theme === 'dark' ? '☀️' : '🌙';
            localStorage.setItem('theme', theme);
        }
        
        function toggleTheme() {
            const currentTheme = document.documentElement.getAttribute('data-theme');
            const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
            setTheme(newTheme);
        }
        
        // Initialize on page load
        initTheme();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
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
