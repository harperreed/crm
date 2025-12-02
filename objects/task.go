// ABOUTME: TaskObject implementation for Office OS task management
// ABOUTME: Provides task creation, status transitions, and due date tracking
package objects

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TaskObject represents a task in the Office OS.
type TaskObject struct {
	BaseObject
}

// Task field keys.
const (
	TaskFieldTitle            = "title"
	TaskFieldStatus           = "status"
	TaskFieldAssigneeID       = "assigneeId"
	TaskFieldDueAt            = "dueAt"
	TaskFieldCompletedAt      = "completedAt"
	TaskFieldRelatedRecordIDs = "relatedRecordIds"
)

// Task statuses.
const (
	TaskStatusTodo       = "todo"
	TaskStatusInProgress = "in_progress"
	TaskStatusDone       = "done"
	TaskStatusCancelled  = "cancelled"
)

// NewTaskObject creates a new task object.
func NewTaskObject(createdBy uuid.UUID, title string, assigneeID uuid.UUID, dueAt *time.Time) *TaskObject {
	now := time.Now().UTC()
	id := uuid.New()

	fields := map[string]interface{}{
		TaskFieldTitle:            title,
		TaskFieldStatus:           TaskStatusTodo,
		TaskFieldAssigneeID:       assigneeID.String(),
		TaskFieldRelatedRecordIDs: []string{},
	}

	if dueAt != nil {
		fields[TaskFieldDueAt] = dueAt.Format(time.RFC3339)
	}

	return &TaskObject{
		BaseObject: BaseObject{
			ID:        id,
			Kind:      KindTask,
			CreatedAt: now,
			UpdatedAt: now,
			CreatedBy: createdBy,
			ACL: []ACLEntry{
				{ActorID: createdBy, Role: RoleOwner},
			},
			Tags:   []string{},
			Fields: fields,
		},
	}
}

// GetTitle returns the task title.
func (t *TaskObject) GetTitle() string {
	if title, ok := t.Fields[TaskFieldTitle].(string); ok {
		return title
	}
	return ""
}

// SetTitle sets the task title.
func (t *TaskObject) SetTitle(title string) {
	t.Fields[TaskFieldTitle] = title
	t.UpdatedAt = time.Now().UTC()
}

// GetStatus returns the task status.
func (t *TaskObject) GetStatus() string {
	if status, ok := t.Fields[TaskFieldStatus].(string); ok {
		return status
	}
	return TaskStatusTodo
}

// SetStatus sets the task status without validation.
func (t *TaskObject) SetStatus(status string) {
	t.Fields[TaskFieldStatus] = status
	t.UpdatedAt = time.Now().UTC()
}

// TransitionStatus validates and transitions the task status.
func (t *TaskObject) TransitionStatus(newStatus string) error {
	// Validate status
	validStatuses := map[string]bool{
		TaskStatusTodo:       true,
		TaskStatusInProgress: true,
		TaskStatusDone:       true,
		TaskStatusCancelled:  true,
	}

	if !validStatuses[newStatus] {
		return fmt.Errorf("invalid task status: %s", newStatus)
	}

	oldStatus := t.GetStatus()
	t.SetStatus(newStatus)

	// Track completion
	if newStatus == TaskStatusDone && oldStatus != TaskStatusDone {
		now := time.Now().UTC()
		t.Fields[TaskFieldCompletedAt] = now.Format(time.RFC3339)
	} else if newStatus != TaskStatusDone {
		delete(t.Fields, TaskFieldCompletedAt)
	}

	return nil
}

// GetAssigneeID returns the assignee ID.
func (t *TaskObject) GetAssigneeID() uuid.UUID {
	if assigneeStr, ok := t.Fields[TaskFieldAssigneeID].(string); ok {
		if id, err := uuid.Parse(assigneeStr); err == nil {
			return id
		}
	}
	return uuid.Nil
}

// SetAssigneeID sets the assignee ID.
func (t *TaskObject) SetAssigneeID(assigneeID uuid.UUID) {
	t.Fields[TaskFieldAssigneeID] = assigneeID.String()
	t.UpdatedAt = time.Now().UTC()
}

// GetDueAt returns the due date.
func (t *TaskObject) GetDueAt() *time.Time {
	if dueStr, ok := t.Fields[TaskFieldDueAt].(string); ok {
		if dueAt, err := time.Parse(time.RFC3339, dueStr); err == nil {
			return &dueAt
		}
	}
	return nil
}

// SetDueAt sets the due date.
func (t *TaskObject) SetDueAt(dueAt *time.Time) {
	if dueAt != nil {
		t.Fields[TaskFieldDueAt] = dueAt.Format(time.RFC3339)
	} else {
		delete(t.Fields, TaskFieldDueAt)
	}
	t.UpdatedAt = time.Now().UTC()
}

// GetCompletedAt returns the completion timestamp.
func (t *TaskObject) GetCompletedAt() *time.Time {
	if completedStr, ok := t.Fields[TaskFieldCompletedAt].(string); ok {
		if completedAt, err := time.Parse(time.RFC3339, completedStr); err == nil {
			return &completedAt
		}
	}
	return nil
}

// GetRelatedRecordIDs returns the list of related record IDs.
func (t *TaskObject) GetRelatedRecordIDs() []uuid.UUID {
	var result []uuid.UUID

	switch v := t.Fields[TaskFieldRelatedRecordIDs].(type) {
	case []string:
		for _, idStr := range v {
			if id, err := uuid.Parse(idStr); err == nil {
				result = append(result, id)
			}
		}
	case []interface{}:
		for _, item := range v {
			if idStr, ok := item.(string); ok {
				if id, err := uuid.Parse(idStr); err == nil {
					result = append(result, id)
				}
			}
		}
	}

	return result
}

// AddRelatedRecord adds a related record ID.
func (t *TaskObject) AddRelatedRecord(recordID uuid.UUID) {
	ids := t.GetRelatedRecordIDs()

	// Check if already exists
	for _, id := range ids {
		if id == recordID {
			return
		}
	}

	ids = append(ids, recordID)

	// Convert to string slice for storage
	stringIDs := make([]string, len(ids))
	for i, id := range ids {
		stringIDs[i] = id.String()
	}

	t.Fields[TaskFieldRelatedRecordIDs] = stringIDs
	t.UpdatedAt = time.Now().UTC()
}

// IsOverdue returns true if the task is past its due date and not completed.
func (t *TaskObject) IsOverdue() bool {
	// Completed or cancelled tasks are not overdue
	status := t.GetStatus()
	if status == TaskStatusDone || status == TaskStatusCancelled {
		return false
	}

	dueAt := t.GetDueAt()
	if dueAt == nil {
		return false
	}

	return time.Now().UTC().After(*dueAt)
}

// IsDueSoon returns true if the task is due within the specified number of days.
func (t *TaskObject) IsDueSoon(days int) bool {
	// Completed or cancelled tasks are not due soon
	status := t.GetStatus()
	if status == TaskStatusDone || status == TaskStatusCancelled {
		return false
	}

	dueAt := t.GetDueAt()
	if dueAt == nil {
		return false
	}

	threshold := time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour)
	return dueAt.Before(threshold) && dueAt.After(time.Now().UTC())
}

// MarshalJSON implements json.Marshaler.
func (t *TaskObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.BaseObject)
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *TaskObject) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &t.BaseObject)
}
