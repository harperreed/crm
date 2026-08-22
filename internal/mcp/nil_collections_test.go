// ABOUTME: Regresses MCP updates of legacy SQLite rows whose JSON collections are null.
// ABOUTME: Uses real SDK transport and checks reads, writes, and history without mocks.
package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/harperreed/crm/internal/history"
	"github.com/harperreed/crm/internal/storage"
)

func TestLegacyNilCollectionsTransport(t *testing.T) {
	for _, test := range []struct {
		name       string
		getTool    string
		updateTool string
		seed       func(*testing.T, *sql.DB, uuid.UUID, time.Time)
	}{
		{name: "contact", getTool: "get_contact", updateTool: "update_contact", seed: seedLegacyNilContact},
		{name: "company", getTool: "get_company", updateTool: "update_company", seed: seedLegacyNilCompany},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			store, entityID := openLegacyNilCollectionStore(t, test.seed)
			session := connectHistoryTestServer(t, ctx, NewServer(store))
			entityArgs := map[string]any{"id": entityID.String()}

			assertTransportCollectionsCanonical(t, callHistoryTool(t, ctx, session, test.getTool, entityArgs))
			if summaries := listHistoryThroughTransport(t, ctx, session, map[string]any{"entity_id": entityID.String()}); len(summaries) != 0 {
				t.Fatalf("legacy read fabricated %d history events, want 0", len(summaries))
			}

			updateArgs := map[string]any{
				"id":     entityID.String(),
				"fields": map[string]any{"source": "legacy"},
			}
			updated := callHistoryTool(t, ctx, session, test.updateTool, updateArgs)
			assertTransportCollectionsCanonical(t, updated)
			var entity struct {
				Fields map[string]any `json:"Fields"`
			}
			if err := json.Unmarshal([]byte(contentText(updated)), &entity); err != nil {
				t.Fatalf("parse %s response: %v", test.updateTool, err)
			}
			if entity.Fields["source"] != "legacy" {
				t.Errorf("%s Fields = %#v, want source=legacy", test.updateTool, entity.Fields)
			}
			if summaries := listHistoryThroughTransport(t, ctx, session, map[string]any{"entity_id": entityID.String()}); len(summaries) != 1 {
				t.Fatalf("legacy update history event count = %d, want 1", len(summaries))
			}
		})
	}
}

func openLegacyNilCollectionStore(t *testing.T, seed func(*testing.T, *sql.DB, uuid.UUID, time.Time)) (storage.Storage, uuid.UUID) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "legacy-null.db")
	initial, err := storage.NewSqliteStore(dbPath, history.SourceMCP)
	if err != nil {
		t.Fatalf("initialize SQLite store: %v", err)
	}
	if err := initial.Close(); err != nil {
		t.Fatalf("close initialized SQLite store: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open SQLite seed database: %v", err)
	}
	entityID := uuid.New()
	seed(t, db, entityID, time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC))
	if err := db.Close(); err != nil {
		t.Fatalf("close SQLite seed database: %v", err)
	}

	store, err := storage.NewSqliteStore(dbPath, history.SourceMCP)
	if err != nil {
		t.Fatalf("reopen SQLite store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, entityID
}

func seedLegacyNilContact(t *testing.T, db *sql.DB, id uuid.UUID, timestamp time.Time) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO contacts (id, name, email, phone, fields, tags, created_at, updated_at)
		VALUES (?, ?, '', '', 'null', 'null', ?, ?)`, id.String(), "Legacy Nil Contact", timestamp, timestamp); err != nil {
		t.Fatalf("seed legacy contact: %v", err)
	}
}

func seedLegacyNilCompany(t *testing.T, db *sql.DB, id uuid.UUID, timestamp time.Time) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO companies (id, name, domain, fields, tags, created_at, updated_at)
		VALUES (?, ?, '', 'null', 'null', ?, ?)`, id.String(), "Legacy Nil Company", timestamp, timestamp); err != nil {
		t.Fatalf("seed legacy company: %v", err)
	}
}

func assertTransportCollectionsCanonical(t *testing.T, result *sdkmcp.CallToolResult) {
	t.Helper()
	var entity struct {
		Fields map[string]any `json:"Fields"`
		Tags   []string       `json:"Tags"`
	}
	if err := json.Unmarshal([]byte(contentText(result)), &entity); err != nil {
		t.Fatalf("parse entity response: %v", err)
	}
	if entity.Fields == nil || entity.Tags == nil {
		t.Errorf("transport collections = fields:%#v tags:%#v, want non-nil", entity.Fields, entity.Tags)
	}
}
