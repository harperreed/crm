// ABOUTME: Integration tests for activity generation and timeline
// ABOUTME: Tests the full workflow from object changes to timeline queries
package officeos

import (
	"testing"
	"time"
)

func TestIntegration_CreateObjectGeneratesActivity(t *testing.T) {
	store := NewMemoryStore()
	hooks := NewActivityGenerator(store)
	timeline := NewTimeline(store)

	// Create a contact object
	contact := &BaseObject{
		ID:        "contact-123",
		Kind:      KindRecord,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "user-1",
		ACL:       []ACLEntry{{ActorID: "user-1", Role: "owner"}},
		Fields: map[string]interface{}{
			"schemaId":  "schema:crm_contact",
			"firstName": "Sarah",
			"lastName":  "Chen",
			"email":     "sarah@acme.com",
		},
	}

	// Store the object
	if err := store.Create(contact); err != nil {
		t.Fatalf("Failed to create contact: %v", err)
	}

	// Trigger onCreate hook
	if err := hooks.OnCreate(contact); err != nil {
		t.Fatalf("Failed to generate activity: %v", err)
	}

	// Retrieve timeline for the contact
	activities, err := timeline.GetTimelineForObject(contact.ID)
	if err != nil {
		t.Fatalf("Failed to get timeline: %v", err)
	}

	// Should have 1 activity
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity, got %d", len(activities))
	}

	// Verify the activity
	activity := activities[0]
	fields, err := activity.GetFields()
	if err != nil {
		t.Fatalf("Failed to get activity fields: %v", err)
	}

	if fields.Verb != VerbCreated {
		t.Errorf("Expected verb %s, got %s", VerbCreated, fields.Verb)
	}

	if fields.ActorID != "user-1" {
		t.Errorf("Expected actor user-1, got %s", fields.ActorID)
	}

	if fields.ObjectID != contact.ID {
		t.Errorf("Expected objectId %s, got %s", contact.ID, fields.ObjectID)
	}

	if fields.ObjectKind != KindRecord {
		t.Errorf("Expected objectKind %s, got %s", KindRecord, fields.ObjectKind)
	}
}

func TestIntegration_UpdateObjectGeneratesActivity(t *testing.T) {
	store := NewMemoryStore()
	hooks := NewActivityGenerator(store)
	timeline := NewTimeline(store)

	// Create initial object
	contact := &BaseObject{
		ID:        "contact-456",
		Kind:      KindRecord,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "user-1",
		ACL:       []ACLEntry{{ActorID: "user-1", Role: "owner"}},
		Fields: map[string]interface{}{
			"title": "VP Engineering",
			"email": "test@example.com",
		},
	}

	_ = store.Create(contact) //nolint:errcheck

	// Update the object
	time.Sleep(time.Millisecond) // Ensure different timestamp
	updatedContact := &BaseObject{
		ID:        contact.ID,
		Kind:      contact.Kind,
		CreatedAt: contact.CreatedAt,
		UpdatedAt: time.Now(),
		CreatedBy: "user-1",
		ACL:       contact.ACL,
		Fields: map[string]interface{}{
			"title": "CTO",
			"email": "test@example.com",
		},
	}

	// Trigger onUpdate hook
	if err := hooks.OnUpdate(contact, updatedContact); err != nil {
		t.Fatalf("Failed to generate update activity: %v", err)
	}

	// Retrieve timeline
	activities, err := timeline.GetTimelineForObject(contact.ID)
	if err != nil {
		t.Fatalf("Failed to get timeline: %v", err)
	}

	// Should have 1 activity (update)
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity, got %d", len(activities))
	}

	// Verify the activity
	activity := activities[0]
	fields, err := activity.GetFields()
	if err != nil {
		t.Fatalf("Failed to get activity fields: %v", err)
	}

	if fields.Verb != VerbUpdated {
		t.Errorf("Expected verb %s, got %s", VerbUpdated, fields.Verb)
	}

	// Check metadata contains changes
	if fields.Metadata == nil {
		t.Fatal("Expected metadata to contain changes")
	}

	changes, ok := fields.Metadata["changes"]
	if !ok {
		t.Fatal("Expected changes in metadata")
	}

	changesMap, ok := changes.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected changes to be a map, got %T", changes)
	}

	if _, exists := changesMap["title"]; !exists {
		t.Error("Expected title in changes")
	}
}

