// ABOUTME: TUI view for follow-up tracking
// ABOUTME: Displays prioritized list of contacts needing follow-up
package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
)

func (m Model) renderFollowupsTable() string {
	followups, err := m.client.GetFollowupList(100)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	columns := []table.Column{
		{Title: "Status", Width: 6},
		{Title: "Name", Width: 25},
		{Title: "Days", Width: 8},
		{Title: "Priority", Width: 10},
		{Title: "Strength", Width: 10},
		{Title: "Email", Width: 30},
	}

	var rows []table.Row
	for _, f := range followups {
		indicator := "🟢"
		if f.DaysSinceContact > f.CadenceDays+7 {
			indicator = "🔴"
		} else if f.DaysSinceContact >= f.CadenceDays-3 {
			indicator = "🟡"
		}

		rows = append(rows, table.Row{
			indicator,
			f.Name,
			fmt.Sprintf("%d", f.DaysSinceContact),
			fmt.Sprintf("%.1f", f.PriorityScore),
			f.RelationshipStrength,
			f.Email,
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
