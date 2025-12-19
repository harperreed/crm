package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/harperreed/pagen/viz"
)

func (m Model) renderGraphView() string {
	var s strings.Builder

	// Title
	s.WriteString(titleStyle.Render("GRAPH VIEW"))
	s.WriteString("\n\n")

	// DOT source (scrollable in future)
	if m.graphDOT == "" {
		s.WriteString("Generating graph...\n")
	} else {
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Render(m.graphDOT))
	}

	s.WriteString("\n\n")

	// Help
	s.WriteString(m.renderGraphHelp())

	return s.String()
}

func (m Model) renderGraphHelp() string {
	help := []string{
		"Esc: Back",
		"q: Quit",
	}
	return helpStyle.Render(strings.Join(help, " • "))
}

func (m Model) handleGraphKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.viewMode = ViewDetail
		m.graphDOT = ""
	}

	return m, nil
}

func (m *Model) generateGraph() error {
	generator := viz.NewGraphGenerator(m.client)

	var dot string
	var err error

	switch m.entityType {
	case EntityContacts:
		id, _ := uuid.Parse(m.selectedID)
		dot, err = generator.GenerateContactGraph(&id)
	case EntityCompanies:
		id, _ := uuid.Parse(m.selectedID)
		dot, err = generator.GenerateCompanyGraph(id)
	case EntityDeals:
		dot, err = generator.GeneratePipelineGraph()
	}

	if err != nil {
		return err
	}

	m.graphDOT = dot
	return nil
}
