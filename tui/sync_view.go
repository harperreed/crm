// ABOUTME: TUI view for Charm KV sync status and controls
// ABOUTME: Displays sync configuration and allows triggering manual syncs
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	syncLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Width(15)

	syncValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	syncEnabledStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10")).
				Bold(true)

	syncDisabledStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("9"))

	syncMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Italic(true)

	syncSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10"))

	syncErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	syncSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("235")).
				Foreground(lipgloss.Color("255")).
				Bold(true)
)

// SyncCompleteMsg is sent when a sync operation completes.
type SyncCompleteMsg struct {
	Error error
}

// AutoSyncToggleMsg is sent when auto-sync is toggled.
type AutoSyncToggleMsg struct {
	Enabled bool
	Error   error
}

func (m Model) renderSyncView() string {
	var s strings.Builder

	// Title
	s.WriteString(syncTitleStyle.Render("Charm KV Sync"))
	s.WriteString("\n\n")

	// Get config from client
	cfg := m.client.Config()

	// Configuration section
	s.WriteString(syncHeaderStyle.Render("Configuration"))
	s.WriteString("\n\n")

	// Host
	s.WriteString(syncLabelStyle.Render("Server:"))
	s.WriteString(syncValueStyle.Render(cfg.Host))
	s.WriteString("\n")

	// Link status
	s.WriteString(syncLabelStyle.Render("Status:"))
	if cfg.Linked {
		s.WriteString(syncEnabledStyle.Render("✓ Linked"))
	} else {
		s.WriteString(syncDisabledStyle.Render("✗ Not linked"))
		s.WriteString("\n\n")
		s.WriteString(syncMessageStyle.Render("Run 'pagen charm link' to connect to the sync server."))
	}
	s.WriteString("\n")

	// Auto-sync status
	s.WriteString(syncLabelStyle.Render("Auto-sync:"))
	if cfg.AutoSync {
		s.WriteString(syncEnabledStyle.Render("✓ Enabled"))
	} else {
		s.WriteString(syncDisabledStyle.Render("✗ Disabled"))
	}
	s.WriteString("\n\n")

	// Sync actions section
	if cfg.Linked {
		s.WriteString(syncHeaderStyle.Render("Actions"))
		s.WriteString("\n\n")

		actions := []string{"Sync Now", "Toggle Auto-sync"}
		for i, action := range actions {
			if i == m.selectedService {
				s.WriteString("▶ ")
				s.WriteString(syncSelectedStyle.Render(action))
			} else {
				s.WriteString("  ")
				s.WriteString(action)
			}
			s.WriteString("\n")
		}
		s.WriteString("\n")
	}

	// Sync status
	if m.syncInProgress["charm"] {
		s.WriteString(syncMessageStyle.Render("⟳ Syncing..."))
		s.WriteString("\n\n")
	}

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
			s.WriteString("  " + m.syncMessages[i])
			s.WriteString("\n")
		}
		s.WriteString("\n")
	}

	// Help
	s.WriteString(m.renderSyncHelp())

	return s.String()
}

func (m Model) renderSyncHelp() string {
	cfg := m.client.Config()
	var help []string

	if cfg.Linked {
		help = []string{
			"↑/↓: Select action",
			"Enter: Execute",
			"s: Sync now",
			"a: Toggle auto-sync",
		}
	} else {
		help = []string{
			"Run 'pagen charm link' in terminal to connect",
		}
	}

	help = append(help, "Esc: Back", "q: Quit")
	return helpStyle.Render(strings.Join(help, " • "))
}

func (m Model) handleSyncKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cfg := m.client.Config()

	switch msg.String() {
	case "up", "k":
		if m.selectedService > 0 {
			m.selectedService--
		}
	case "down", "j":
		if m.selectedService < 1 { // 2 actions (0-1)
			m.selectedService++
		}
	case "enter":
		if !cfg.Linked {
			return m, nil
		}
		// Execute selected action
		switch m.selectedService {
		case 0: // Sync Now
			return m.triggerSync()
		case 1: // Toggle Auto-sync
			return m.toggleAutoSync()
		}
	case "s":
		// Quick sync
		if cfg.Linked {
			return m.triggerSync()
		}
	case "a":
		// Toggle auto-sync
		if cfg.Linked {
			return m.toggleAutoSync()
		}
	case "esc":
		// Go back to main view
		m.viewMode = ViewList
		m.entityType = EntityContacts
	}

	return m, nil
}

// triggerSync starts a manual sync operation.
func (m Model) triggerSync() (tea.Model, tea.Cmd) {
	m.syncInProgress["charm"] = true
	m.addSyncMessage("Starting sync...")

	return m, func() tea.Msg {
		err := m.client.Sync()
		return SyncCompleteMsg{Error: err}
	}
}

// toggleAutoSync toggles the auto-sync setting.
func (m Model) toggleAutoSync() (tea.Model, tea.Cmd) {
	cfg := m.client.Config()
	newState := !cfg.AutoSync

	return m, func() tea.Msg {
		err := cfg.SetAutoSync(newState)
		return AutoSyncToggleMsg{Enabled: newState, Error: err}
	}
}

// addSyncMessage adds a message to the sync message log.
func (m *Model) addSyncMessage(msg string) {
	timestamp := time.Now().Format("15:04:05")
	m.syncMessages = append(m.syncMessages, fmt.Sprintf("[%s] %s", timestamp, msg))
}

// handleSyncComplete handles sync completion messages.
func (m *Model) handleSyncComplete(msg SyncCompleteMsg) tea.Cmd {
	// Mark as no longer in progress
	m.syncInProgress["charm"] = false

	if msg.Error != nil {
		m.addSyncMessage(syncErrorStyle.Render(fmt.Sprintf("✗ Sync failed: %v", msg.Error)))
	} else {
		m.addSyncMessage(syncSuccessStyle.Render("✓ Sync completed"))
	}

	return nil
}

// handleAutoSyncToggle handles auto-sync toggle completion.
func (m *Model) handleAutoSyncToggle(msg AutoSyncToggleMsg) tea.Cmd {
	if msg.Error != nil {
		m.addSyncMessage(syncErrorStyle.Render(fmt.Sprintf("✗ Failed to toggle auto-sync: %v", msg.Error)))
	} else {
		if msg.Enabled {
			m.addSyncMessage(syncSuccessStyle.Render("✓ Auto-sync enabled"))
		} else {
			m.addSyncMessage(syncSuccessStyle.Render("✓ Auto-sync disabled"))
		}
	}

	return nil
}
