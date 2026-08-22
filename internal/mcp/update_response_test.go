// ABOUTME: Verifies MCP update responses belong to the request that performed the write.
// ABOUTME: Stresses real SDK transport and SQLite under concurrent updates and deletes.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/harperreed/crm/v2/internal/history"
	"github.com/harperreed/crm/v2/internal/models"
	"github.com/harperreed/crm/v2/internal/storage"
)

type updateResponseEntity struct {
	ID     uuid.UUID      `json:"ID"`
	Name   string         `json:"Name"`
	Fields map[string]any `json:"Fields"`
}

type updateResponseTools struct {
	name       string
	addTool    string
	updateTool string
	deleteTool string
	entityType history.EntityType
}

type committedUpdateGateStore struct {
	storage.Storage
	entityType history.EntityType
	committed  chan struct{}
	release    chan struct{}
}

type asyncToolResult struct {
	result *sdkmcp.CallToolResult
	err    error
}

func TestMeaningfulUpdateReturnsRequestStateTransport(t *testing.T) {
	for _, tools := range updateResponseToolCases() {
		t.Run(tools.name, func(t *testing.T) {
			ctx, session := openUpdateResponseSession(t)
			created := callUpdateResponseEntity(t, ctx, session, tools.addTool, map[string]any{"name": "Original"})
			historyArgs := map[string]any{"entity_id": created.ID.String()}
			before := listHistoryThroughTransport(t, ctx, session, historyArgs)

			updated := callUpdateResponseEntity(t, ctx, session, tools.updateTool, map[string]any{
				"id":     created.ID.String(),
				"name":   "Owned Response",
				"fields": map[string]any{"request": tools.name},
			})
			if updated.Name != "Owned Response" || updated.Fields["request"] != tools.name {
				t.Errorf("%s response = name:%q fields:%#v, want this request's values", tools.updateTool, updated.Name, updated.Fields)
			}
			after := listHistoryThroughTransport(t, ctx, session, historyArgs)
			if len(after) != len(before)+1 {
				t.Errorf("%s history count = %d, want %d", tools.updateTool, len(after), len(before)+1)
			}
		})
	}
}

func TestConcurrentUpdateResponsesBelongToRequestsTransport(t *testing.T) {
	for _, tools := range updateResponseToolCases() {
		t.Run(tools.name, func(t *testing.T) {
			ctx, session := openUpdateResponseSession(t)
			for round := 0; round < 2; round++ {
				created := callUpdateResponseEntity(t, ctx, session, tools.addTool, map[string]any{"name": fmt.Sprintf("Round %d", round)})
				stressConcurrentUpdates(t, ctx, session, tools.updateTool, created.ID, round)
			}
		})
	}
}

func TestConcurrentUpdateDeleteDoesNotReportCommittedUpdateAsErrorTransport(t *testing.T) {
	for _, tools := range updateResponseToolCases() {
		t.Run(tools.name, func(t *testing.T) {
			ctx, session, gate := openCommittedUpdateGateSession(t, tools.entityType)
			created := callUpdateResponseEntity(t, ctx, session, tools.addTool, map[string]any{"name": "Delete Race"})
			name := "Committed Before Delete"
			updateDone := make(chan asyncToolResult, 1)
			go func() {
				result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
					Name:      tools.updateTool,
					Arguments: map[string]any{"id": created.ID.String(), "name": name},
				})
				updateDone <- asyncToolResult{result: result, err: err}
			}()

			select {
			case <-gate.committed:
			case update := <-updateDone:
				if update.err != nil {
					t.Fatalf("%s returned before storage commit: %v", tools.updateTool, update.err)
				}
				if update.result == nil {
					t.Fatalf("%s returned no result before storage commit", tools.updateTool)
				}
				t.Fatalf("%s returned before storage commit: %s", tools.updateTool, contentText(update.result))
			}
			deleteResult, deleteErr := session.CallTool(ctx, &sdkmcp.CallToolParams{
				Name:      tools.deleteTool,
				Arguments: map[string]any{"id": created.ID.String()},
			})
			close(gate.release)
			update := <-updateDone

			if deleteErr != nil {
				t.Fatalf("%s transport error after committed update: %v", tools.deleteTool, deleteErr)
			}
			if deleteResult.IsError {
				t.Fatalf("%s failed after committed update: %s", tools.deleteTool, contentText(deleteResult))
			}
			if update.err != nil {
				t.Fatalf("%s transport error after its update committed: %v", tools.updateTool, update.err)
			}
			if update.result.IsError {
				t.Fatalf("%s reported error after its update committed: %s", tools.updateTool, contentText(update.result))
			}
			entity, err := parseUpdateResponseEntity(update.result)
			if err != nil {
				t.Fatalf("parse successful %s response: %v", tools.updateTool, err)
			}
			if entity.Name != name {
				t.Errorf("successful %s request %q returned %q", tools.updateTool, name, entity.Name)
			}

			summaries := listHistoryThroughTransport(t, ctx, session, map[string]any{"entity_id": created.ID.String()})
			assertUpdateDeleteHistory(t, summaries)
		})
	}
}

