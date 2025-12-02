// ABOUTME: Database operations for task management
// ABOUTME: Provides CRUD operations and queries for TaskObject
package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/objects"
)

// TaskFilter defines filtering options for task queries.
type TaskFilter struct {
	Status     string
	AssigneeID *uuid.UUID
	DueBefore  *time.Time
	DueAfter   *time.Time
}

// CreateTask inserts a new task into the database.
func CreateTask(db *sql.DB, task *objects.TaskObject) error {
	aclJSON, err := json.Marshal(task.ACL)
	if err != nil {
		return fmt.Errorf("failed to marshal ACL: %w", err)
	}

	tagsJSON, err := json.Marshal(task.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	fieldsJSON, err := json.Marshal(task.Fields)
	if err != nil {
		return fmt.Errorf("failed to marshal fields: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO objects (id, kind, created_at, updated_at, created_by, acl, tags, fields)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		task.ID.String(),
		task.Kind,
		task.CreatedAt.Format(time.RFC3339),
		task.UpdatedAt.Format(time.RFC3339),
		task.CreatedBy.String(),
		string(aclJSON),
		string(tagsJSON),
		string(fieldsJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to insert task: %w", err)
	}

	return nil
}

// GetTask retrieves a task by ID.
func GetTask(db *sql.DB, id uuid.UUID) (*objects.TaskObject, error) {
	var (
		idStr      string
		kind       string
		createdAt  string
		updatedAt  string
		createdBy  string
		aclJSON    string
		tagsJSON   sql.NullString
		fieldsJSON string
	)

	err := db.QueryRow(`
		SELECT id, kind, created_at, updated_at, created_by, acl, tags, fields
		FROM objects
		WHERE id = ? AND kind = ?
	`, id.String(), objects.KindTask).Scan(
		&idStr, &kind, &createdAt, &updatedAt, &createdBy, &aclJSON, &tagsJSON, &fieldsJSON,
	)

	if err != nil {
		return nil, err
	}

	return parseTaskObject(idStr, kind, createdAt, updatedAt, createdBy, aclJSON, tagsJSON, fieldsJSON)
}

// UpdateTask updates an existing task.
func UpdateTask(db *sql.DB, task *objects.TaskObject) error {
	task.UpdatedAt = time.Now().UTC()

	aclJSON, err := json.Marshal(task.ACL)
	if err != nil {
		return fmt.Errorf("failed to marshal ACL: %w", err)
	}

	tagsJSON, err := json.Marshal(task.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	fieldsJSON, err := json.Marshal(task.Fields)
	if err != nil {
		return fmt.Errorf("failed to marshal fields: %w", err)
	}

	_, err = db.Exec(`
		UPDATE objects
		SET updated_at = ?, acl = ?, tags = ?, fields = ?
		WHERE id = ? AND kind = ?
	`,
		task.UpdatedAt.Format(time.RFC3339),
		string(aclJSON),
		string(tagsJSON),
		string(fieldsJSON),
		task.ID.String(),
		objects.KindTask,
	)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// DeleteTask removes a task from the database.
func DeleteTask(db *sql.DB, id uuid.UUID) error {
	_, err := db.Exec(`
		DELETE FROM objects
		WHERE id = ? AND kind = ?
	`, id.String(), objects.KindTask)

	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// ListTasks retrieves tasks with optional filtering.
func ListTasks(db *sql.DB, filter *TaskFilter) ([]*objects.TaskObject, error) {
	query := `
		SELECT id, kind, created_at, updated_at, created_by, acl, tags, fields
		FROM objects
		WHERE kind = ?
	`
	args := []interface{}{objects.KindTask}

	if filter != nil {
		if filter.Status != "" {
			query += ` AND json_extract(fields, '$.status') = ?`
			args = append(args, filter.Status)
		}

		if filter.AssigneeID != nil {
			query += ` AND json_extract(fields, '$.assigneeId') = ?`
			args = append(args, filter.AssigneeID.String())
		}

		if filter.DueBefore != nil {
			query += ` AND json_extract(fields, '$.dueAt') IS NOT NULL AND json_extract(fields, '$.dueAt') <= ?`
			args = append(args, filter.DueBefore.Format(time.RFC3339))
		}

		if filter.DueAfter != nil {
			query += ` AND json_extract(fields, '$.dueAt') IS NOT NULL AND json_extract(fields, '$.dueAt') >= ?`
			args = append(args, filter.DueAfter.Format(time.RFC3339))
		}
	}

	query += ` ORDER BY created_at DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tasks []*objects.TaskObject
	for rows.Next() {
		var (
			idStr      string
			kind       string
			createdAt  string
			updatedAt  string
			createdBy  string
			aclJSON    string
			tagsJSON   sql.NullString
			fieldsJSON string
		)

		err := rows.Scan(&idStr, &kind, &createdAt, &updatedAt, &createdBy, &aclJSON, &tagsJSON, &fieldsJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		task, err := parseTaskObject(idStr, kind, createdAt, updatedAt, createdBy, aclJSON, tagsJSON, fieldsJSON)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// ListOverdueTasks returns all overdue tasks.
func ListOverdueTasks(db *sql.DB) ([]*objects.TaskObject, error) {
	now := time.Now().UTC()

	rows, err := db.Query(`
		SELECT id, kind, created_at, updated_at, created_by, acl, tags, fields
		FROM objects
		WHERE kind = ?
			AND json_extract(fields, '$.dueAt') IS NOT NULL
			AND json_extract(fields, '$.dueAt') < ?
			AND json_extract(fields, '$.status') NOT IN (?, ?)
		ORDER BY json_extract(fields, '$.dueAt') ASC
	`, objects.KindTask, now.Format(time.RFC3339), objects.TaskStatusDone, objects.TaskStatusCancelled)

	if err != nil {
		return nil, fmt.Errorf("failed to query overdue tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tasks []*objects.TaskObject
	for rows.Next() {
		var (
			idStr      string
			kind       string
			createdAt  string
			updatedAt  string
			createdBy  string
			aclJSON    string
			tagsJSON   sql.NullString
			fieldsJSON string
		)

		err := rows.Scan(&idStr, &kind, &createdAt, &updatedAt, &createdBy, &aclJSON, &tagsJSON, &fieldsJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		task, err := parseTaskObject(idStr, kind, createdAt, updatedAt, createdBy, aclJSON, tagsJSON, fieldsJSON)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// ListDueSoonTasks returns tasks due within the specified number of days.
func ListDueSoonTasks(db *sql.DB, days int) ([]*objects.TaskObject, error) {
	now := time.Now().UTC()
	threshold := now.Add(time.Duration(days) * 24 * time.Hour)

	rows, err := db.Query(`
		SELECT id, kind, created_at, updated_at, created_by, acl, tags, fields
		FROM objects
		WHERE kind = ?
			AND json_extract(fields, '$.dueAt') IS NOT NULL
			AND json_extract(fields, '$.dueAt') > ?
			AND json_extract(fields, '$.dueAt') <= ?
			AND json_extract(fields, '$.status') NOT IN (?, ?)
		ORDER BY json_extract(fields, '$.dueAt') ASC
	`, objects.KindTask, now.Format(time.RFC3339), threshold.Format(time.RFC3339),
		objects.TaskStatusDone, objects.TaskStatusCancelled)

	if err != nil {
		return nil, fmt.Errorf("failed to query due soon tasks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tasks []*objects.TaskObject
	for rows.Next() {
		var (
			idStr      string
			kind       string
			createdAt  string
			updatedAt  string
			createdBy  string
			aclJSON    string
			tagsJSON   sql.NullString
			fieldsJSON string
		)

		err := rows.Scan(&idStr, &kind, &createdAt, &updatedAt, &createdBy, &aclJSON, &tagsJSON, &fieldsJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		task, err := parseTaskObject(idStr, kind, createdAt, updatedAt, createdBy, aclJSON, tagsJSON, fieldsJSON)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// ListTasksByRelatedRecord returns tasks linked to a specific record.
func ListTasksByRelatedRecord(db *sql.DB, recordID uuid.UUID) ([]*objects.TaskObject, error) {
	// SQLite JSON array contains check
	rows, err := db.Query(`
		SELECT objects.id, objects.kind, objects.created_at, objects.updated_at, objects.created_by, objects.acl, objects.tags, objects.fields
		FROM objects, json_each(json_extract(objects.fields, '$.relatedRecordIds'))
		WHERE objects.kind = ? AND json_each.value = ?
		ORDER BY objects.created_at DESC
	`, objects.KindTask, recordID.String())

	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by related record: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tasks []*objects.TaskObject
	for rows.Next() {
		var (
			idStr      string
			kind       string
			createdAt  string
			updatedAt  string
			createdBy  string
			aclJSON    string
			tagsJSON   sql.NullString
			fieldsJSON string
		)

		err := rows.Scan(&idStr, &kind, &createdAt, &updatedAt, &createdBy, &aclJSON, &tagsJSON, &fieldsJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		task, err := parseTaskObject(idStr, kind, createdAt, updatedAt, createdBy, aclJSON, tagsJSON, fieldsJSON)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func parseTaskObject(idStr, kind, createdAtStr, updatedAtStr, createdByStr, aclJSON string, tagsJSON sql.NullString, fieldsJSON string) (*objects.TaskObject, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task ID: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	updatedAt, err := time.Parse(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse updated_at: %w", err)
	}

	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_by: %w", err)
	}

	var acl []objects.ACLEntry
	if err := json.Unmarshal([]byte(aclJSON), &acl); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ACL: %w", err)
	}

	var tags []string
	if tagsJSON.Valid {
		if err := json.Unmarshal([]byte(tagsJSON.String), &tags); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
		}
	}

	var fields map[string]interface{}
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fields: %w", err)
	}

	return &objects.TaskObject{
		BaseObject: objects.BaseObject{
			ID:        id,
			Kind:      kind,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			CreatedBy: createdBy,
			ACL:       acl,
			Tags:      tags,
			Fields:    fields,
		},
	}, nil
}
