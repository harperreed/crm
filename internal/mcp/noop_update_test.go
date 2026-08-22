// ABOUTME: Regresses no-op MCP updates against persisted contact and company state.
// ABOUTME: Uses the real in-memory SDK transport and SQLite history backend.
package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/harperreed/crm/internal/history"
	"github.com/harperreed/crm/internal/storage"
)

func TestNoOpUpdateReturnsPersistedStateTransport(t *testing.T) {
	for _, test := range []struct {
		name       string
		addTool    string
		addArgs    map[string]any
		updateTool string
		getTool    string
	}{
		{
			name:       "contact",
			addTool:    "add_contact",
			addArgs:    map[string]any{"name": "No-op Contact", "email": "same@example.com"},
			updateTool: "update_contact",
			getTool:    "get_contact",
		},
		{
			name:       "company",
			addTool:    "add_company",
			addArgs:    map[string]any{"name": "No-op Company", "domain": "same.example"},
			updateTool: "update_company",
			getTool:    "get_company",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertNoOpUpdateReturnsPersistedState(t, test.addTool, test.addArgs, test.updateTool, test.getTool)
		})
	}
}

func assertNoOpUpdateReturnsPersistedState(t *testing.T, addTool string, addArgs map[string]any, updateTool, getTool string) {
	t.Helper()
	ctx := context.Background()
	store, err := storage.NewSqliteStore(filepath.Join(t.TempDir(), "noop.db"), history.SourceMCP)
	if err != nil {
		t.Fatalf("NewSqliteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	session := connectHistoryTestServer(t, ctx, NewServer(store))

	addResult := callHistoryTool(t, ctx, session, addTool, addArgs)
	var created struct {
		ID uuid.UUID `json:"ID"`
	}
	if err := json.Unmarshal([]byte(contentText(addResult)), &created); err != nil {
		t.Fatalf("parse %s response: %v", addTool, err)
	}

	getArgs := map[string]any{"id": created.ID.String()}
	persistedBefore := contentText(callHistoryTool(t, ctx, session, getTool, getArgs))
	historyArgs := map[string]any{"entity_id": created.ID.String()}
	historyBefore := contentText(callHistoryTool(t, ctx, session, "list_history", historyArgs))

	updateResult := callHistoryTool(t, ctx, session, updateTool, getArgs)
	persistedAfter := contentText(callHistoryTool(t, ctx, session, getTool, getArgs))
	historyAfter := contentText(callHistoryTool(t, ctx, session, "list_history", historyArgs))

	if got := contentText(updateResult); got != persistedAfter {
		t.Errorf("%s response differs from persisted state:\nresponse: %s\npersisted: %s", updateTool, got, persistedAfter)
	}
	if persistedAfter != persistedBefore {
		t.Errorf("%s no-op changed persisted state:\nbefore: %s\nafter: %s", updateTool, persistedBefore, persistedAfter)
	}
	if historyAfter != historyBefore {
		t.Errorf("%s no-op changed history:\nbefore: %s\nafter: %s", updateTool, historyBefore, historyAfter)
	}
}
