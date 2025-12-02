// ABOUTME: Timeline query implementation for activity retrieval
// ABOUTME: Provides filtering and chronological ordering of activities
package officeos

import (
	"sort"
	"time"
)

// TimelineFilter represents filter options for timeline queries.
type TimelineFilter struct {
	ObjectID   string       // Filter by related object ID
	ActorID    string       // Filter by actor ID
	Verb       ActivityVerb // Filter by verb
	StartDate  *time.Time   // Filter by date range start
	EndDate    *time.Time   // Filter by date range end
	ObjectKind ObjectKind   // Filter by object kind
	Limit      int          // Limit number of results (0 = no limit)
}

// Timeline provides methods to query and retrieve activities.
type Timeline struct {
	store ObjectStore
}

// NewTimeline creates a new timeline instance.
func NewTimeline(store ObjectStore) *Timeline {
	return &Timeline{store: store}
}

// GetTimeline retrieves activities for an object, optionally filtered.
func (t *Timeline) GetTimeline(filter TimelineFilter) ([]*ActivityObject, error) {
	// Query for all activities
	filters := map[string]interface{}{}

	activities, err := t.store.Query(KindActivity, filters)
	if err != nil {
		return nil, err
	}

	// Convert to ActivityObjects and apply filters
	var results []*ActivityObject
	for _, obj := range activities {
		activityObj := &ActivityObject{BaseObject: *obj}
		fields, err := activityObj.GetFields()
		if err != nil {
			continue // Skip invalid activities
		}

		// Apply filters
		if filter.ObjectID != "" && fields.ObjectID != filter.ObjectID {
			continue
		}

		if filter.ActorID != "" && fields.ActorID != filter.ActorID {
			continue
		}

		if filter.Verb != "" && fields.Verb != filter.Verb {
			continue
		}

		if filter.ObjectKind != "" && fields.ObjectKind != filter.ObjectKind {
			continue
		}

		if filter.StartDate != nil && obj.CreatedAt.Before(*filter.StartDate) {
			continue
		}

		if filter.EndDate != nil && obj.CreatedAt.After(*filter.EndDate) {
			continue
		}

		results = append(results, activityObj)
	}

	// Sort by creation time (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	// Apply limit if specified
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	return results, nil
}

// GetTimelineForObject is a convenience method to get all activities for a specific object.
func (t *Timeline) GetTimelineForObject(objectID string) ([]*ActivityObject, error) {
	return t.GetTimeline(TimelineFilter{ObjectID: objectID})
}

// GetRecentActivities retrieves the most recent activities across all objects.
func (t *Timeline) GetRecentActivities(limit int) ([]*ActivityObject, error) {
	return t.GetTimeline(TimelineFilter{Limit: limit})
}

// GetActivitiesByActor retrieves all activities performed by a specific actor.
func (t *Timeline) GetActivitiesByActor(actorID string, limit int) ([]*ActivityObject, error) {
	return t.GetTimeline(TimelineFilter{ActorID: actorID, Limit: limit})
}

// GetActivitiesByVerb retrieves all activities with a specific verb.
func (t *Timeline) GetActivitiesByVerb(verb ActivityVerb, limit int) ([]*ActivityObject, error) {
	return t.GetTimeline(TimelineFilter{Verb: verb, Limit: limit})
}
