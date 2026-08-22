// ABOUTME: Exercises CRM history tools through a real in-memory MCP transport.
// ABOUTME: Verifies MCP-sourced contact changes retain exact event snapshots.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/uuid"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/harperreed/crm/v2/internal/history"
	"github.com/harperreed/crm/v2/internal/models"
	"github.com/harperreed/crm/v2/internal/storage"
)

func TestHistoryToolsTransport(t *testing.T) {
	ctx := context.Background()
	store, err := storage.NewSqliteStore(filepath.Join(t.TempDir(), "history.db"), history.SourceMCP)
	if err != nil {
		t.Fatalf("NewSqliteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	clientSession := connectHistoryTestServer(t, ctx, NewServer(store))

	created := callContactTool(t, ctx, clientSession, "add_contact", map[string]any{
		"name":   "Grace Hopper",
		"email":  "grace@example.com",
		"phone":  "+1-555-0100",
		"fields": map[string]any{"rank": "rear admiral"},
		"tags":   []string{"compiler", "navy"},
	})
	createdSnapshot, err := history.SnapshotContact(contactInUTC(created))
	if err != nil {
		t.Fatalf("SnapshotContact(created): %v", err)
	}

	updated := callContactTool(t, ctx, clientSession, "update_contact", map[string]any{
		"id":    created.ID.String(),
		"email": "hopper@example.com",
		"fields": map[string]any{
			"language": "COBOL",
		},
	})
	updatedSnapshot, err := history.SnapshotContact(contactInUTC(updated))
	if err != nil {
		t.Fatalf("SnapshotContact(updated): %v", err)
	}

	summaries := listHistoryThroughTransport(t, ctx, clientSession, map[string]any{
		"entity_id": created.ID.String(),
		"limit":     2,
	})
	assertHistorySummaries(t, summaries, created.ID)
	event := getHistoryEventThroughTransport(t, ctx, clientSession, summaries[0].ID.String())
	assertUpdateHistoryEvent(t, event, summaries[0].ID, created.ID, createdSnapshot, updatedSnapshot)
	assertHistoryLimitBehavior(t, ctx, clientSession, created.ID)
	assertEmptyHistoryIDsRejected(t, ctx, clientSession)
}

func connectHistoryTestServer(t *testing.T, ctx context.Context, server *Server) *sdkmcp.ClientSession {
	t.Helper()
	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	serverSession, err := server.server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "crm-history-test", Version: "1.0.0"},
		nil,
	)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })
	return clientSession
}

func listHistoryThroughTransport(t *testing.T, ctx context.Context, session *sdkmcp.ClientSession, arguments map[string]any) []*history.Summary {
	t.Helper()
	result := callHistoryTool(t, ctx, session, "list_history", arguments)
	var summaries []*history.Summary
	if err := parseContent(result, &summaries); err != nil {
		t.Fatalf("parse list_history: %v", err)
	}
	return summaries
}

func getHistoryEventThroughTransport(t *testing.T, ctx context.Context, session *sdkmcp.ClientSession, eventID string) *history.Event {
	t.Helper()
	result := callHistoryTool(t, ctx, session, "get_history_event", map[string]any{"event_id": eventID})
	var event history.Event
	if err := parseContent(result, &event); err != nil {
		t.Fatalf("parse get_history_event: %v", err)
	}
	return &event
}

func assertHistorySummaries(t *testing.T, summaries []*history.Summary, entityID uuid.UUID) {
	t.Helper()
	if len(summaries) != 2 {
		t.Fatalf("list_history returned %d events, want 2", len(summaries))
	}
	if summaries[0].Action != history.ActionUpdate || summaries[1].Action != history.ActionCreate {
		t.Fatalf("list_history actions = [%q, %q], want [update, create]", summaries[0].Action, summaries[1].Action)
	}
	for index, summary := range summaries {
		if summary.Source != history.SourceMCP {
			t.Errorf("summary %d source = %q, want %q", index, summary.Source, history.SourceMCP)
		}
		if summary.EntityID != entityID {
			t.Errorf("summary %d entity ID = %s, want %s", index, summary.EntityID, entityID)
		}
	}
}

