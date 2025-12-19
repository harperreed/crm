package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/harperreed/pagen/charm"
)

var (
	fieldLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			Width(20)

	fieldValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)

func (m Model) renderDetailView() string {
	var s strings.Builder

	// Title
	s.WriteString(titleStyle.Render("DETAIL VIEW"))
	s.WriteString("\n\n")

	// Entity details
	switch m.entityType {
	case EntityContacts:
		s.WriteString(m.renderContactDetail())
	case EntityCompanies:
		s.WriteString(m.renderCompanyDetail())
	case EntityDeals:
		s.WriteString(m.renderDealDetail())
	}

	s.WriteString("\n\n")

	// Help
	s.WriteString(m.renderDetailHelp())

	return s.String()
}

func (m Model) renderContactDetail() string {
	id, err := uuid.Parse(m.selectedID)
	if err != nil {
		return fmt.Sprintf("Error: invalid ID: %v", err)
	}

	contact, err := m.client.GetContact(id)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	var s strings.Builder

	s.WriteString(m.renderField("Name", contact.Name))
	s.WriteString(m.renderField("Email", contact.Email))
	s.WriteString(m.renderField("Phone", contact.Phone))

	// Company name is denormalized in charm model
	if contact.CompanyName != "" {
		s.WriteString(m.renderField("Company", contact.CompanyName))
	}

	if contact.LastContactedAt != nil {
		s.WriteString(m.renderField("Last Contacted", contact.LastContactedAt.Format("2006-01-02")))
	}

	s.WriteString(m.renderField("Notes", contact.Notes))

	// Related entities
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("RELATIONSHIPS"))
	s.WriteString("\n")

	relationships, _ := m.client.ListRelationshipsForContact(id)
	for _, rel := range relationships {
		s.WriteString(fmt.Sprintf("  • %s (%s)\n", rel.Context, rel.RelationshipType))
	}

	return s.String()
}

func (m Model) renderCompanyDetail() string {
	id, err := uuid.Parse(m.selectedID)
	if err != nil {
		return fmt.Sprintf("Error: invalid ID: %v", err)
	}

	company, err := m.client.GetCompany(id)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	var s strings.Builder

	s.WriteString(m.renderField("Name", company.Name))
	s.WriteString(m.renderField("Domain", company.Domain))
	s.WriteString(m.renderField("Industry", company.Industry))
	s.WriteString(m.renderField("Notes", company.Notes))

	// Contacts at company
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("CONTACTS"))
	s.WriteString("\n")

	contacts, _ := m.client.ListContacts(&charm.ContactFilter{
		CompanyID: &id,
		Limit:     100,
	})
	for _, contact := range contacts {
		s.WriteString(fmt.Sprintf("  • %s (%s)\n", contact.Name, contact.Email))
	}

	return s.String()
}

func (m Model) renderDealDetail() string {
	id, err := uuid.Parse(m.selectedID)
	if err != nil {
		return fmt.Sprintf("Error: invalid ID: %v", err)
	}

	deal, err := m.client.GetDeal(id)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	var s strings.Builder

	s.WriteString(m.renderField("Title", deal.Title))

	// Company and contact names are denormalized in charm model
	if deal.CompanyName != "" {
		s.WriteString(m.renderField("Company", deal.CompanyName))
	}

	if deal.ContactName != "" {
		s.WriteString(m.renderField("Contact", deal.ContactName))
	}

	s.WriteString(m.renderField("Stage", deal.Stage))
	s.WriteString(m.renderField("Amount", fmt.Sprintf("$%d %s", deal.Amount/100, deal.Currency)))

	if deal.ExpectedCloseDate != nil {
		s.WriteString(m.renderField("Expected Close", deal.ExpectedCloseDate.Format("2006-01-02")))
	}

	// Notes
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("NOTES"))
	s.WriteString("\n")

	notes, _ := m.client.ListDealNotes(id)
	for _, note := range notes {
		s.WriteString(fmt.Sprintf("  • [%s] %s\n", note.CreatedAt.Format("2006-01-02"), note.Content))
	}

	return s.String()
}

func (m Model) renderField(label, value string) string {
	if value == "" {
		value = "-"
	}
	return fmt.Sprintf("%s %s\n",
		fieldLabelStyle.Render(label+":"),
		fieldValueStyle.Render(value))
}

func (m Model) renderDetailHelp() string {
	help := []string{
		"Esc: Back",
		"e: Edit",
		"d: Delete",
		"g: View graph",
		"q: Quit",
	}
	return helpStyle.Render(strings.Join(help, " • "))
}

func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.viewMode = ViewList
	case "e":
		m.viewMode = ViewEdit
		m.initFormInputs()
	case "d":
		// Show delete confirmation
		m.viewMode = ViewConfirmDelete
		m.deleteConfirmed = false
	case "g":
		m.viewMode = ViewGraph
		err := m.generateGraph()
		if err != nil {
			m.err = err
			m.viewMode = ViewDetail
		}
	}

	return m, nil
}
