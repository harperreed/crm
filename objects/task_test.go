// ABOUTME: Tests for TaskObject and task-specific functionality
// ABOUTME: Validates task creation, status transitions, and field management
package objects

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskObject(t *testing.T) {
	createdBy := uuid.New()
	title := "Follow up with Sarah about Q1 planning"
	assigneeID := uuid.New()
	dueAt := time.Now().Add(24 * time.Hour).UTC()

	task := NewTaskObject(createdBy, title, assigneeID, &dueAt)

	assert.NotEqual(t, uuid.Nil, task.ID)
	assert.Equal(t, KindTask, task.Kind)
	assert.Equal(t, createdBy, task.CreatedBy)
	assert.NotZero(t, task.CreatedAt)
	assert.NotZero(t, task.UpdatedAt)
	assert.Len(t, task.ACL, 1)
	assert.Equal(t, createdBy, task.ACL[0].ActorID)
	assert.Equal(t, RoleOwner, task.ACL[0].Role)
	assert.Equal(t, title, task.GetTitle())
	assert.Equal(t, TaskStatusTodo, task.GetStatus())
	assert.Equal(t, assigneeID, task.GetAssigneeID())
	assert.NotNil(t, task.GetDueAt())
	assert.Equal(t, dueAt.Unix(), task.GetDueAt().Unix())
}

func TestTaskObject_GettersSetters(t *testing.T) {
	createdBy := uuid.New()
	task := NewTaskObject(createdBy, "Test task", createdBy, nil)

	// Test title
	newTitle := "Updated title"
	task.SetTitle(newTitle)
	assert.Equal(t, newTitle, task.GetTitle())

	// Test status
	task.SetStatus(TaskStatusInProgress)
	assert.Equal(t, TaskStatusInProgress, task.GetStatus())

	// Test assignee
	newAssignee := uuid.New()
	task.SetAssigneeID(newAssignee)
	assert.Equal(t, newAssignee, task.GetAssigneeID())

	// Test due date
	dueAt := time.Now().Add(48 * time.Hour).UTC()
	task.SetDueAt(&dueAt)
	assert.NotNil(t, task.GetDueAt())
	assert.Equal(t, dueAt.Unix(), task.GetDueAt().Unix())

	// Test related records
	relatedID := uuid.New()
	task.AddRelatedRecord(relatedID)
	assert.Contains(t, task.GetRelatedRecordIDs(), relatedID)
}

func TestTaskObject_StatusTransitions(t *testing.T) {
	tests := []struct {
		name        string
		fromStatus  string
		toStatus    string
		shouldError bool
	}{
		{"todo to in_progress", TaskStatusTodo, TaskStatusInProgress, false},
		{"todo to done", TaskStatusTodo, TaskStatusDone, false},
		{"todo to cancelled", TaskStatusTodo, TaskStatusCancelled, false},
		{"in_progress to done", TaskStatusInProgress, TaskStatusDone, false},
		{"in_progress to cancelled", TaskStatusInProgress, TaskStatusCancelled, false},
		{"in_progress to todo", TaskStatusInProgress, TaskStatusTodo, false},
		{"done to todo", TaskStatusDone, TaskStatusTodo, false},
		{"cancelled to todo", TaskStatusCancelled, TaskStatusTodo, false},
		{"invalid status", TaskStatusTodo, "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createdBy := uuid.New()
			task := NewTaskObject(createdBy, "Test", createdBy, nil)
			task.SetStatus(tt.fromStatus)

			err := task.TransitionStatus(tt.toStatus)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.toStatus, task.GetStatus())
			}
		})
	}
}

func TestTaskObject_CompletionTracking(t *testing.T) {
	createdBy := uuid.New()
	task := NewTaskObject(createdBy, "Test", createdBy, nil)

	// Initially not completed
	assert.Nil(t, task.GetCompletedAt())

	// Complete the task
	err := task.TransitionStatus(TaskStatusDone)
	require.NoError(t, err)
	assert.NotNil(t, task.GetCompletedAt())
	completedAt := task.GetCompletedAt()

	// Reopen the task
	err = task.TransitionStatus(TaskStatusTodo)
	require.NoError(t, err)
	assert.Nil(t, task.GetCompletedAt())

	// Re-complete should update timestamp
	time.Sleep(100 * time.Millisecond)
	err = task.TransitionStatus(TaskStatusDone)
	require.NoError(t, err)
	assert.NotNil(t, task.GetCompletedAt())
	newCompletedAt := task.GetCompletedAt()
	assert.True(t, newCompletedAt.After(*completedAt) || newCompletedAt.Equal(*completedAt))
}

func TestTaskObject_IsOverdue(t *testing.T) {
	createdBy := uuid.New()

	// No due date - not overdue
	task1 := NewTaskObject(createdBy, "Test", createdBy, nil)
	assert.False(t, task1.IsOverdue())

	// Due date in past - overdue
	pastDue := time.Now().Add(-24 * time.Hour).UTC()
	task2 := NewTaskObject(createdBy, "Test", createdBy, &pastDue)
	assert.True(t, task2.IsOverdue())

	// Due date in future - not overdue
	futureDue := time.Now().Add(24 * time.Hour).UTC()
	task3 := NewTaskObject(createdBy, "Test", createdBy, &futureDue)
	assert.False(t, task3.IsOverdue())

	// Completed task - not overdue
	task4 := NewTaskObject(createdBy, "Test", createdBy, &pastDue)
	task4.SetStatus(TaskStatusDone)
	assert.False(t, task4.IsOverdue())
}

func TestTaskObject_IsDueSoon(t *testing.T) {
	createdBy := uuid.New()

	// No due date - not due soon
	task1 := NewTaskObject(createdBy, "Test", createdBy, nil)
	assert.False(t, task1.IsDueSoon(7))

	// Due in 3 days - due soon (within 7 days)
	soonDue := time.Now().Add(3 * 24 * time.Hour).UTC()
	task2 := NewTaskObject(createdBy, "Test", createdBy, &soonDue)
	assert.True(t, task2.IsDueSoon(7))

	// Due in 10 days - not due soon (outside 7 days)
	laterDue := time.Now().Add(10 * 24 * time.Hour).UTC()
	task3 := NewTaskObject(createdBy, "Test", createdBy, &laterDue)
	assert.False(t, task3.IsDueSoon(7))

	// Completed task - not due soon
	task4 := NewTaskObject(createdBy, "Test", createdBy, &soonDue)
	task4.SetStatus(TaskStatusDone)
	assert.False(t, task4.IsDueSoon(7))
}

func TestTaskObject_JSONSerialization(t *testing.T) {
	createdBy := uuid.New()
	assigneeID := uuid.New()
	dueAt := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	relatedID := uuid.New()

	task := NewTaskObject(createdBy, "Follow up with Sarah", assigneeID, &dueAt)
	task.AddRelatedRecord(relatedID)
	task.AddRelatedRecord(uuid.New())

	// Serialize
	data, err := json.Marshal(task)
	require.NoError(t, err)

	// Deserialize
	var decoded TaskObject
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, task.ID, decoded.ID)
	assert.Equal(t, task.Kind, decoded.Kind)
	assert.Equal(t, task.GetTitle(), decoded.GetTitle())
	assert.Equal(t, task.GetStatus(), decoded.GetStatus())
	assert.Equal(t, task.GetAssigneeID(), decoded.GetAssigneeID())
	assert.Equal(t, task.GetDueAt().Unix(), decoded.GetDueAt().Unix())
	assert.Equal(t, len(task.GetRelatedRecordIDs()), len(decoded.GetRelatedRecordIDs()))
}
