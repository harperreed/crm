// ABOUTME: Root Cobra command and global flags.
// ABOUTME: Sets up CLI structure and Execute entry point.

package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "crm",
	Short: "A simple CRM for contacts, companies, and relationships",
	Long:  "CRM is a lightweight contact relationship manager accessible via CLI and MCP.",
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
