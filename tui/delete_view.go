// ABOUTME: Delete confirmation view for TUI
// ABOUTME: Handles deletion of contacts, companies, and deals with confirmation dialog
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

var (
	confirmBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("9")).
			Padding(1, 2).
			Width(60).
			Align(lipgloss.Center)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	confirmButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("9")).
				Padding(0, 2).
				MarginRight(2)

	cancelButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("8")).
				Padding(0, 2)
)

func (m Model) renderConfirmDeleteView() string {
	var entityName string
	var entityType string

	id, err := uuid.Parse(m.selectedID)
	if err != nil {
		return fmt.Sprintf("Error: invalid ID: %v", err)
	}

	switch m.entityType {
	case EntityContacts:
		contact, err := m.client.GetContact(id)
		if err != nil {
			return fmt.Sprintf("Error loading contact: %v", err)
		}
		entityName = contact.Name
		entityType = "contact"
	case EntityCompanies:
		company, err := m.client.GetCompany(id)
		if err != nil {
			return fmt.Sprintf("Error loading company: %v", err)
		}
		entityName = company.Name
		entityType = "company"
	case EntityDeals:
		deal, err := m.client.GetDeal(id)
		if err != nil {
			return fmt.Sprintf("Error loading deal: %v", err)
		}
		entityName = deal.Title
		entityType = "deal"
	}

	title := warningStyle.Render("⚠  DELETE CONFIRMATION  ⚠")
	message := fmt.Sprintf("Are you sure you want to delete this %s?", entityType)
	entityInfo := fmt.Sprintf("\n%s: %s\n", strings.ToUpper(entityType), entityName)
	warning := "\nThis action cannot be undone!"

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Left,
		confirmButtonStyle.Render("Yes, Delete (y)"),
		cancelButtonStyle.Render("Cancel (n/esc)"),
	)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		"",
		message,
		entityInfo,
		warning,
		"",
		buttons,
	)

	box := confirmBoxStyle.Render(content)

	// Center the box on screen
	dialog := lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)

	return dialog
}

func (m Model) handleConfirmDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Confirm delete
		err := m.performDelete()
		if err != nil {
			m.err = err
			m.deleteMessage = "Error: " + err.Error()
			m.viewMode = ViewList
		} else {
			m.deleteMessage = "Successfully deleted"
			m.viewMode = ViewList
			m.selectedID = "" // Clear selection
		}
	case "n", "N", "esc":
		// Cancel delete
		m.viewMode = ViewDetail
	}

	return m, nil
}

func (m Model) performDelete() error {
	id, err := uuid.Parse(m.selectedID)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}

	switch m.entityType {
	case EntityContacts:
		return m.client.DeleteContact(id)
	case EntityCompanies:
		return m.client.DeleteCompany(id)
	case EntityDeals:
		return m.client.DeleteDeal(id)
	default:
		return fmt.Errorf("unknown entity type")
	}
}
