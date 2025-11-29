// ABOUTME: TUI view for Google sync status and controls
// ABOUTME: Displays sync states and allows triggering syncs for contacts, calendar, gmail
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/sync"
)

var (
	syncTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			MarginBottom(1)

	syncHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Underline(true)

	syncServiceStyle = lipgloss.NewStyle().
				Bold(true).
				Width(12)

	syncIdleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	syncSyncingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")).
				Bold(true)

	syncErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	syncSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("235")).
				Foreground(lipgloss.Color("255")).
				Bold(true)

	syncMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Italic(true)
)

// SyncCompleteMsg is sent when a sync operation completes
type SyncCompleteMsg struct {
	Service string
	Error   error
}

// SyncStartMsg is sent when a sync operation starts
type SyncStartMsg struct {
	Service string
}

// SyncStatusUpdateMsg is sent to refresh sync status
type SyncStatusUpdateMsg struct{}

func (m Model) renderSyncView() string {
	var s strings.Builder

	// Title
	s.WriteString(syncTitleStyle.Render("Google Sync Management"))
	s.WriteString("\n\n")

	// Refresh sync states
	m.loadSyncStates()

	// If no sync states, show initialization message
	if len(m.syncStates) == 0 {
		s.WriteString(syncMessageStyle.Render("No sync data found. Run sync initialization first."))
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("Press 'i' to initialize sync, 'Esc' to go back"))
		return s.String()
	}

	// Header
	s.WriteString(syncHeaderStyle.Render("Service Status"))
	s.WriteString("\n\n")

	// Service status table
	services := []string{"calendar", "contacts", "gmail"}
	for i, service := range services {
		// Find state for this service
		var state *SyncStateDisplay
		for j := range m.syncStates {
			if m.syncStates[j].Service == service {
				state = &m.syncStates[j]
				break
			}
		}

		// Build row
		var row strings.Builder

		// Selection indicator
		if i == m.selectedService {
			row.WriteString("▶ ")
		} else {
			row.WriteString("  ")
		}

		// Service name
		serviceName := strings.ToUpper(service[:1]) + service[1:]
		if i == m.selectedService {
			row.WriteString(syncSelectedStyle.Render(syncServiceStyle.Render(serviceName)))
		} else {
			row.WriteString(syncServiceStyle.Render(serviceName))
		}

		// Status
		if state == nil {
			row.WriteString(syncMessageStyle.Render("  Not synced yet"))
		} else if state.InProgress || m.syncInProgress[service] {
			row.WriteString(syncSyncingStyle.Render("  ⟳ Syncing..."))
		} else if state.Status == "error" {
			row.WriteString(syncErrorStyle.Render("  ✗ Error"))
			if state.ErrorMessage != "" {
				row.WriteString(syncErrorStyle.Render(": " + state.ErrorMessage))
			}
		} else {
			row.WriteString(syncIdleStyle.Render("  ✓ Idle"))
			if state.LastSyncTime != "" {
				row.WriteString(syncMessageStyle.Render(" • Last synced " + state.LastSyncTime))
			}
		}

		s.WriteString(row.String())
		s.WriteString("\n")
	}

	s.WriteString("\n")

	// Recent messages
	if len(m.syncMessages) > 0 {
		s.WriteString(syncHeaderStyle.Render("Recent Activity"))
		s.WriteString("\n\n")
		// Show last 5 messages
		start := 0
		if len(m.syncMessages) > 5 {
			start = len(m.syncMessages) - 5
		}
		for i := start; i < len(m.syncMessages); i++ {
			s.WriteString(syncMessageStyle.Render("  " + m.syncMessages[i]))
			s.WriteString("\n")
		}
		s.WriteString("\n")
	}

	// Help
	s.WriteString(m.renderSyncHelp())

	return s.String()
}

func (m Model) renderSyncHelp() string {
	help := []string{
		"↑/↓: Select service",
		"Enter: Sync selected",
		"a: Sync all",
		"r: Refresh status",
		"Esc: Back",
		"q: Quit",
	}
	return helpStyle.Render(strings.Join(help, " • "))
}

