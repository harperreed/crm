// ABOUTME: Tests for the MCP server creation and tool execution via in-memory transport.
// ABOUTME: Validates server construction, tool availability, and basic CRUD tool round-trips.
package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/harperreed/crm/internal/storage"
)

// newTestStore creates a temporary SQLite store for testing.
func newTestStore(t *testing.T) storage.Storage {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := storage.NewSqliteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSqliteStore(%q): %v", dbPath, err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// connectTestServer creates a Server and connects a client via in-memory transport.
func connectTestServer(t *testing.T, store storage.Storage) *mcp.ClientSession {
	t.Helper()
	srv := NewServer(store)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	// Connect server (must happen before client).
	go func() {
		_ = srv.server.Run(context.Background(), serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.1.0"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestNewServer(t *testing.T) {
	store := newTestStore(t)
	srv := NewServer(store)

	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.server == nil {
		t.Fatal("expected non-nil mcp.Server")
	}
	if srv.store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestServerListTools(t *testing.T) {
	store := newTestStore(t)
	session := connectTestServer(t, store)

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	expectedTools := []string{
		"add_contact", "list_contacts", "get_contact", "update_contact", "delete_contact",
		"add_company", "list_companies", "get_company", "update_company", "delete_company",
		"link", "unlink",
	}

	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}

	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("expected tool %q not found", name)
		}
	}

	if len(result.Tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(result.Tools))
	}
}

func TestServerAddAndGetContact(t *testing.T) {
	store := newTestStore(t)
	session := connectTestServer(t, store)
	ctx := context.Background()

	// Add a contact via tool call.
	addResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add_contact",
		Arguments: map[string]any{
			"name":  "Ada Lovelace",
			"email": "ada@example.com",
			"tags":  []string{"pioneer", "math"},
		},
	})
	if err != nil {
		t.Fatalf("CallTool add_contact: %v", err)
	}
	if addResult.IsError {
		t.Fatalf("add_contact returned error: %s", contentText(addResult))
	}

	// List contacts and verify the one we added is there.
	listResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_contacts",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool list_contacts: %v", err)
	}
	if listResult.IsError {
		t.Fatalf("list_contacts returned error: %s", contentText(listResult))
	}

	text := contentText(listResult)
	if text == "" || text == "[]" || text == "null" {
		t.Error("expected non-empty contact list")
	}
}

func TestServerAddAndGetCompany(t *testing.T) {
	store := newTestStore(t)
	session := connectTestServer(t, store)
	ctx := context.Background()

	addResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add_company",
		Arguments: map[string]any{
			"name":   "Babbage Inc",
			"domain": "babbage.io",
		},
	})
	if err != nil {
		t.Fatalf("CallTool add_company: %v", err)
	}
	if addResult.IsError {
		t.Fatalf("add_company returned error: %s", contentText(addResult))
	}

	listResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_companies",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool list_companies: %v", err)
	}
	if listResult.IsError {
		t.Fatalf("list_companies returned error: %s", contentText(listResult))
	}

	text := contentText(listResult)
	if text == "" || text == "[]" || text == "null" {
		t.Error("expected non-empty company list")
	}
}

func TestServerDeleteContact(t *testing.T) {
	store := newTestStore(t)
	session := connectTestServer(t, store)
	ctx := context.Background()

	// Add then delete.
	addResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "add_contact",
		Arguments: map[string]any{"name": "Temp Contact"},
	})
	if err != nil {
		t.Fatalf("add_contact: %v", err)
	}
	if addResult.IsError {
		t.Fatalf("add_contact error: %s", contentText(addResult))
	}

	// Extract the ID from the JSON response.
	type idHolder struct {
		ID string `json:"ID"`
	}
	var holder idHolder
	if err := parseContent(addResult, &holder); err != nil {
		t.Fatalf("parse add result: %v", err)
	}

	delResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "delete_contact",
		Arguments: map[string]any{"id": holder.ID},
	})
	if err != nil {
		t.Fatalf("delete_contact: %v", err)
	}
	if delResult.IsError {
		t.Fatalf("delete_contact error: %s", contentText(delResult))
	}
}