func updateResponseToolCases() []updateResponseTools {
	return []updateResponseTools{
		{name: "contact", addTool: "add_contact", updateTool: "update_contact", deleteTool: "delete_contact", entityType: history.EntityContact},
		{name: "company", addTool: "add_company", updateTool: "update_company", deleteTool: "delete_company", entityType: history.EntityCompany},
	}
}

func openUpdateResponseSession(t *testing.T) (context.Context, *sdkmcp.ClientSession) {
	t.Helper()
	ctx := context.Background()
	store, err := storage.NewSqliteStore(filepath.Join(t.TempDir(), "responses.db"), history.SourceMCP)
	if err != nil {
		t.Fatalf("NewSqliteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return ctx, connectHistoryTestServer(t, ctx, NewServer(store))
}

func openCommittedUpdateGateSession(t *testing.T, entityType history.EntityType) (context.Context, *sdkmcp.ClientSession, *committedUpdateGateStore) {
	t.Helper()
	ctx := context.Background()
	store, err := storage.NewSqliteStore(filepath.Join(t.TempDir(), "committed-update.db"), history.SourceMCP)
	if err != nil {
		t.Fatalf("NewSqliteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	gate := &committedUpdateGateStore{
		Storage:    store,
		entityType: entityType,
		committed:  make(chan struct{}),
		release:    make(chan struct{}),
	}
	return ctx, connectHistoryTestServer(t, ctx, NewServer(gate)), gate
}

func (s *committedUpdateGateStore) UpdateContact(contact *models.Contact) error {
	err := s.Storage.UpdateContact(contact)
	if err != nil || s.entityType != history.EntityContact {
		return err
	}
	close(s.committed)
	<-s.release
	return nil
}

func (s *committedUpdateGateStore) UpdateCompany(company *models.Company) error {
	err := s.Storage.UpdateCompany(company)
	if err != nil || s.entityType != history.EntityCompany {
		return err
	}
	close(s.committed)
	<-s.release
	return nil
}

func callUpdateResponseEntity(t *testing.T, ctx context.Context, session *sdkmcp.ClientSession, tool string, arguments map[string]any) updateResponseEntity {
	t.Helper()
	result := callHistoryTool(t, ctx, session, tool, arguments)
	entity, err := parseUpdateResponseEntity(result)
	if err != nil {
		t.Fatalf("parse %s response: %v", tool, err)
	}
	return entity
}

func parseUpdateResponseEntity(result *sdkmcp.CallToolResult) (updateResponseEntity, error) {
	var entity updateResponseEntity
	if err := json.Unmarshal([]byte(contentText(result)), &entity); err != nil {
		return entity, err
	}
	return entity, nil
}

func stressConcurrentUpdates(t *testing.T, ctx context.Context, session *sdkmcp.ClientSession, updateTool string, entityID uuid.UUID, round int) {
	t.Helper()
	const requests = 8
	start := make(chan struct{})
	results := make(chan string, requests)
	var successes atomic.Int32
	var wait sync.WaitGroup
	for request := 0; request < requests; request++ {
		name := fmt.Sprintf("round-%d-request-%d", round, request)
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
				Name:      updateTool,
				Arguments: map[string]any{"id": entityID.String(), "name": name},
			})
			if err != nil || result.IsError {
				return
			}
			successes.Add(1)
			entity, err := parseUpdateResponseEntity(result)
			if err != nil {
				results <- fmt.Sprintf("request %q returned invalid entity: %v", name, err)
				return
			}
			if entity.Name != name {
				results <- fmt.Sprintf("request %q returned concurrent state %q", name, entity.Name)
			}
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	for failure := range results {
		t.Error(failure)
	}
	if successes.Load() == 0 {
		t.Error("concurrent update exercise had no successful updates")
	}
}

func assertUpdateDeleteHistory(t *testing.T, summaries []*history.Summary) {
	t.Helper()
	if len(summaries) != 3 {
		t.Fatalf("history event count = %d, want create/update/delete", len(summaries))
	}
	want := []history.Action{history.ActionDelete, history.ActionUpdate, history.ActionCreate}
	for index, summary := range summaries {
		if summary.Action != want[index] {
			t.Errorf("history[%d].Action = %q, want %q", index, summary.Action, want[index])
		}
	}
}
