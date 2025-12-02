// ABOUTME: Task list and board views for the TUI
// ABOUTME: Renders task tables and kanban board displays
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/objects"
)

func renderTaskBoard(tasks []*objects.TaskObject) string {
	var s strings.Builder

	// Group tasks by status
	todoTasks := []*objects.TaskObject{}
	inProgressTasks := []*objects.TaskObject{}
	doneTasks := []*objects.TaskObject{}

	for _, task := range tasks {
		switch task.GetStatus() {
		case objects.TaskStatusTodo:
			todoTasks = append(todoTasks, task)
		case objects.TaskStatusInProgress:
			inProgressTasks = append(inProgressTasks, task)
		case objects.TaskStatusDone:
			doneTasks = append(doneTasks, task)
		}
	}

	// Column styles
	columnStyle := lipgloss.NewStyle().
		Width(30).
		Padding(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	// Render columns
	todoColumn := renderTaskColumn("TODO", todoTasks, columnStyle, headerStyle)
	inProgressColumn := renderTaskColumn("IN PROGRESS", inProgressTasks, columnStyle, headerStyle)
	doneColumn := renderTaskColumn("DONE", doneTasks, columnStyle, headerStyle)

	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, todoColumn, inProgressColumn, doneColumn))

	return s.String()
}

func renderTaskColumn(title string, tasks []*objects.TaskObject, columnStyle, headerStyle lipgloss.Style) string {
	var s strings.Builder

	s.WriteString(headerStyle.Render(title))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("(%d tasks)\n\n", len(tasks)))

	for _, task := range tasks {
		taskCard := renderTaskCard(task)
		s.WriteString(taskCard)
		s.WriteString("\n")
	}

	return columnStyle.Render(s.String())
}

func renderTaskCard(task *objects.TaskObject) string {
	cardStyle := lipgloss.NewStyle().
		Width(26).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238"))

	var s strings.Builder

	s.WriteString(task.GetTitle())
	s.WriteString("\n")

	if dueAt := task.GetDueAt(); dueAt != nil {
		dueStr := dueAt.Format("Jan 2, 2006")
		if task.IsOverdue() {
			dueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
			s.WriteString(dueStyle.Render("⚠ " + dueStr))
		} else if task.IsDueSoon(7) {
			dueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
			s.WriteString(dueStyle.Render("◷ " + dueStr))
		} else {
			s.WriteString("◷ " + dueStr)
		}
	}

	return cardStyle.Render(s.String())
}

// RenderTaskList renders a simple task list view.
func RenderTaskList(database interface{}, filter *db.TaskFilter) (string, error) {
	sqlDB, ok := database.(*interface{})
	if !ok {
		return "", fmt.Errorf("invalid database type")
	}

	// This is a stub - would need proper type assertion
	_ = sqlDB
	_ = filter

	var s strings.Builder

	s.WriteString("Task List\n\n")
	s.WriteString("No tasks found.\n")

	return s.String(), nil
}

// FormatDuration formats a time duration into a human-readable string.
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	days := hours / 24

	if days > 0 {
		return fmt.Sprintf("%d days", days)
	}
	if hours > 0 {
		return fmt.Sprintf("%d hours", hours)
	}
	return "soon"
}
