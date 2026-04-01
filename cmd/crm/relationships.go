// ABOUTME: CLI commands for managing CRM relationships between entities.
// ABOUTME: Provides link and unlink subcommands for creating and deleting connections.

package main

import (
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link <source-id> <target-id>",
	Short: "Create a relationship between two entities",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceID, err := uuid.Parse(args[0])
		if err != nil {
			return err
		}
		targetID, err := uuid.Parse(args[1])
		if err != nil {
			return err
		}

		relType, _ := cmd.Flags().GetString("type")
		context, _ := cmd.Flags().GetString("context")

		rel := models.NewRelationship(sourceID, targetID, relType, context)
		if err := store.CreateRelationship(rel); err != nil {
			return err
		}

		cyan := color.New(color.FgCyan)
		out("Created relationship %s\n", cyan.Sprint(rel.ID))
		return nil
	},
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink <relationship-id>",
	Short: "Delete a relationship by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := uuid.Parse(args[0])
		if err != nil {
			return err
		}

		if err := store.DeleteRelationship(id); err != nil {
			return err
		}

		out("Deleted relationship %s\n", color.New(color.FgCyan).Sprint(id))
		return nil
	},
}

func init() {
	linkCmd.Flags().String("type", "", "relationship type (required)")
	_ = linkCmd.MarkFlagRequired("type")
	linkCmd.Flags().String("context", "", "optional context for the relationship")

	rootCmd.AddCommand(linkCmd)
	rootCmd.AddCommand(unlinkCmd)
}
