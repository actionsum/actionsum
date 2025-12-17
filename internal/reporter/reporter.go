package reporter

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hugo/actionsum/internal/config"
	"github.com/hugo/actionsum/internal/database"
	"github.com/hugo/actionsum/internal/models"
)

// Reporter handles report generation
type Reporter struct {
	config *config.Config
	repo   *database.Repository
}

// New creates a new reporter
func New(cfg *config.Config, repo *database.Repository) *Reporter {
	return &Reporter{
		config: cfg,
		repo:   repo,
	}
}

// GenerateReport generates a report for the specified period
func (r *Reporter) GenerateReport(periodType string) (*models.Report, error) {
	period, err := r.getPeriod(periodType)
	if err != nil {
		return nil, err
	}

	// Get raw summaries from database (SQL does the SUM)
	summaries, err := r.repo.GetAppSummarySince(period.Start)
	if err != nil {
		return nil, fmt.Errorf("failed to get app summary: %w", err)
	}

	// Runtime calculates derived fields and percentages
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

	report := &models.Report{
		Period:       *period,
		Apps:         summaries,
		TotalSeconds: totalSeconds,
		TotalMinutes: float64(totalSeconds) / 60.0,
		TotalHours:   float64(totalSeconds) / 3600.0,
		GeneratedAt:  time.Now(),
	}

	return report, nil
}

// getPeriod calculates the time range for the report
func (r *Reporter) getPeriod(periodType string) (*models.ReportPeriod, error) {
	now := time.Now()
	var start, end time.Time

	switch periodType {
	case "day", "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.Add(24 * time.Hour)

	case "week":
		// Start of week (Monday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7
		}
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(weekday - 1))
		end = start.AddDate(0, 0, 7)

	case "month":
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0)

	default:
		return nil, fmt.Errorf("invalid period type: %s (valid: day, week, month)", periodType)
	}

	return &models.ReportPeriod{
		Start: start,
		End:   end,
		Type:  periodType,
	}, nil
}

// FormatReportText formats the report as human-readable text
func (r *Reporter) FormatReportText(report *models.Report) string {
	output := fmt.Sprintf("Activity Report - %s\n", report.Period.Type)
	output += fmt.Sprintf("Period: %s to %s\n",
		report.Period.Start.Format("2006-01-02 15:04"),
		report.Period.End.Format("2006-01-02 15:04"))
	output += fmt.Sprintf("Total Time: %.2fh (%.0fm)\n\n", report.TotalHours, report.TotalMinutes)

	if len(report.Apps) == 0 {
		output += "No activity recorded for this period.\n"
		return output
	}

	output += fmt.Sprintf("%-30s %10s %10s %10s\n", "Application", "Hours", "Minutes", "Percent")
	output += fmt.Sprintf("%s\n", "--------------------------------------------------------------------------------")

	for _, app := range report.Apps {
		output += fmt.Sprintf("%-30s %10.2f %10.0f %9.1f%%\n",
			truncate(app.AppName, 30),
			app.TotalHours,
			app.TotalMinutes,
			app.Percentage)
	}

	return output
}

// FormatReportJSON formats the report as JSON
func (r *Reporter) FormatReportJSON(report *models.Report) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
