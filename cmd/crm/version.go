// ABOUTME: Version command to display build information.
// ABOUTME: Shows version, commit hash, and build date.

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		output := cmd.OutOrStdout()
		if _, err := fmt.Fprintf(output, "crm version %s\n", displayVersion()); err != nil {
			return fmt.Errorf("write version: %w", err)
		}
		if _, err := fmt.Fprintf(output, "  commit: %s\n", commit); err != nil {
			return fmt.Errorf("write commit: %w", err)
		}
		if _, err := fmt.Fprintf(output, "  built:  %s\n", date); err != nil {
			return fmt.Errorf("write build date: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