func assertUpdateHistoryEvent(t *testing.T, event *history.Event, eventID, entityID uuid.UUID, before, after json.RawMessage) {
	t.Helper()
	if err := event.Validate(); err != nil {
		t.Errorf("history event validation: %v", err)
	}
	if event.ID != eventID || event.EntityID != entityID {
		t.Errorf("event identity = (%s, %s), want (%s, %s)", event.ID, event.EntityID, eventID, entityID)
	}
	if want := []uuid.UUID{entityID}; !slices.Equal(event.RelatedEntityIDs, want) {
		t.Errorf("event related entity IDs = %v, want %v", event.RelatedEntityIDs, want)
	}
	if event.Action != history.ActionUpdate || event.Source != history.SourceMCP {
		t.Errorf("event action/source = %q/%q, want update/mcp", event.Action, event.Source)
	}
	assertExactContactSnapshot(t, "before", event.Before, before)
	assertExactContactSnapshot(t, "after", event.After, after)
}

func assertHistoryLimitBehavior(t *testing.T, ctx context.Context, session *sdkmcp.ClientSession, entityID uuid.UUID) {
	t.Helper()
	for index := 0; index < history.DefaultLimit-1; index++ {
		callContactTool(t, ctx, session, "update_contact", map[string]any{
			"id":    entityID.String(),
			"phone": fmt.Sprintf("+1-555-%04d", index+200),
		})
	}
	summaries := listHistoryThroughTransport(t, ctx, session, map[string]any{"entity_id": entityID.String()})
	if len(summaries) != history.DefaultLimit {
		t.Errorf("list_history with omitted limit returned %d events, want default %d", len(summaries), history.DefaultLimit)
	}

	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "list_history",
		Arguments: map[string]any{
			"entity_id": entityID.String(),
			"limit":     history.MaxLimit + 1,
		},
	})
	if err != nil {
		t.Fatalf("CallTool list_history with excessive limit: %v", err)
	}
	if !result.IsError {
		t.Fatal("list_history with excessive limit returned success")
	}
	if got, want := contentText(result), "list history: history limit must not exceed 100"; got != want {
		t.Errorf("list_history excessive-limit error = %q, want %q", got, want)
	}
}

func assertEmptyHistoryIDsRejected(t *testing.T, ctx context.Context, session *sdkmcp.ClientSession) {
	t.Helper()
	for _, test := range []struct {
		name      string
		arguments map[string]any
		want      string
	}{
		{name: "list_history", arguments: map[string]any{"entity_id": ""}, want: "entity_id is required"},
		{name: "list_history", arguments: map[string]any{"entity_id": " \t"}, want: "entity_id is required"},
		{name: "get_history_event", arguments: map[string]any{"event_id": ""}, want: "event_id is required"},
		{name: "get_history_event", arguments: map[string]any{"event_id": " \t"}, want: "event_id is required"},
	} {
		result, callErr := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: test.name, Arguments: test.arguments})
		if callErr != nil {
			t.Fatalf("CallTool %s with empty ID: %v", test.name, callErr)
		}
		if !result.IsError {
			t.Errorf("%s with empty ID returned success", test.name)
		}
		if got := contentText(result); got != test.want {
			t.Errorf("%s with ID %q error = %q, want %q", test.name, test.arguments, got, test.want)
		}
	}
}

func contactInUTC(contact *models.Contact) *models.Contact {
	normalized := *contact
	normalized.CreatedAt = contact.CreatedAt.UTC()
	normalized.UpdatedAt = contact.UpdatedAt.UTC()
	return &normalized
}

func assertExactContactSnapshot(t *testing.T, name string, got, want json.RawMessage) {
	t.Helper()
	equal, err := history.EqualSnapshots(history.EntityContact, got, want)
	if err != nil {
		t.Fatalf("compare %s snapshot: %v", name, err)
	}
	if !equal {
		t.Errorf("event %s = %s, want exact snapshot %s", name, got, want)
	}
}

func callContactTool(t *testing.T, ctx context.Context, session *sdkmcp.ClientSession, name string, arguments map[string]any) *models.Contact {
	t.Helper()
	result := callHistoryTool(t, ctx, session, name, arguments)
	var contact models.Contact
	if err := json.Unmarshal([]byte(contentText(result)), &contact); err != nil {
		t.Fatalf("parse %s contact: %v", name, err)
	}
	return &contact
}

func callHistoryTool(t *testing.T, ctx context.Context, session *sdkmcp.ClientSession, name string, arguments map[string]any) *sdkmcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	if result.IsError {
		t.Fatalf("%s returned error: %s", name, contentText(result))
	}
	return result
}
