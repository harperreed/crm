// ABOUTME: Integration tests for task management system
// ABOUTME: Tests end-to-end task workflows with database
package objects

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func setupIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// Create the objects table
	_, err = db.Exec(`
		CREATE TABLE objects (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			created_by TEXT NOT NULL,
			acl TEXT NOT NULL,
			tags TEXT,
			fields TEXT NOT NULL
		);
		CREATE INDEX idx_objects_kind ON objects(kind);
		CREATE INDEX idx_objects_created_by ON objects(created_by);
		CREATE INDEX idx_objects_created_at ON objects(created_at);
	`)
	require.NoError(t, err)

	return db
}

func TestTaskManagementWorkflow(t *testing.T) {
	_ = setupIntegrationDB(t)
	createdBy := uuid.New()
	assigneeID := uuid.New()

	// Create a task with a due date
	dueAt := time.Now().Add(7 * 24 * time.Hour).UTC()
	task := NewTaskObject(createdBy, "Complete Q1 planning", assigneeID, &dueAt)

	assert.Equal(t, "Complete Q1 planning", task.GetTitle())
	assert.Equal(t, TaskStatusTodo, task.GetStatus())
	assert.False(t, task.IsOverdue())
	assert.True(t, task.IsDueSoon(7))

	// Transition to in progress
	err := task.TransitionStatus(TaskStatusInProgress)
	require.NoError(t, err)
	assert.Equal(t, TaskStatusInProgress, task.GetStatus())

	// Add related records
	contactID := uuid.New()
	dealID := uuid.New()
	task.AddRelatedRecord(contactID)
	task.AddRelatedRecord(dealID)

	relatedIDs := task.GetRelatedRecordIDs()
	assert.Len(t, relatedIDs, 2)
	assert.Contains(t, relatedIDs, contactID)
	assert.Contains(t, relatedIDs, dealID)

	// Complete the task
	err = task.TransitionStatus(TaskStatusDone)
	require.NoError(t, err)
	assert.Equal(t, TaskStatusDone, task.GetStatus())
	assert.NotNil(t, task.GetCompletedAt())
	assert.False(t, task.IsOverdue())
}

func TestOverdueTaskDetection(t *testing.T) {
	_ = setupIntegrationDB(t)
	createdBy := uuid.New()

	// Create an overdue task
	pastDue := time.Now().Add(-7 * 24 * time.Hour).UTC()
	overdueTask := NewTaskObject(createdBy, "Overdue task", createdBy, &pastDue)

	assert.True(t, overdueTask.IsOverdue())
	assert.False(t, overdueTask.IsDueSoon(7))

	// Completing it should clear overdue status
	err := overdueTask.TransitionStatus(TaskStatusDone)
	require.NoError(t, err)
	assert.False(t, overdueTask.IsOverdue())
}

func TestTaskStatusValidation(t *testing.T) {
	createdBy := uuid.New()
	task := NewTaskObject(createdBy, "Test task", createdBy, nil)

	// Invalid status should error
	err := task.TransitionStatus("invalid_status")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task status")

	// Status should remain unchanged
	assert.Equal(t, TaskStatusTodo, task.GetStatus())
}

func TestTaskRelatedRecordLinking(t *testing.T) {
	createdBy := uuid.New()
	task := NewTaskObject(createdBy, "Follow up on deal", createdBy, nil)

	// Add multiple related records
	ids := []uuid.UUID{
		uuid.New(),
		uuid.New(),
		uuid.New(),
	}

	for _, id := range ids {
		task.AddRelatedRecord(id)
	}

	relatedIDs := task.GetRelatedRecordIDs()
	assert.Len(t, relatedIDs, 3)

	// Adding duplicate should not create duplicates
	task.AddRelatedRecord(ids[0])
	relatedIDs = task.GetRelatedRecordIDs()
	assert.Len(t, relatedIDs, 3)
}

func TestTaskCompletionCycle(t *testing.T) {
	createdBy := uuid.New()
	task := NewTaskObject(createdBy, "Task", createdBy, nil)

	// Initially no completion time
	assert.Nil(t, task.GetCompletedAt())

	// Complete the task
	err := task.TransitionStatus(TaskStatusDone)
	require.NoError(t, err)
	assert.NotNil(t, task.GetCompletedAt())
	firstCompletedAt := task.GetCompletedAt()

	// Reopen the task
	err = task.TransitionStatus(TaskStatusTodo)
	require.NoError(t, err)
	assert.Nil(t, task.GetCompletedAt())

	// Complete again
	time.Sleep(10 * time.Millisecond)
	err = task.TransitionStatus(TaskStatusDone)
	require.NoError(t, err)
	assert.NotNil(t, task.GetCompletedAt())

	// New completion time should be different
	secondCompletedAt := task.GetCompletedAt()
	assert.True(t, secondCompletedAt.After(*firstCompletedAt) || secondCompletedAt.Equal(*firstCompletedAt))
}