func (m *Model) loadSyncStates() {
	states, err := db.GetAllSyncStates(m.db)
	if err != nil {
		m.syncStates = []SyncStateDisplay{}
		return
	}

	m.syncStates = []SyncStateDisplay{}
	for _, state := range states {
		display := SyncStateDisplay{
			Service:    state.Service,
			Status:     state.Status,
			InProgress: m.syncInProgress[state.Service],
		}

		if state.LastSyncTime != nil {
			display.LastSyncTime = formatTimeSince(*state.LastSyncTime)
		}

		if state.ErrorMessage != nil {
			display.ErrorMessage = *state.ErrorMessage
		}

		m.syncStates = append(m.syncStates, display)
	}
}

func (m Model) handleSyncKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selectedService > 0 {
			m.selectedService--
		}
	case "down", "j":
		if m.selectedService < 2 { // 3 services (0-2)
			m.selectedService++
		}
	case "enter":
		// Sync selected service
		services := []string{"calendar", "contacts", "gmail"}
		if m.selectedService < len(services) {
			service := services[m.selectedService]
			// Send start message immediately, then queue the async sync
			m.syncInProgress[service] = true
			m.addSyncMessage(fmt.Sprintf("Starting %s sync...", service))
			return m, m.syncService(service)
		}
	case "a":
		// Sync all services
		services := []string{"calendar", "contacts", "gmail"}
		for _, service := range services {
			m.syncInProgress[service] = true
			m.addSyncMessage(fmt.Sprintf("Starting %s sync...", service))
		}
		return m, m.syncAllServices()
	case "r":
		// Refresh sync status
		m.loadSyncStates()
	case "esc":
		// Go back to main view
		m.viewMode = ViewList
		m.entityType = EntityContacts
	}

	return m, nil
}

// syncService triggers a sync for a specific service
func (m Model) syncService(service string) tea.Cmd {
	return func() tea.Msg {
		// Send start message first
		// (state updates happen in Update() when handling SyncStartMsg)

		// Update database status
		_ = db.UpdateSyncStatus(m.db, service, "syncing", nil)

		// Load OAuth token
		token, err := sync.LoadToken()
		if err != nil {
			errMsg := fmt.Sprintf("Authentication failed: %v", err)
			_ = db.UpdateSyncStatus(m.db, service, "error", &errMsg)
			return SyncCompleteMsg{Service: service, Error: err}
		}

		// Perform sync based on service
		var syncErr error
		switch service {
		case "contacts":
			client, err := sync.NewPeopleClient(token)
			if err != nil {
				syncErr = fmt.Errorf("failed to create People API client: %w", err)
			} else {
				syncErr = sync.ImportContacts(m.db, client)
			}
		case "calendar":
			client, err := sync.NewCalendarClient(token)
			if err != nil {
				syncErr = fmt.Errorf("failed to create Calendar client: %w", err)
			} else {
				syncErr = sync.ImportCalendar(m.db, client, false)
			}
		case "gmail":
			client, err := sync.NewGmailClient(token)
			if err != nil {
				syncErr = fmt.Errorf("failed to create Gmail client: %w", err)
			} else {
				syncErr = sync.ImportGmail(m.db, client, false)
			}
		}

		return SyncCompleteMsg{Service: service, Error: syncErr}
	}
}

// syncAllServices triggers a sync for all services
func (m *Model) syncAllServices() tea.Cmd {
	return tea.Batch(
		m.syncService("calendar"),
		m.syncService("contacts"),
		m.syncService("gmail"),
	)
}

// addSyncMessage adds a message to the sync message log
func (m *Model) addSyncMessage(msg string) {
	timestamp := time.Now().Format("15:04:05")
	m.syncMessages = append(m.syncMessages, fmt.Sprintf("[%s] %s", timestamp, msg))
}

// handleSyncComplete handles sync completion messages
func (m *Model) handleSyncComplete(msg SyncCompleteMsg) tea.Cmd {
	// Mark as no longer in progress
	m.syncInProgress[msg.Service] = false

	if msg.Error != nil {
		m.addSyncMessage(fmt.Sprintf("✗ %s sync failed: %v", msg.Service, msg.Error))
		errMsg := msg.Error.Error()
		_ = db.UpdateSyncStatus(m.db, msg.Service, "error", &errMsg)
	} else {
		m.addSyncMessage(fmt.Sprintf("✓ %s sync completed", msg.Service))
	}

	// Reload sync states
	m.loadSyncStates()

	return nil
}

// formatTimeSince formats a time duration in a human-readable way
func formatTimeSince(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
