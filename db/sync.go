// ABOUTME: Database operations for sync_state and sync_log tables
// ABOUTME: Manages sync status, tokens, and import tracking for external services
package db

import (
	"database/sql"
	"fmt"
	"time"
)

// SyncState represents the sync state for a service
type SyncState struct {
	Service       string
	LastSyncTime  *time.Time
	LastSyncToken *string
	Status        string
	ErrorMessage  *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// GetSyncState retrieves the sync state for a service
func GetSyncState(db *sql.DB, service string) (*SyncState, error) {
	var state SyncState
	var lastSyncTime sql.NullTime
	var lastSyncToken sql.NullString
	var errorMessage sql.NullString

	err := db.QueryRow(`
		SELECT service, last_sync_time, last_sync_token, status, error_message, created_at, updated_at
		FROM sync_state
		WHERE service = ?
	`, service).Scan(
		&state.Service,
		&lastSyncTime,
		&lastSyncToken,
		&state.Status,
		&errorMessage,
		&state.CreatedAt,
		&state.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sync state: %w", err)
	}

	if lastSyncTime.Valid {
		state.LastSyncTime = &lastSyncTime.Time
	}
	if lastSyncToken.Valid {
		state.LastSyncToken = &lastSyncToken.String
	}
	if errorMessage.Valid {
		state.ErrorMessage = &errorMessage.String
	}

	return &state, nil
}

// UpdateSyncStatus updates the sync status for a service
func UpdateSyncStatus(db *sql.DB, service, status string, errorMsg *string) error {
	var errorMsgVal sql.NullString
	if errorMsg != nil {
		errorMsgVal = sql.NullString{String: *errorMsg, Valid: true}
	}

	_, err := db.Exec(`
		INSERT INTO sync_state (service, status, error_message, created_at, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(service) DO UPDATE SET
			status = excluded.status,
			error_message = excluded.error_message,
			updated_at = CURRENT_TIMESTAMP
	`, service, status, errorMsgVal)

	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	return nil
}

// UpdateSyncToken updates the sync token and last sync time for a service
func UpdateSyncToken(db *sql.DB, service, token string) error {
	_, err := db.Exec(`
		INSERT INTO sync_state (service, last_sync_time, last_sync_token, status, created_at, updated_at)
		VALUES (?, CURRENT_TIMESTAMP, ?, 'idle', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(service) DO UPDATE SET
			last_sync_time = CURRENT_TIMESTAMP,
			last_sync_token = excluded.last_sync_token,
			status = 'idle',
			error_message = NULL,
			updated_at = CURRENT_TIMESTAMP
	`, service, token)

	if err != nil {
		return fmt.Errorf("failed to update sync token: %w", err)
	}

	return nil
}

// CheckSyncLogExists checks if an entity has already been imported
func CheckSyncLogExists(db *sql.DB, sourceService, sourceID string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sync_log
		WHERE source_service = ? AND source_id = ?
	`, sourceService, sourceID).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check sync log: %w", err)
	}

	return count > 0, nil
}

// CreateSyncLog creates a sync log entry for an imported entity
func CreateSyncLog(db *sql.DB, id, sourceService, sourceID, entityType, entityID, metadata string) error {
	_, err := db.Exec(`
		INSERT INTO sync_log (id, source_service, source_id, entity_type, entity_id, imported_at, metadata)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?)
	`, id, sourceService, sourceID, entityType, entityID, metadata)

	if err != nil {
		return fmt.Errorf("failed to create sync log: %w", err)
	}

	return nil
}