func TestServerLinkAndUnlink(t *testing.T) {
	store := newTestStore(t)
	session := connectTestServer(t, store)
	ctx := context.Background()

	// Create two contacts.
	type idHolder struct {
		ID string `json:"ID"`
	}

	r1, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "add_contact",
		Arguments: map[string]any{"name": "Alice"},
	})
	if err != nil || r1.IsError {
		t.Fatalf("add_contact Alice: err=%v isError=%v", err, r1.IsError)
	}
	var h1 idHolder
	if err := parseContent(r1, &h1); err != nil {
		t.Fatalf("parse Alice: %v", err)
	}

	r2, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "add_contact",
		Arguments: map[string]any{"name": "Bob"},
	})
	if err != nil || r2.IsError {
		t.Fatalf("add_contact Bob: err=%v isError=%v", err, r2.IsError)
	}
	var h2 idHolder
	if err := parseContent(r2, &h2); err != nil {
		t.Fatalf("parse Bob: %v", err)
	}

	// Link them.
	linkResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "link",
		Arguments: map[string]any{
			"source_id": h1.ID,
			"target_id": h2.ID,
			"type":      "knows",
		},
	})
	if err != nil || linkResult.IsError {
		t.Fatalf("link: err=%v isError=%v text=%s", err, linkResult.IsError, contentText(linkResult))
	}

	var relHolder idHolder
	if err := parseContent(linkResult, &relHolder); err != nil {
		t.Fatalf("parse link: %v", err)
	}

	// Unlink.
	unlinkResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "unlink",
		Arguments: map[string]any{"id": relHolder.ID},
	})
	if err != nil || unlinkResult.IsError {
		t.Fatalf("unlink: err=%v isError=%v text=%s", err, unlinkResult.IsError, contentText(unlinkResult))
	}
}

func TestServerListPrompts(t *testing.T) {
	store := newTestStore(t)
	session := connectTestServer(t, store)

	result, err := session.ListPrompts(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}

	expectedPrompts := []string{"add-contact-workflow", "relationship-mapping", "crm-search"}
	promptNames := make(map[string]bool)
	for _, p := range result.Prompts {
		promptNames[p.Name] = true
	}
	for _, name := range expectedPrompts {
		if !promptNames[name] {
			t.Errorf("expected prompt %q not found", name)
		}
	}
}

func TestServerListResourceTemplates(t *testing.T) {
	store := newTestStore(t)
	session := connectTestServer(t, store)

	result, err := session.ListResourceTemplates(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListResourceTemplates: %v", err)
	}

	expected := []string{"crm://contacts/{id}", "crm://companies/{id}"}
	templateURIs := make(map[string]bool)
	for _, tmpl := range result.ResourceTemplates {
		templateURIs[tmpl.URITemplate] = true
	}
	for _, uri := range expected {
		if !templateURIs[uri] {
			t.Errorf("expected resource template %q not found", uri)
		}
	}
}

func TestServerUpdateContact(t *testing.T) {
	store := newTestStore(t)
	session := connectTestServer(t, store)
	ctx := context.Background()

	// Create a contact.
	addResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "add_contact",
		Arguments: map[string]any{
			"name":  "Original Name",
			"email": "original@example.com",
		},
	})
	if err != nil || addResult.IsError {
		t.Fatalf("add_contact: err=%v isError=%v", err, addResult.IsError)
	}
	type idHolder struct {
		ID string `json:"ID"`
	}
	var holder idHolder
	if err := parseContent(addResult, &holder); err != nil {
		t.Fatalf("parse add: %v", err)
	}

	// Update the contact.
	updateResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "update_contact",
		Arguments: map[string]any{
			"id":   holder.ID,
			"name": "Updated Name",
		},
	})
	if err != nil || updateResult.IsError {
		t.Fatalf("update_contact: err=%v isError=%v text=%s", err, updateResult.IsError, contentText(updateResult))
	}

	// Verify the update.
	type contactData struct {
		Name  string `json:"Name"`
		Email string `json:"Email"`
	}
	var updated contactData
	if err := parseContent(updateResult, &updated); err != nil {
		t.Fatalf("parse update: %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Errorf("expected name %q, got %q", "Updated Name", updated.Name)
	}
	if updated.Email != "original@example.com" {
		t.Errorf("expected email preserved as %q, got %q", "original@example.com", updated.Email)
	}
}

func TestServerErrorOnMissingRequired(t *testing.T) {
	store := newTestStore(t)
	session := connectTestServer(t, store)
	ctx := context.Background()

	// add_contact without name should fail.
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "add_contact",
		Arguments: map[string]any{"email": "no-name@example.com"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for add_contact without name")
	}
}

// --- test helpers ---

// contentText extracts the text from the first content block of a CallToolResult.
func contentText(r *mcp.CallToolResult) string {
	if len(r.Content) == 0 {
		return ""
	}
	if tc, ok := r.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

// parseContent unmarshals the first text content block into the given struct.
func parseContent(r *mcp.CallToolResult, v any) error {
	text := contentText(r)
	if text == "" {
		return nil
	}
	return json.Unmarshal([]byte(text), v)
}
