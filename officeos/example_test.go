// ABOUTME: Example demonstrating the activity timeline system
// ABOUTME: Shows how to use activities, timeline queries, and views
package officeos

import (
	"fmt"
	"time"
)

// Example_activityTimeline demonstrates the complete activity timeline workflow.
func Example_activityTimeline() {
	// Create store and components
	store := NewMemoryStore()
	hooks := NewActivityGenerator(store)
	timeline := NewTimeline(store)
	_ = NewTimelineView(timeline, 80, 40) // Available for rendering

	// Simulate creating a contact
	contact := &BaseObject{
		ID:        "contact-sarah",
		Kind:      KindRecord,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: "you",
		ACL:       []ACLEntry{{ActorID: "you", Role: "owner"}},
		Fields: map[string]interface{}{
			"schemaId":  "schema:crm_contact",
			"firstName": "Sarah",
			"lastName":  "Chen",
			"email":     "sarah@acme.com",
			"title":     "VP Engineering",
		},
	}

	// Store and generate creation activity
	_ = store.Create(contact)   //nolint:errcheck
	_ = hooks.OnCreate(contact) //nolint:errcheck

	time.Sleep(time.Millisecond)

	// Simulate updating the contact
	updatedContact := &BaseObject{
		ID:        contact.ID,
		Kind:      contact.Kind,
		CreatedAt: contact.CreatedAt,
		UpdatedAt: time.Now(),
		CreatedBy: "you",
		ACL:       contact.ACL,
		Fields: map[string]interface{}{
			"schemaId":  "schema:crm_contact",
			"firstName": "Sarah",
			"lastName":  "Chen",
			"email":     "sarah@acme.com",
			"title":     "CTO", // Changed!
		},
	}

	_ = hooks.OnUpdate(contact, updatedContact) //nolint:errcheck

	// Query the timeline
	activities, _ := timeline.GetTimelineForObject(contact.ID)

	fmt.Printf("Timeline for Sarah Chen:\n")
	fmt.Printf("Total activities: %d\n", len(activities))

	for _, activity := range activities {
		fields, _ := activity.GetFields()
		fmt.Printf("- %s %s\n", fields.ActorID, fields.Verb)
	}

	// Output:
	// Timeline for Sarah Chen:
	// Total activities: 2
	// - you updated
	// - you created
}

// Example_timelineFiltering demonstrates filtering timeline activities.
func Example_timelineFiltering() {
	store := NewMemoryStore()
	timeline := NewTimeline(store)

	// Create multiple activities
	activities := []*ActivityObject{
		NewActivityObject("user-1", VerbCreated, "obj-1", KindRecord, nil),
		NewActivityObject("user-2", VerbUpdated, "obj-1", KindRecord, nil),
		NewActivityObject("user-1", VerbViewed, "obj-2", KindTask, nil),
	}

	for _, act := range activities {
		_ = store.Create(&act.BaseObject) //nolint:errcheck
	}

	// Filter by actor
	user1Activities, _ := timeline.GetActivitiesByActor("user-1", 0)
	fmt.Printf("Activities by user-1: %d\n", len(user1Activities))

	// Filter by verb
	updatedActivities, _ := timeline.GetActivitiesByVerb(VerbUpdated, 0)
	fmt.Printf("Update activities: %d\n", len(updatedActivities))

	// Output:
	// Activities by user-1: 2
	// Update activities: 1
}

// Example_timelineView demonstrates rendering a timeline.
func Example_timelineView() {
	store := NewMemoryStore()
	timeline := NewTimeline(store)
	view := NewTimelineView(timeline, 80, 40)

	// Create an activity
	act := NewActivityObject("you", VerbCreated, "contact-123", KindRecord, map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
	})
	_ = store.Create(&act.BaseObject) //nolint:errcheck

	// Render the timeline
	output := view.Render("contact-123")
	fmt.Printf("Timeline rendered: %d characters\n", len(output))

	// Output shows the timeline was rendered
	// Output:
	// Timeline rendered: 169 characters
}
