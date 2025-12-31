package reporter

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/actionsum/actionsum/internal/config"
	"github.com/actionsum/actionsum/internal/database"
	"github.com/actionsum/actionsum/internal/models"
	"github.com/actionsum/actionsum/pkg/utils"
)

type Reporter struct {
	config *config.Config
	repo   *database.Repository
}

func New(cfg *config.Config, repo *database.Repository) *Reporter {
	return &Reporter{
		config: cfg,
		repo:   repo,
	}
}

func (r *Reporter) GenerateReport(periodType string) (*models.Report, error) {
	period, err := r.getPeriod(periodType)
	if err != nil {
		return nil, err
	}

	summaries, err := r.repo.GetAppSummarySince(period.Start)
	if err != nil {
		return nil, fmt.Errorf("failed to get app summary: %w", err)
	}

	var totalSeconds int64
	for i := range summaries {
		summaries[i].TotalMinutes = float64(summaries[i].TotalSeconds) / 60.0
		summaries[i].TotalHours = float64(summaries[i].TotalSeconds) / 3600.0
		totalSeconds += summaries[i].TotalSeconds
	}

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

func (r *Reporter) getPeriod(periodType string) (*models.ReportPeriod, error) {
	now := time.Now()
	var start, end time.Time

	switch periodType {
	case "day", "today":
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end = start.Add(24 * time.Hour)

	case "week":
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

func (r *Reporter) FormatReportText(report *models.Report) string {
	output := fmt.Sprintf("Activity Report - %s\n", report.Period.Type)
	output += fmt.Sprintf("Period: %s to %s\n",
		report.Period.Start.Format("2006-01-02 15:04"),
		report.Period.End.Format("2006-01-02 15:04"))
	output += fmt.Sprintf("Total Time: %s\n\n", utils.FormatRoundedUnit(report.TotalSeconds))

	if len(report.Apps) == 0 {
		output += "No activity recorded for this period.\n"
		return output
	}

	output += fmt.Sprintf("%-30s %10s %10s %10s\n", "Application", "Hours", "Time", "Percent")
	output += fmt.Sprintf("%s\n", "--------------------------------------------------------------------------------")

	for _, app := range report.Apps {
		timeStr := utils.FormatRoundedUnit(app.TotalSeconds)

		output += fmt.Sprintf("%-30s %10.2f %10s %9.1f%%\n",
			truncate(app.AppName, 30),
			app.TotalHours,
			timeStr,
			app.Percentage)
	}

	return output
}

func (r *Reporter) FormatReportJSON(report *models.Report) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
