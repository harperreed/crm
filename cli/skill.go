// ABOUTME: Install Claude Code skill command for pagen
// ABOUTME: Embeds and installs the skill definition to ~/.claude/skills/

package cli

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed skill/SKILL.md
var skillFS embed.FS

// InstallSkillCommand installs the pagen skill for Claude Code.
// If skipConfirm is true, it installs without asking for confirmation.
func InstallSkillCommand(skipConfirm bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	return installSkillToDir(home, skipConfirm)
}

// installSkillToDir is the internal implementation that allows specifying a custom home directory.
// This enables testing without modifying the real home directory.
func installSkillToDir(homeDir string, skipConfirm bool) error {
	skillDir := filepath.Join(homeDir, ".claude", "skills", "pagen")
	skillPath := filepath.Join(skillDir, "SKILL.md")

	// Show explanation
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│             Pagen Skill for Claude Code                     │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("This will install the pagen skill, enabling Claude Code to:")
	fmt.Println()
	fmt.Println("  • Manage contacts and relationships")
	fmt.Println("  • Track companies and deals")
	fmt.Println("  • Personal CRM functionality")
	fmt.Println("  • Use the /pagen slash command")
	fmt.Println()
	fmt.Println("Destination:")
	fmt.Printf("  %s\n", skillPath)
	fmt.Println()

	// Check if already installed
	if _, err := os.Stat(skillPath); err == nil {
		fmt.Println("Note: A skill file already exists and will be overwritten.")
		fmt.Println()
	}

	// Ask for confirmation unless skipConfirm is true
	if !skipConfirm {
		fmt.Print("Install the pagen skill? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Installation cancelled.")
			return nil
		}
		fmt.Println()
	}

	// Read embedded skill file
	content, err := skillFS.ReadFile("skill/SKILL.md")
	if err != nil {
		return fmt.Errorf("failed to read embedded skill: %w", err)
	}

	// Create directory
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Write skill file
	if err := os.WriteFile(skillPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	fmt.Println("✓ Installed pagen skill successfully!")
	fmt.Println()
	fmt.Println("Claude Code will now recognize /pagen commands.")
	fmt.Println("Try asking Claude: \"Add a contact for John Smith\" or \"Show my recent contacts\"")
	return nil
}
