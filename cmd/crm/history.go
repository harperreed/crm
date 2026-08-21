// ABOUTME: CLI commands for listing and inspecting immutable CRM history events.
// ABOUTME: Formats timeline summaries and complete before-and-after snapshots.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/harperreed/crm/internal/history"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history <entity-id-or-prefix>",
	Short: "List an entity's history",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		normalizedLimit, err := history.NormalizeLimit(limit)
		if err != nil {
			return err
		}
		summaries, err := store.ListHistory(args[0], normalizedLimit)
		if err != nil {
			return err
		}
		if len(summaries) == 0 {
			outln("No history found.")
			return nil
		}
		for _, summary := range summaries {
			outln(formatHistorySummary(summary))
		}
		return nil
	},
}

var historyShowCmd = &cobra.Command{
	Use:   "show <event-id-or-prefix>",
	Short: "Show a history event",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		event, err := store.GetHistoryEvent(args[0])
		if err != nil {
			return err
		}
		formatted, err := formatHistoryEvent(event)
		if err != nil {
			return err
		}
		out("%s", formatted)
		return nil
	},
}

func formatHistorySummary(summary *history.Summary) string {
	fields := slices.Clone(summary.ChangedFields)
	slices.Sort(fields)
	formatted := fmt.Sprintf(
		"%s  %s %s [%s] %s",
		summary.OccurredAt.UTC().Format(time.RFC3339),
		strings.ToUpper(string(summary.Action)),
		summary.EntityType,
		summary.Source,
		summary.ID.String()[:8],
	)
	if summary.Action == history.ActionUpdate && len(fields) > 0 {
		formatted += "  " + strings.Join(fields, ", ")
	}
	return formatted
}

func formatHistoryEvent(event *history.Event) (string, error) {
	before, err := formatHistorySnapshot(event.Before)
	if err != nil {
		return "", fmt.Errorf("format before snapshot: %w", err)
	}
	after, err := formatHistorySnapshot(event.After)
	if err != nil {
		return "", fmt.Errorf("format after snapshot: %w", err)
	}

	related := make([]string, len(event.RelatedEntityIDs))
	for index, id := range event.RelatedEntityIDs {
		related[index] = id.String()
	}
	return fmt.Sprintf(
		"ID: %s\nSchema: %d\nOccurred: %s\nEntity: %s %s\nAction: %s\nSource: %s\nRelated: %s\nBefore:\n%s\nAfter:\n%s\n",
		event.ID,
		event.SchemaVersion,
		event.OccurredAt.UTC().Format(time.RFC3339),
		event.EntityType,
		event.EntityID,
		strings.ToUpper(string(event.Action)),
		event.Source,
		strings.Join(related, ", "),
		before,
		after,
	), nil
}

func formatHistorySnapshot(snapshot json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(snapshot)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "null", nil
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, trimmed, "", "  "); err != nil {
		return "", err
	}
	return formatted.String(), nil
}

func init() {
	historyCmd.Flags().IntP("limit", "n", history.DefaultLimit, "max events to show")
	historyCmd.AddCommand(historyShowCmd)
	rootCmd.AddCommand(historyCmd)
}
