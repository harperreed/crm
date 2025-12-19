// ABOUTME: Tests for Charm KV sync view functionality
// ABOUTME: Verifies sync status display and command handling
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/harperreed/pagen/charm"
)

func TestSyncViewRendering(t *testing.T) {
	client := charm.NewTestClient(t)

	// Create model
	m := NewModel(client)
	m.entityType = EntitySync

	// Render the view
	output := m.renderSyncView()

	// Verify basic structure
	if output == "" {
		t.Fatal("Sync view should not be empty")
	}

	// Should contain title
	if !contains(output, "Charm KV Sync") {
		t.Error("Sync view should contain title 'Charm KV Sync'")
	}

	// Should show configuration section
	if !contains(output, "Configuration") {
		t.Error("Sync view should contain Configuration section")
	}
}

func TestSyncViewShowsStatus(t *testing.T) {
	client := charm.NewTestClient(t)

	// Create model
	m := NewModel(client)
	m.entityType = EntitySync

	// Render the view
	output := m.renderSyncView()

	// Should show server
	if !contains(output, "Server:") {
		t.Error("Should show server field")
	}

	// Should show status (linked/unlinked)
	if !contains(output, "Status:") {
		t.Error("Should show status field")
	}

	// Should show auto-sync status
	if !contains(output, "Auto-sync:") {
		t.Error("Should show auto-sync field")
	}
}

func TestSyncKeyNavigation(t *testing.T) {
	client := charm.NewTestClient(t)

	m := NewModel(client)
	m.entityType = EntitySync

	// Test down navigation
	m.selectedService = 0
	updated, _ := m.handleSyncKeys(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.selectedService != 1 {
		t.Errorf("Expected selectedService=1, got %d", m.selectedService)
	}

	// Test up navigation
	updated, _ = m.handleSyncKeys(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.selectedService != 0 {
		t.Errorf("Expected selectedService=0, got %d", m.selectedService)
	}

	// Test escape key goes back
	m.viewMode = ViewList
	m.entityType = EntitySync
	updated, _ = m.handleSyncKeys(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.viewMode != ViewList {
		t.Error("Escape should keep view mode as ViewList")
	}
	if m.entityType == EntitySync {
		t.Error("Escape should change entity type away from EntitySync")
	}
}

func TestSyncCompleteMessage(t *testing.T) {
	client := charm.NewTestClient(t)

	m := NewModel(client)

	// Mark a sync as in progress
	m.syncInProgress["charm"] = true

	// Handle completion
	msg := SyncCompleteMsg{
		Error: nil,
	}

	_ = m.handleSyncComplete(msg)

	// Should no longer be in progress
	if m.syncInProgress["charm"] {
		t.Error("Sync should not be in progress after completion")
	}

	// Should have a message
	if len(m.syncMessages) == 0 {
		t.Error("Should have added a completion message")
	}
}

func TestSyncCompleteWithError(t *testing.T) {
	client := charm.NewTestClient(t)

	m := NewModel(client)

	// Mark a sync as in progress
	m.syncInProgress["charm"] = true

	// Handle completion with error
	msg := SyncCompleteMsg{
		Error: &testError{msg: "test sync error"},
	}

	_ = m.handleSyncComplete(msg)

	// Should no longer be in progress
	if m.syncInProgress["charm"] {
		t.Error("Sync should not be in progress after error")
	}

	// Should have a message
	if len(m.syncMessages) == 0 {
		t.Error("Should have added an error message")
	}
}

func TestAutoSyncToggleMessage(t *testing.T) {
	client := charm.NewTestClient(t)

	m := NewModel(client)

	// Handle toggle enabled
	msg := AutoSyncToggleMsg{
		Enabled: true,
		Error:   nil,
	}

	_ = m.handleAutoSyncToggle(msg)

	// Should have a message
	if len(m.syncMessages) == 0 {
		t.Error("Should have added a toggle message")
	}

	// Handle toggle disabled
	msg = AutoSyncToggleMsg{
		Enabled: false,
		Error:   nil,
	}

	_ = m.handleAutoSyncToggle(msg)

	// Should have two messages now
	if len(m.syncMessages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(m.syncMessages))
	}
}

func TestAutoSyncToggleWithError(t *testing.T) {
	client := charm.NewTestClient(t)

	m := NewModel(client)

	// Handle toggle with error
	msg := AutoSyncToggleMsg{
		Enabled: true,
		Error:   &testError{msg: "failed to toggle"},
	}

	_ = m.handleAutoSyncToggle(msg)

	// Should have an error message
	if len(m.syncMessages) == 0 {
		t.Error("Should have added an error message")
	}
}

func TestSyncMessageAddition(t *testing.T) {
	client := charm.NewTestClient(t)

	m := NewModel(client)

	m.addSyncMessage("Test message 1")
	m.addSyncMessage("Test message 2")

	if len(m.syncMessages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(m.syncMessages))
	}

	// Check that messages contain timestamps
	if !contains(m.syncMessages[0], "Test message 1") {
		t.Error("First message should contain content")
	}
	if !contains(m.syncMessages[1], "Test message 2") {
		t.Error("Second message should contain content")
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
		containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
