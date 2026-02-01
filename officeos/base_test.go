// ABOUTME: Tests for Office OS base types
// ABOUTME: Covers BaseObject, ACL, and JSON serialization
package officeos

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBaseObjectFieldsAsJSON(t *testing.T) {
	obj := &BaseObject{
		ID:        "test-1",
		Kind:      KindRecord,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Fields: map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
		},
	}

	jsonStr, err := obj.FieldsAsJSON()
	if err != nil {
		t.Fatalf("FieldsAsJSON failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("Expected non-empty JSON string")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}

	if parsed["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", parsed["name"])
	}
}

func TestBaseObjectFieldsAsJSONWithNilFields(t *testing.T) {
	obj := &BaseObject{
		ID:     "test-1",
		Kind:   KindRecord,
		Fields: nil,
	}

	jsonStr, err := obj.FieldsAsJSON()
	if err != nil {
		t.Fatalf("FieldsAsJSON failed: %v", err)
	}

	if jsonStr != "null" {
		t.Errorf("Expected 'null', got %s", jsonStr)
	}
}

func TestBaseObjectACLAsJSON(t *testing.T) {
	obj := &BaseObject{
		ID:   "test-1",
		Kind: KindRecord,
		ACL: []ACLEntry{
			{ActorID: "user-1", Role: "owner"},
			{ActorID: "user-2", Role: "viewer"},
		},
	}

	jsonStr, err := obj.ACLAsJSON()
	if err != nil {
		t.Fatalf("ACLAsJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var parsed []ACLEntry
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}

	if len(parsed) != 2 {
		t.Errorf("Expected 2 ACL entries, got %d", len(parsed))
	}
}

func TestBaseObjectACLAsJSONEmpty(t *testing.T) {
	obj := &BaseObject{
		ID:   "test-1",
		Kind: KindRecord,
		ACL:  []ACLEntry{},
	}

	jsonStr, err := obj.ACLAsJSON()
	if err != nil {
		t.Fatalf("ACLAsJSON failed: %v", err)
	}

	if jsonStr != "[]" {
		t.Errorf("Expected '[]', got %s", jsonStr)
	}
}

func TestBaseObjectTagsAsJSON(t *testing.T) {
	obj := &BaseObject{
		ID:   "test-1",
		Kind: KindRecord,
		Tags: []string{"important", "work", "urgent"},
	}

	jsonStr, err := obj.TagsAsJSON()
	if err != nil {
		t.Fatalf("TagsAsJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var parsed []string
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}

	if len(parsed) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(parsed))
	}
}

func TestBaseObjectTagsAsJSONEmpty(t *testing.T) {
	obj := &BaseObject{
		ID:   "test-1",
		Kind: KindRecord,
		Tags: []string{},
	}

	jsonStr, err := obj.TagsAsJSON()
	if err != nil {
		t.Fatalf("TagsAsJSON failed: %v", err)
	}

	if jsonStr != "[]" {
		t.Errorf("Expected '[]', got %s", jsonStr)
	}
}

func TestBaseObjectTagsAsJSONNil(t *testing.T) {
	obj := &BaseObject{
		ID:   "test-1",
		Kind: KindRecord,
		Tags: nil,
	}

	jsonStr, err := obj.TagsAsJSON()
	if err != nil {
		t.Fatalf("TagsAsJSON failed: %v", err)
	}

	if jsonStr != "[]" {
		t.Errorf("Expected '[]' for nil tags, got %s", jsonStr)
	}
}

func TestObjectKindConstants(t *testing.T) {
	// Test that constants are defined correctly
	tests := []struct {
		kind     ObjectKind
		expected string
	}{
		{KindUser, "user"},
		{KindRecord, "record"},
		{KindTask, "task"},
		{KindEvent, "event"},
		{KindMessage, "message"},
		{KindActivity, "activity"},
		{KindNotification, "notification"},
	}

	for _, tt := range tests {
		if string(tt.kind) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, tt.kind)
		}
	}
}

func TestACLEntry(t *testing.T) {
	entry := ACLEntry{
		ActorID: "user-123",
		Role:    "admin",
	}

	if entry.ActorID != "user-123" {
		t.Errorf("Expected ActorID 'user-123', got %s", entry.ActorID)
	}

	if entry.Role != "admin" {
		t.Errorf("Expected Role 'admin', got %s", entry.Role)
	}
}
