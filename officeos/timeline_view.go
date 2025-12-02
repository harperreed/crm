// ABOUTME: TUI component for rendering activity timelines
// ABOUTME: Provides formatted timeline display with timestamps and filters
package officeos

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Timeline view styles.
	timelineHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("170")).
				MarginBottom(1)

	activityItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				MarginLeft(2)

	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(20)

	actorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	verbCreatedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46")) // Green

	verbUpdatedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("33")) // Blue

	verbDeletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")) // Red

	verbViewedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")) // Gray

	metadataStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			MarginLeft(4)
)

// TimelineView handles rendering of activity timelines.
type TimelineView struct {
	timeline *Timeline
	width    int
	height   int
}

// NewTimelineView creates a new timeline view.
func NewTimelineView(timeline *Timeline, width, height int) *TimelineView {
	return &TimelineView{
		timeline: timeline,
		width:    width,
		height:   height,
	}
}

// Render renders the timeline for a specific object.
func (tv *TimelineView) Render(objectID string) string {
	activities, err := tv.timeline.GetTimelineForObject(objectID)
	if err != nil {
		return fmt.Sprintf("Error loading timeline: %v", err)
	}

	if len(activities) == 0 {
		return tv.renderEmpty()
	}

	return tv.renderActivities(activities)
}

// RenderFiltered renders activities with a custom filter.
func (tv *TimelineView) RenderFiltered(filter TimelineFilter) string {
	activities, err := tv.timeline.GetTimeline(filter)
	if err != nil {
		return fmt.Sprintf("Error loading timeline: %v", err)
	}

	if len(activities) == 0 {
		return tv.renderEmpty()
	}

	return tv.renderActivities(activities)
}

// renderActivities renders a list of activities.
func (tv *TimelineView) renderActivities(activities []*ActivityObject) string {
	var s strings.Builder

	s.WriteString(timelineHeaderStyle.Render("ACTIVITY TIMELINE"))
	s.WriteString("\n\n")

	for _, activity := range activities {
		s.WriteString(tv.renderActivity(activity))
		s.WriteString("\n")
	}

	return s.String()
}

// renderActivity renders a single activity.
func (tv *TimelineView) renderActivity(activity *ActivityObject) string {
	fields, err := activity.GetFields()
	if err != nil {
		return fmt.Sprintf("Error: %v\n", err)
	}

	var s strings.Builder

	// Timestamp
	timestamp := tv.formatTimestamp(activity.CreatedAt)
	s.WriteString(timestampStyle.Render(timestamp))
	s.WriteString("  ")

	// Actor
	s.WriteString(actorStyle.Render(fields.ActorID))
	s.WriteString(" ")

	// Verb with color coding
	verbStyle := tv.getVerbStyle(fields.Verb)
	s.WriteString(verbStyle.Render(string(fields.Verb)))
	s.WriteString(" ")

	// Object type
	s.WriteString(string(fields.ObjectKind))

	// Metadata (if any)
	if len(fields.Metadata) > 0 {
		s.WriteString("\n")
		s.WriteString(tv.renderMetadata(fields.Metadata))
	}

	return activityItemStyle.Render(s.String())
}

// renderMetadata renders activity metadata.
func (tv *TimelineView) renderMetadata(metadata map[string]interface{}) string {
	var s strings.Builder

	// Render changes if present
	if changes, ok := metadata["changes"].(map[string]interface{}); ok {
		s.WriteString(metadataStyle.Render("Changes:"))
		s.WriteString("\n")

		for field, change := range changes {
			changeMap, ok := change.(map[string]interface{})
			if !ok {
				continue
			}

			before := changeMap["before"]
			after := changeMap["after"]

			changeStr := fmt.Sprintf("  %s: %v → %v", field, before, after)
			s.WriteString(metadataStyle.Render(changeStr))
			s.WriteString("\n")
		}
	}

	// Render other metadata
	for key, value := range metadata {
		if key == "changes" || key == "timestamp" {
			continue // Already handled
		}

		metaStr := fmt.Sprintf("%s: %v", key, value)
		s.WriteString(metadataStyle.Render(metaStr))
		s.WriteString("\n")
	}

	return s.String()
}

// renderEmpty renders an empty timeline message.
func (tv *TimelineView) renderEmpty() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("No activities to display")
}

// formatTimestamp formats a timestamp for display.
func (tv *TimelineView) formatTimestamp(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return fmt.Sprintf("%d min ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

// getVerbStyle returns the appropriate style for a verb.
func (tv *TimelineView) getVerbStyle(verb ActivityVerb) lipgloss.Style {
	switch verb {
	case VerbCreated:
		return verbCreatedStyle
	case VerbUpdated:
		return verbUpdatedStyle
	case VerbDeleted:
		return verbDeletedStyle
	case VerbViewed:
		return verbViewedStyle
	default:
		return lipgloss.NewStyle()
	}
}

// SetSize updates the view dimensions.
func (tv *TimelineView) SetSize(width, height int) {
	tv.width = width
	tv.height = height
}
