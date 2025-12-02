// ABOUTME: Tests for timeline view rendering
// ABOUTME: Validates timeline display formatting and styling
package officeos

import (
	"strings"
	"testing"
	"time"
)

func TestTimelineView_RenderEmpty(t *testing.T) {
	store := NewMemoryStore()
	timeline := NewTimeline(store)
	view := NewTimelineView(timeline, 80, 24)

	output := view.Render("nonexistent-object")

	if !strings.Contains(output, "No activities") {
		t.Errorf("Expected empty message, got: %s", output)
	}
}

func TestTimelineView_RenderActivities(t *testing.T) {
	store := NewMemoryStore()
	timeline := NewTimeline(store)
	view := NewTimelineView(timeline, 80, 24)

	objectID := "test-object"

	// Create some activities
	act1 := NewActivityObject("user-1", VerbCreated, objectID, KindRecord, nil)
	act2 := NewActivityObject("user-1", VerbUpdated, objectID, KindRecord, map[string]interface{}{
		"changes": map[string]interface{}{
			"title": map[string]interface{}{
				"before": "Old Title",
				"after":  "New Title",
			},
		},
	})

	_ = store.Create(&act1.BaseObject) //nolint:errcheck
	_ = store.Create(&act2.BaseObject) //nolint:errcheck

	output := view.Render(objectID)

	// Verify output contains expected elements
	if !strings.Contains(output, "ACTIVITY TIMELINE") {
		t.Error("Expected timeline header")
	}

	if !strings.Contains(output, "user-1") {
		t.Error("Expected actor name")
	}

	if !strings.Contains(output, "created") {
		t.Error("Expected created verb")
	}

	if !strings.Contains(output, "updated") {
		t.Error("Expected updated verb")
	}

	if !strings.Contains(output, "record") {
		t.Error("Expected object kind")
	}
}

func TestTimelineView_RenderMetadata(t *testing.T) {
	store := NewMemoryStore()
	timeline := NewTimeline(store)
	view := NewTimelineView(timeline, 80, 24)

	objectID := "test-object"

	// Create activity with metadata
	act := NewActivityObject("user-1", VerbUpdated, objectID, KindRecord, map[string]interface{}{
		"changes": map[string]interface{}{
			"title": map[string]interface{}{
				"before": "VP Engineering",
				"after":  "CTO",
			},
		},
	})

	_ = store.Create(&act.BaseObject) //nolint:errcheck

	output := view.Render(objectID)

	// Verify metadata is rendered
	if !strings.Contains(output, "Changes") {
		t.Error("Expected 'Changes' label")
	}

	if !strings.Contains(output, "title") {
		t.Error("Expected field name 'title'")
	}

	if !strings.Contains(output, "VP Engineering") {
		t.Error("Expected before value")
	}

	if !strings.Contains(output, "CTO") {
		t.Error("Expected after value")
	}
}

func TestTimelineView_FormatTimestamp(t *testing.T) {
	store := NewMemoryStore()
	timeline := NewTimeline(store)
	view := NewTimelineView(timeline, 80, 24)

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "just now",
			time:     time.Now().Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "minutes ago",
			time:     time.Now().Add(-5 * time.Minute),
			expected: "5 min ago",
		},
		{
			name:     "hours ago",
			time:     time.Now().Add(-3 * time.Hour),
			expected: "3 hours ago",
		},
		{
			name:     "days ago",
			time:     time.Now().Add(-2 * 24 * time.Hour),
			expected: "2 days ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := view.formatTimestamp(tt.time)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestTimelineView_RenderFiltered(t *testing.T) {
	store := NewMemoryStore()
	timeline := NewTimeline(store)
	view := NewTimelineView(timeline, 80, 24)

	// Create activities for different objects
	act1 := NewActivityObject("user-1", VerbCreated, "obj-1", KindRecord, nil)
	act2 := NewActivityObject("user-2", VerbUpdated, "obj-2", KindTask, nil)
	act3 := NewActivityObject("user-1", VerbDeleted, "obj-3", KindRecord, nil)

	_ = store.Create(&act1.BaseObject) //nolint:errcheck
	_ = store.Create(&act2.BaseObject) //nolint:errcheck
	_ = store.Create(&act3.BaseObject) //nolint:errcheck

	// Filter by actor
	output := view.RenderFiltered(TimelineFilter{ActorID: "user-1"})

	// Should only show user-1's activities
	countUser1 := strings.Count(output, "user-1")
	countUser2 := strings.Count(output, "user-2")

	if countUser1 == 0 {
		t.Error("Expected to find user-1 activities")
	}

	if countUser2 != 0 {
		t.Error("Should not find user-2 activities")
	}
}

func TestTimelineView_SetSize(t *testing.T) {
	store := NewMemoryStore()
	timeline := NewTimeline(store)
	view := NewTimelineView(timeline, 80, 24)

	view.SetSize(120, 40)

	if view.width != 120 {
		t.Errorf("Expected width 120, got %d", view.width)
	}

	if view.height != 40 {
		t.Errorf("Expected height 40, got %d", view.height)
	}
}

func TestTimelineView_VerbColors(t *testing.T) {
	store := NewMemoryStore()
	timeline := NewTimeline(store)
	view := NewTimelineView(timeline, 80, 24)

	// Test that different verbs produce styled outputs by checking they render
	verbs := []ActivityVerb{VerbCreated, VerbUpdated, VerbDeleted, VerbViewed}

	for _, verb := range verbs {
		style := view.getVerbStyle(verb)
		// Just verify the style can be used to render
		output := style.Render(string(verb))
		if output == "" {
			t.Errorf("Expected non-empty output for verb %s", verb)
		}
	}
}