func TestIntegration_DeleteObjectGeneratesActivity(t *testing.T) {
	store := NewMemoryStore()
	hooks := NewActivityGenerator(store)
	timeline := NewTimeline(store)

	// Create an object
	task := &BaseObject{
		ID:        "task-789",
		Kind:      KindTask,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "user-1",
		ACL:       []ACLEntry{{ActorID: "user-1", Role: "owner"}},
		Fields: map[string]interface{}{
			"title":  "Follow up with Sarah",
			"status": "todo",
		},
	}

	_ = store.Create(task) //nolint:errcheck

	// Trigger onDelete hook
	if err := hooks.OnDelete(task); err != nil {
		t.Fatalf("Failed to generate delete activity: %v", err)
	}

	// Retrieve timeline
	activities, err := timeline.GetTimelineForObject(task.ID)
	if err != nil {
		t.Fatalf("Failed to get timeline: %v", err)
	}

	// Should have 1 activity (delete)
	if len(activities) != 1 {
		t.Fatalf("Expected 1 activity, got %d", len(activities))
	}

	// Verify the activity
	activity := activities[0]
	fields, err := activity.GetFields()
	if err != nil {
		t.Fatalf("Failed to get activity fields: %v", err)
	}

	if fields.Verb != VerbDeleted {
		t.Errorf("Expected verb %s, got %s", VerbDeleted, fields.Verb)
	}

	if fields.ObjectKind != KindTask {
		t.Errorf("Expected objectKind %s, got %s", KindTask, fields.ObjectKind)
	}
}

func TestIntegration_CompleteWorkflow(t *testing.T) {
	store := NewMemoryStore()
	hooks := NewActivityGenerator(store)
	timeline := NewTimeline(store)

	// Simulate a complete workflow
	contactID := "contact-complete-workflow"

	// 1. Create a contact
	contact := &BaseObject{
		ID:        contactID,
		Kind:      KindRecord,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "user-1",
		ACL:       []ACLEntry{{ActorID: "user-1", Role: "owner"}},
		Fields: map[string]interface{}{
			"firstName": "John",
			"lastName":  "Doe",
		},
	}
	_ = store.Create(contact)   //nolint:errcheck
	_ = hooks.OnCreate(contact) //nolint:errcheck

	time.Sleep(time.Millisecond)

	// 2. Update the contact (first time)
	updated1 := &BaseObject{
		ID:        contactID,
		Kind:      KindRecord,
		CreatedAt: contact.CreatedAt,
		UpdatedAt: time.Now(),
		CreatedBy: "user-1",
		ACL:       contact.ACL,
		Fields: map[string]interface{}{
			"firstName": "John",
			"lastName":  "Doe",
			"title":     "Engineer",
		},
	}
	_ = hooks.OnUpdate(contact, updated1) //nolint:errcheck

	time.Sleep(time.Millisecond)

	// 3. Update the contact (second time)
	updated2 := &BaseObject{
		ID:        contactID,
		Kind:      KindRecord,
		CreatedAt: contact.CreatedAt,
		UpdatedAt: time.Now(),
		CreatedBy: "user-1",
		ACL:       contact.ACL,
		Fields: map[string]interface{}{
			"firstName": "John",
			"lastName":  "Doe",
			"title":     "Senior Engineer",
		},
	}
	_ = hooks.OnUpdate(updated1, updated2) //nolint:errcheck

	// 4. Get the complete timeline
	activities, err := timeline.GetTimelineForObject(contactID)
	if err != nil {
		t.Fatalf("Failed to get timeline: %v", err)
	}

	// Should have 3 activities: create + 2 updates
	if len(activities) != 3 {
		t.Fatalf("Expected 3 activities, got %d", len(activities))
	}

	// Verify chronological order (newest first)
	verbs := []ActivityVerb{VerbUpdated, VerbUpdated, VerbCreated}
	for i, expected := range verbs {
		fields, _ := activities[i].GetFields()
		if fields.Verb != expected {
			t.Errorf("Activity %d: expected verb %s, got %s", i, expected, fields.Verb)
		}
	}
}

func TestIntegration_NoActivityForNoChange(t *testing.T) {
	store := NewMemoryStore()
	hooks := NewActivityGenerator(store)
	timeline := NewTimeline(store)

	// Create an object
	task := &BaseObject{
		ID:        "task-nochange",
		Kind:      KindTask,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "user-1",
		ACL:       []ACLEntry{{ActorID: "user-1", Role: "owner"}},
		Fields: map[string]interface{}{
			"title": "Same title",
		},
	}

	// "Update" with no actual changes
	_ = hooks.OnUpdate(task, task) //nolint:errcheck

	// Should have no activities
	activities, err := timeline.GetTimelineForObject(task.ID)
	if err != nil {
		t.Fatalf("Failed to get timeline: %v", err)
	}

	if len(activities) != 0 {
		t.Errorf("Expected 0 activities for no changes, got %d", len(activities))
	}
}
