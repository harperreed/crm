// ABOUTME: Tests for task database operations
// ABOUTME: Validates task CRUD operations and queries
package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/objects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTaskTestDB(t *testing.T) *sql.DB {
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

func TestCreateTask(t *testing.T) {
	db := setupTaskTestDB(t)

	createdBy := uuid.New()
	assigneeID := uuid.New()
	dueAt := time.Now().Add(24 * time.Hour).UTC()

	task := objects.NewTaskObject(createdBy, "Test task", assigneeID, &dueAt)

	err := CreateTask(db, task)
	require.NoError(t, err)

	// Verify task was created
	retrieved, err := GetTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, retrieved.ID)
	assert.Equal(t, task.GetTitle(), retrieved.GetTitle())
	assert.Equal(t, task.GetStatus(), retrieved.GetStatus())
	assert.Equal(t, task.GetAssigneeID(), retrieved.GetAssigneeID())
}

func TestGetTask(t *testing.T) {
	db := setupTaskTestDB(t)

	createdBy := uuid.New()
	task := objects.NewTaskObject(createdBy, "Test task", createdBy, nil)

	err := CreateTask(db, task)
	require.NoError(t, err)

	// Test successful retrieval
	retrieved, err := GetTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, retrieved.ID)

	// Test non-existent task
	_, err = GetTask(db, uuid.New())
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestUpdateTask(t *testing.T) {
	db := setupTaskTestDB(t)

	createdBy := uuid.New()
	task := objects.NewTaskObject(createdBy, "Original title", createdBy, nil)

	err := CreateTask(db, task)
	require.NoError(t, err)

	// Update task
	task.SetTitle("Updated title")
	task.SetStatus(objects.TaskStatusInProgress)

	err = UpdateTask(db, task)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := GetTask(db, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated title", retrieved.GetTitle())
	assert.Equal(t, objects.TaskStatusInProgress, retrieved.GetStatus())
}

func TestDeleteTask(t *testing.T) {
	db := setupTaskTestDB(t)

	createdBy := uuid.New()
	task := objects.NewTaskObject(createdBy, "Test task", createdBy, nil)

	err := CreateTask(db, task)
	require.NoError(t, err)

	// Delete task
	err = DeleteTask(db, task.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = GetTask(db, task.ID)
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestListTasks(t *testing.T) {
	db := setupTaskTestDB(t)

	createdBy := uuid.New()

	// Create multiple tasks
	task1 := objects.NewTaskObject(createdBy, "Task 1", createdBy, nil)
	task2 := objects.NewTaskObject(createdBy, "Task 2", createdBy, nil)
	task2.SetStatus(objects.TaskStatusInProgress)
	task3 := objects.NewTaskObject(createdBy, "Task 3", createdBy, nil)
	task3.SetStatus(objects.TaskStatusDone)

	require.NoError(t, CreateTask(db, task1))
	require.NoError(t, CreateTask(db, task2))
	require.NoError(t, CreateTask(db, task3))

	// List all tasks
	tasks, err := ListTasks(db, nil)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)

	// Filter by status
	filter := &TaskFilter{Status: objects.TaskStatusTodo}
	tasks, err = ListTasks(db, filter)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Task 1", tasks[0].GetTitle())

	// Filter by assignee
	filter = &TaskFilter{AssigneeID: &createdBy}
	tasks, err = ListTasks(db, filter)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)
}

func TestListOverdueTasks(t *testing.T) {
	db := setupTaskTestDB(t)

	createdBy := uuid.New()

	// Create overdue task
	pastDue := time.Now().Add(-24 * time.Hour).UTC()
	overdueTask := objects.NewTaskObject(createdBy, "Overdue task", createdBy, &pastDue)
	require.NoError(t, CreateTask(db, overdueTask))

	// Create future task
	futureDue := time.Now().Add(24 * time.Hour).UTC()
	futureTask := objects.NewTaskObject(createdBy, "Future task", createdBy, &futureDue)
	require.NoError(t, CreateTask(db, futureTask))

	// Create completed overdue task
	completedTask := objects.NewTaskObject(createdBy, "Completed task", createdBy, &pastDue)
	completedTask.SetStatus(objects.TaskStatusDone)
	require.NoError(t, CreateTask(db, completedTask))

	// List overdue tasks
	tasks, err := ListOverdueTasks(db)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Overdue task", tasks[0].GetTitle())
}

func TestListDueSoonTasks(t *testing.T) {
	db := setupTaskTestDB(t)

	createdBy := uuid.New()

	// Create task due in 3 days
	soonDue := time.Now().Add(3 * 24 * time.Hour).UTC()
	soonTask := objects.NewTaskObject(createdBy, "Due soon task", createdBy, &soonDue)
	require.NoError(t, CreateTask(db, soonTask))

	// Create task due in 10 days
	laterDue := time.Now().Add(10 * 24 * time.Hour).UTC()
	laterTask := objects.NewTaskObject(createdBy, "Due later task", createdBy, &laterDue)
	require.NoError(t, CreateTask(db, laterTask))

	// List tasks due soon (within 7 days)
	tasks, err := ListDueSoonTasks(db, 7)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Due soon task", tasks[0].GetTitle())
}

func TestListTasksByRelatedRecord(t *testing.T) {
	db := setupTaskTestDB(t)

	createdBy := uuid.New()
	relatedID := uuid.New()

	// Create task with related record
	task1 := objects.NewTaskObject(createdBy, "Task 1", createdBy, nil)
	task1.AddRelatedRecord(relatedID)
	require.NoError(t, CreateTask(db, task1))

	// Create task without related record
	task2 := objects.NewTaskObject(createdBy, "Task 2", createdBy, nil)
	require.NoError(t, CreateTask(db, task2))

	// List tasks by related record
	tasks, err := ListTasksByRelatedRecord(db, relatedID)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Task 1", tasks[0].GetTitle())
}
