// ABOUTME: Tests for task TUI views
// ABOUTME: Validates task list and board rendering
package tui

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/objects"
	"github.com/stretchr/testify/assert"
)

func TestRenderTaskBoard(t *testing.T) {
	createdBy := uuid.New()

	// Create tasks with different statuses
	todoTask := objects.NewTaskObject(createdBy, "Todo task", createdBy, nil)

	inProgressTask := objects.NewTaskObject(createdBy, "In progress task", createdBy, nil)
	inProgressTask.SetStatus(objects.TaskStatusInProgress)

	dueAt := time.Now().Add(24 * time.Hour)
	doneTask := objects.NewTaskObject(createdBy, "Done task", createdBy, &dueAt)
	_ = doneTask.TransitionStatus(objects.TaskStatusDone)

	tasks := []*objects.TaskObject{todoTask, inProgressTask, doneTask}

	// Render board
	output := renderTaskBoard(tasks)

	// Verify output contains status columns
	assert.Contains(t, output, "TODO")
	assert.Contains(t, output, "IN PROGRESS")
	assert.Contains(t, output, "DONE")

	// Verify task titles appear
	assert.Contains(t, output, "Todo task")
	assert.Contains(t, output, "In progress task")
	assert.Contains(t, output, "Done task")
}

func TestRenderTaskCard(t *testing.T) {
	createdBy := uuid.New()

	// Task with due date
	dueAt := time.Now().Add(24 * time.Hour)
	task := objects.NewTaskObject(createdBy, "Test task", createdBy, &dueAt)

	output := renderTaskCard(task)

	assert.Contains(t, output, "Test task")
	assert.Contains(t, output, "◷") // Time symbol
}

func TestRenderTaskCardOverdue(t *testing.T) {
	createdBy := uuid.New()

	// Overdue task
	pastDue := time.Now().Add(-24 * time.Hour)
	task := objects.NewTaskObject(createdBy, "Overdue task", createdBy, &pastDue)

	output := renderTaskCard(task)

	assert.Contains(t, output, "Overdue task")
	assert.Contains(t, output, "⚠") // Warning symbol for overdue
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"days", 48 * time.Hour, "2 days"},
		{"hours", 5 * time.Hour, "5 hours"},
		{"minutes", 30 * time.Minute, "soon"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}
