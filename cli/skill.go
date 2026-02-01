// ABOUTME: Install Claude Code skill command for pagen
// ABOUTME: Embeds and installs the skill definition to ~/.claude/skills/

package cli

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed skill/SKILL.md
var skillFS embed.FS

// InstallSkillCommand installs the pagen skill for Claude Code.
func InstallSkillCommand() error {
	// Read embedded skill file
	content, err := skillFS.ReadFile("skill/SKILL.md")
	if err != nil {
		return fmt.Errorf("failed to read embedded skill: %w", err)
	}

	// Determine destination
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	skillDir := filepath.Join(home, ".claude", "skills", "pagen")
	skillPath := filepath.Join(skillDir, "SKILL.md")

	// Create directory
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Write skill file
	if err := os.WriteFile(skillPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	fmt.Printf("Installed pagen skill to %s\n", skillPath)
	fmt.Println("Claude Code will now recognize /pagen commands.")
	return nil
}
