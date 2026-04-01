// ABOUTME: Root Cobra command with config-driven storage initialization.
// ABOUTME: Sets up CLI structure, opens storage on startup, and closes it on exit.

package main

import (
	"fmt"

	"github.com/harperreed/crm/internal/config"
	"github.com/harperreed/crm/internal/storage"
	"github.com/spf13/cobra"
)

var store storage.Storage

var rootCmd = &cobra.Command{
	Use:   "crm",
	Short: "A simple CRM for contacts, companies, and relationships",
	Long:  "CRM is a lightweight contact relationship manager accessible via CLI and MCP.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip storage init for commands that don't need it.
		if cmd.Name() == "version" {
			return nil
		}
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		s, err := cfg.OpenStorage()
		if err != nil {
			return fmt.Errorf("open storage: %w", err)
		}
		store = s
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if store != nil {
			return store.Close()
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
