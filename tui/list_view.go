package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/harperreed/pagen/charm"
)

func (m Model) renderListView() string {
	var s strings.Builder

	// Title
	s.WriteString(titleStyle.Render("PAGEN CRM"))
	s.WriteString("\n\n")

	// Show delete message if present
	if m.deleteMessage != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)
		s.WriteString(msgStyle.Render("✓ " + m.deleteMessage))
		s.WriteString("\n\n")
	}

	// Tabs
	s.WriteString(m.renderTabs())
	s.WriteString("\n\n")

	// Table
	s.WriteString(m.renderTable())
	s.WriteString("\n\n")

	// Help
	s.WriteString(m.renderListHelp())

	return s.String()
}

func (m Model) renderTabs() string {
	tabs := []string{"Contacts", "Companies", "Deals", "Followups", "Sync"}
	var rendered []string

	for i, tab := range tabs {
		if EntityType(i) == m.entityType {
			rendered = append(rendered, tabActiveStyle.Render(tab))
		} else {
			rendered = append(rendered, tabInactiveStyle.Render(tab))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m Model) renderTable() string {
	switch m.entityType {
	case EntityContacts:
		return m.renderContactsTable()
	case EntityCompanies:
		return m.renderCompaniesTable()
	case EntityDeals:
		return m.renderDealsTable()
	case EntityFollowups:
		return m.renderFollowupsTable()
	case EntitySync:
		return m.renderSyncView()
	}
	return ""
}

func (m Model) renderContactsTable() string {
	contacts, err := m.client.ListContacts(&charm.ContactFilter{
		Query: m.searchQuery,
		Limit: 100,
	})
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Email", Width: 30},
		{Title: "Company", Width: 20},
	}

	var rows []table.Row
	for _, contact := range contacts {
		// Company name is denormalized in charm model
		rows = append(rows, table.Row{
			contact.Name,
			contact.Email,
			contact.CompanyName,
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(m.height-10),
	)

	// Set selected row
	if m.selectedRow < len(rows) {
		t.SetCursor(m.selectedRow)
	}

	return t.View()
}

func (m Model) renderCompaniesTable() string {
	companies, err := m.client.ListCompanies(&charm.CompanyFilter{
		Query: m.searchQuery,
		Limit: 100,
	})
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Domain", Width: 30},
		{Title: "Industry", Width: 20},
	}

	var rows []table.Row
	for _, company := range companies {
		rows = append(rows, table.Row{
			company.Name,
			company.Domain,
			company.Industry,
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(m.height-10),
	)

	if m.selectedRow < len(rows) {
		t.SetCursor(m.selectedRow)
	}

	return t.View()
}

func (m Model) renderDealsTable() string {
	deals, err := m.client.ListDeals(&charm.DealFilter{
		Limit: 100,
	})
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	columns := []table.Column{
		{Title: "Title", Width: 30},
		{Title: "Company", Width: 25},
		{Title: "Stage", Width: 15},
		{Title: "Amount", Width: 10},
	}

	var rows []table.Row
	for _, deal := range deals {
		// Company name is denormalized in charm model
		amountStr := fmt.Sprintf("$%dK", deal.Amount/100000)

		rows = append(rows, table.Row{
			deal.Title,
			deal.CompanyName,
			deal.Stage,
			amountStr,
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(m.height-10),
	)

	if m.selectedRow < len(rows) {
		t.SetCursor(m.selectedRow)
	}

	return t.View()
}

func (m Model) renderListHelp() string {
	help := []string{
		"↑/↓: Navigate",
		"Tab: Switch tabs",
		"f: Followups",
		"s: Sync",
		"Enter: View details",
		"/: Search",
		"n: New",
		"q: Quit",
	}
	return helpStyle.Render(strings.Join(help, " • "))
}

func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If we're in sync view, delegate to sync handler
	if m.entityType == EntitySync {
		return m.handleSyncKeys(msg)
	}

	switch msg.String() {
	case "up", "k":
		if m.selectedRow > 0 {
			m.selectedRow--
		}
	case "down", "j":
		m.selectedRow++
	case "tab":
		m.entityType = (m.entityType + 1) % 5
		m.selectedRow = 0
	case "f":
		// Jump to followups tab
		m.entityType = EntityFollowups
		m.selectedRow = 0
	case "s":
		// Jump to sync tab
		m.entityType = EntitySync
		m.selectedRow = 0
	case "enter":
		// Switch to detail view
		m.viewMode = ViewDetail
		m.selectedID = m.getSelectedID()
	case "/":
		// TODO: Enter search mode
	case "n":
		// Switch to edit view (new)
		m.viewMode = ViewEdit
		m.selectedID = ""
		m.initFormInputs()
	}

	return m, nil
}

func (m Model) getSelectedID() string {
	switch m.entityType {
	case EntityContacts:
		contacts, _ := m.client.ListContacts(&charm.ContactFilter{
			Query: m.searchQuery,
			Limit: 100,
		})
		if m.selectedRow < len(contacts) {
			return contacts[m.selectedRow].ID.String()
		}
	case EntityCompanies:
		companies, _ := m.client.ListCompanies(&charm.CompanyFilter{
			Query: m.searchQuery,
			Limit: 100,
		})
		if m.selectedRow < len(companies) {
			return companies[m.selectedRow].ID.String()
		}
	case EntityDeals:
		deals, _ := m.client.ListDeals(&charm.DealFilter{
			Limit: 100,
		})
		if m.selectedRow < len(deals) {
			return deals[m.selectedRow].ID.String()
		}
	}
	return ""
}
