// ABOUTME: Tests for timeline query functionality
// ABOUTME: Covers filtering, ordering, and timeline retrieval
package officeos

import (
	"testing"
	"time"
)

func TestTimeline_GetTimelineForObject(t *testing.T) {
	store := NewMemoryStore()
	timeline := NewTimeline(store)

	objectID := "contact-123"

	// Create some activities
	act1 := NewActivityObject("user-1", VerbCreated, objectID, KindRecord, nil)
	act2 := NewActivityObject("user-1", VerbUpdated, objectID, KindRecord, map[string]interface{}{
		"changes": map[string]interface{}{"title": "changed"},
	})
	act3 := NewActivityObject("user-2", VerbViewed, "other-object", KindRecord, nil)

	if err := store.Create(&act1.BaseObject); err != nil {
		t.Fatalf("Failed to create activity: %v", err)
	}

	// Wait a tiny bit to ensure different timestamps
	time.Sleep(time.Millisecond)

	if err := store.Create(&act2.BaseObject); err != nil {
		t.Fatalf("Failed to create activity: %v", err)
	}

	if err := store.Create(&act3.BaseObject); err != nil {
		t.Fatalf("Failed to create activity: %v", err)
	}

	// Get timeline for objectID
	activities, err := timeline.GetTimelineForObject(objectID)
	if err != nil {
		t.Fatalf("Failed to get timeline: %v", err)
	}

	// Should get 2 activities (act1 and act2), not act3
	if len(activities) != 2 {
		t.Fatalf("Expected 2 activities, got %d", len(activities))
	}

	// Verify they are sorted newest first
	if !activities[0].CreatedAt.After(activities[1].CreatedAt) {
		t.Error("Activities should be sorted newest first")
	}

	// Verify the content
	fields0, _ := activities[0].GetFields()
	fields1, _ := activities[1].GetFields()

	if fields0.Verb != VerbUpdated {
		t.Errorf("Expected first activity to be updated, got %s", fields0.Verb)
	}

	if fields1.Verb != VerbCreated {
		t.Errorf("Expected second activity to be created, got %s", fields1.Verb)
	}
}

func TestTimeline_GetTimelineWithFilters(t *testing.T) {
	store := NewMemoryStore()
	timeline := NewTimeline(store)

	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	// Create activities with different attributes
	act1 := NewActivityObject("user-1", VerbCreated, "obj-1", KindRecord, nil)
	act1.CreatedAt = past

	act2 := NewActivityObject("user-2", VerbUpdated, "obj-1", KindRecord, nil)
	act2.CreatedAt = now

	act3 := NewActivityObject("user-1", VerbDeleted, "obj-2", KindTask, nil)
	act3.CreatedAt = future

	_ = store.Create(&act1.BaseObject) //nolint:errcheck
	_ = store.Create(&act2.BaseObject) //nolint:errcheck
	_ = store.Create(&act3.BaseObject) //nolint:errcheck

	// Test actor filter
	t.Run("FilterByActor", func(t *testing.T) {
		activities, err := timeline.GetActivitiesByActor("user-1", 0)
		if err != nil {
			t.Fatalf("Failed to get activities: %v", err)
		}

		if len(activities) != 2 {
			t.Errorf("Expected 2 activities for user-1, got %d", len(activities))
		}
	})

	// Test verb filter
	t.Run("FilterByVerb", func(t *testing.T) {
		activities, err := timeline.GetActivitiesByVerb(VerbUpdated, 0)
		if err != nil {
			t.Fatalf("Failed to get activities: %v", err)
		}

		if len(activities) != 1 {
			t.Errorf("Expected 1 activity with verb updated, got %d", len(activities))
		}

		fields, _ := activities[0].GetFields()
		if fields.ActorID != "user-2" {
			t.Errorf("Expected actor user-2, got %s", fields.ActorID)
		}
	})

	// Test object kind filter
	t.Run("FilterByObjectKind", func(t *testing.T) {
		activities, err := timeline.GetTimeline(TimelineFilter{ObjectKind: KindTask})
		if err != nil {
			t.Fatalf("Failed to get activities: %v", err)
		}

		if len(activities) != 1 {
			t.Errorf("Expected 1 activity for task objects, got %d", len(activities))
		}
	})

	// Test date range filter
	t.Run("FilterByDateRange", func(t *testing.T) {
		start := now.Add(-1 * time.Hour)
		end := now.Add(1 * time.Hour)

		activities, err := timeline.GetTimeline(TimelineFilter{
			StartDate: &start,
			EndDate:   &end,
		})
		if err != nil {
			t.Fatalf("Failed to get activities: %v", err)
		}

		if len(activities) != 1 {
			t.Errorf("Expected 1 activity in date range, got %d", len(activities))
		}
	})

	// Test limit
	t.Run("Limit", func(t *testing.T) {
		activities, err := timeline.GetRecentActivities(2)
		if err != nil {
			t.Fatalf("Failed to get activities: %v", err)
		}

		if len(activities) != 2 {
			t.Errorf("Expected 2 activities (limited), got %d", len(activities))
		}
	})
}

func TestTimeline_EmptyResults(t *testing.T) {
	store := NewMemoryStore()
	timeline := NewTimeline(store)

	activities, err := timeline.GetTimelineForObject("nonexistent")
	if err != nil {
		t.Fatalf("Failed to get timeline: %v", err)
	}

	if len(activities) != 0 {
		t.Errorf("Expected 0 activities, got %d", len(activities))
	}
}

func TestTimeline_ChronologicalOrdering(t *testing.T) {
	store := NewMemoryStore()
	timeline := NewTimeline(store)

	// Create activities with known order
	for i := 0; i < 5; i++ {
		act := NewActivityObject("user-1", VerbCreated, "obj-1", KindRecord, nil)
		act.CreatedAt = time.Now().Add(time.Duration(i) * time.Second)
		_ = store.Create(&act.BaseObject) //nolint:errcheck
		time.Sleep(time.Millisecond)      // Ensure different timestamps
	}

	activities, err := timeline.GetTimelineForObject("obj-1")
	if err != nil {
		t.Fatalf("Failed to get timeline: %v", err)
	}

	if len(activities) != 5 {
		t.Fatalf("Expected 5 activities, got %d", len(activities))
	}

	// Verify they are in descending chronological order
	for i := 0; i < len(activities)-1; i++ {
		if activities[i].CreatedAt.Before(activities[i+1].CreatedAt) {
			t.Errorf("Activities not properly sorted: %v should be after %v",
				activities[i].CreatedAt, activities[i+1].CreatedAt)
		}
	}
}
