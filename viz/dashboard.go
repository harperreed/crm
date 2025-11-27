// ABOUTME: Terminal dashboard statistics and rendering
// ABOUTME: Provides ASCII dashboard for CRM overview
package viz

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
)

type DashboardStats struct {
	// Pipeline overview
	PipelineByStage map[string]PipelineStageStats

	// Overall stats
	TotalContacts  int
	TotalCompanies int
	TotalDeals     int

	// Recent activity (last 7 days)
	RecentActivity []ActivityItem

	// Needs attention
	StaleContacts []StaleContact
	StaleDeals    []StaleDeal
}

type PipelineStageStats struct {
	Stage  string
	Count  int
	Amount int64 // in cents
}

type ActivityItem struct {
	Date        time.Time
	Description string
}

type StaleContact struct {
	Name      string
	DaysSince int
}

type StaleDeal struct {
	Title     string
	DaysSince int
}

func GenerateDashboardStats(database *sql.DB) (*DashboardStats, error) {
	stats := &DashboardStats{
		PipelineByStage: make(map[string]PipelineStageStats),
	}

	// Get pipeline stats
	deals, err := db.FindDeals(database, "", nil, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deals: %w", err)
	}

	for _, deal := range deals {
		stage := deal.Stage
		if stage == "" {
			stage = "unknown"
		}

		pstats := stats.PipelineByStage[stage]
		pstats.Stage = stage
		pstats.Count++
		pstats.Amount += deal.Amount
		stats.PipelineByStage[stage] = pstats
	}

	stats.TotalDeals = len(deals)

	// Get contact stats
	contacts, err := db.FindContacts(database, "", nil, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contacts: %w", err)
	}
	stats.TotalContacts = len(contacts)

	// Get company stats
	companies, err := db.FindCompanies(database, "", 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch companies: %w", err)
	}
	stats.TotalCompanies = len(companies)

	// Find stale contacts (no contact in 30+ days)
	now := time.Now()
	for _, contact := range contacts {
		if contact.LastContactedAt == nil {
			stats.StaleContacts = append(stats.StaleContacts, StaleContact{
				Name:      contact.Name,
				DaysSince: -1, // Never contacted
			})
		} else {
			daysSince := int(now.Sub(*contact.LastContactedAt).Hours() / 24)
			if daysSince > 30 {
				stats.StaleContacts = append(stats.StaleContacts, StaleContact{
					Name:      contact.Name,
					DaysSince: daysSince,
				})
			}
		}
	}

	// Find stale deals (no activity in 14+ days)
	for _, deal := range deals {
		daysSince := int(now.Sub(deal.LastActivityAt).Hours() / 24)
		if daysSince > 14 {
			stats.StaleDeals = append(stats.StaleDeals, StaleDeal{
				Title:     deal.Title,
				DaysSince: daysSince,
			})
		}
	}

	return stats, nil
}

func RenderDashboard(stats *DashboardStats) string {
	var out strings.Builder

	// Header
	out.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	out.WriteString("  PAGEN CRM DASHBOARD\n")
	out.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Pipeline overview
	out.WriteString("PIPELINE OVERVIEW\n")
	renderPipeline(&out, stats.PipelineByStage)
	out.WriteString("\n")

	// Stats
	out.WriteString("STATS\n")
	out.WriteString(fmt.Sprintf("  📇 %d contacts  🏢 %d companies  💼 %d deals\n\n",
		stats.TotalContacts, stats.TotalCompanies, stats.TotalDeals))

	// Needs attention
	if len(stats.StaleContacts) > 0 || len(stats.StaleDeals) > 0 {
		out.WriteString("NEEDS ATTENTION\n")

		if len(stats.StaleContacts) > 0 {
			out.WriteString(fmt.Sprintf("  ⚠️  %d contacts - no contact in 30+ days\n", len(stats.StaleContacts)))
		}

		if len(stats.StaleDeals) > 0 {
			out.WriteString(fmt.Sprintf("  ⚠️  %d deals - stale (no activity in 14+ days)\n", len(stats.StaleDeals)))
		}
	}

	return out.String()
}

func renderPipeline(out *strings.Builder, pipeline map[string]PipelineStageStats) {
	// Define stage order
	stages := []string{
		models.StageProspecting,
		models.StageQualification,
		models.StageProposal,
		models.StageNegotiation,
		models.StageClosedWon,
		models.StageClosedLost,
	}

	// Find max count for scaling
	maxCount := 0
	for _, pstats := range pipeline {
		if pstats.Count > maxCount {
			maxCount = pstats.Count
		}
	}
	if maxCount == 0 {
		maxCount = 1
	}

	// Render each stage
	for _, stage := range stages {
		pstats, exists := pipeline[stage]
		if !exists {
			continue
		}

		// Calculate bar length (0-10 blocks)
		barLength := (pstats.Count * 10) / maxCount

		// Build bar
		bar := strings.Repeat("█", barLength) + strings.Repeat("░", 10-barLength)

		// Format amount in K
		amountK := pstats.Amount / 100000

		fmt.Fprintf(out, "  %-13s %s  %2d ($%dK)\n",
			stage, bar, pstats.Count, amountK)
	}
}
