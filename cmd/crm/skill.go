// ABOUTME: CLI command to install the CRM skill definition for Claude Code.
// ABOUTME: Embeds SKILL.md and writes it to ~/.claude/skills/crm/.
package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed skill/SKILL.md
var skillMD []byte

var skillCmd = &cobra.Command{
	Use:   "install-skill",
	Short: "Install CRM skill for Claude Code",
	Long:  "Copies the CRM SKILL.md to ~/.claude/skills/crm/ for Claude Code integration.",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("get home dir: %w", err)
		}

		dir := filepath.Join(home, ".claude", "skills", "crm")
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create skill dir: %w", err)
		}

		dest := filepath.Join(dir, "SKILL.md")
		if err := os.WriteFile(dest, skillMD, 0o600); err != nil {
			return fmt.Errorf("write skill file: %w", err)
		}

		fmt.Printf("Installed CRM skill to %s\n", dest)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(skillCmd)
}
