// ABOUTME: Tests for the install-skill command functionality
// ABOUTME: Verifies skill installation, directory creation, and file content

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallSkillCommand_SuccessfulInstall verifies that the skill installs correctly
// when skipConfirm is true.
func TestInstallSkillCommand_SuccessfulInstall(t *testing.T) {
	tempDir := t.TempDir()

	err := installSkillToDir(tempDir, true)
	if err != nil {
		t.Fatalf("installSkillToDir failed: %v", err)
	}

	// Verify file was created
	skillPath := filepath.Join(tempDir, ".claude", "skills", "pagen", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Error("SKILL.md was not created")
	}
}

// TestInstallSkillCommand_CreatesDirectoryStructure verifies that the necessary
// directory structure is created during installation.
func TestInstallSkillCommand_CreatesDirectoryStructure(t *testing.T) {
	tempDir := t.TempDir()

	err := installSkillToDir(tempDir, true)
	if err != nil {
		t.Fatalf("installSkillToDir failed: %v", err)
	}

	// Check each directory in the path exists
	claudeDir := filepath.Join(tempDir, ".claude")
	skillsDir := filepath.Join(claudeDir, "skills")
	pagenDir := filepath.Join(skillsDir, "pagen")

	for _, dir := range []string{claudeDir, skillsDir, pagenDir} {
		info, err := os.Stat(dir)
		if os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
			continue
		}
		if err != nil {
			t.Errorf("Error checking directory %s: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}
}

// TestInstallSkillCommand_FileContent verifies that the installed SKILL.md
// contains the expected content from the embedded file.
func TestInstallSkillCommand_FileContent(t *testing.T) {
	tempDir := t.TempDir()

	err := installSkillToDir(tempDir, true)
	if err != nil {
		t.Fatalf("installSkillToDir failed: %v", err)
	}

	skillPath := filepath.Join(tempDir, ".claude", "skills", "pagen", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("Failed to read SKILL.md: %v", err)
	}

	contentStr := string(content)

	// Verify key content markers from the embedded SKILL.md
	expectedMarkers := []string{
		"---",
		"name: pagen",
		"description:",
		"# pagen - Personal CRM",
		"pagen list",
		"pagen show",
		"pagen add",
	}

	for _, marker := range expectedMarkers {
		if !strings.Contains(contentStr, marker) {
			t.Errorf("SKILL.md missing expected content: %q", marker)
		}
	}
}

// TestInstallSkillCommand_Overwrite verifies that running install twice
// successfully overwrites the existing file.
func TestInstallSkillCommand_Overwrite(t *testing.T) {
	tempDir := t.TempDir()

	// First install
	err := installSkillToDir(tempDir, true)
	if err != nil {
		t.Fatalf("First installSkillToDir failed: %v", err)
	}

	skillPath := filepath.Join(tempDir, ".claude", "skills", "pagen", "SKILL.md")

	// Get initial file info
	initialInfo, err := os.Stat(skillPath)
	if err != nil {
		t.Fatalf("Failed to stat initial file: %v", err)
	}
	initialSize := initialInfo.Size()

	// Modify the file to verify overwrite works
	modifiedContent := []byte("modified content for testing overwrite")
	if err := os.WriteFile(skillPath, modifiedContent, 0644); err != nil {
		t.Fatalf("Failed to modify skill file: %v", err)
	}

	// Second install (overwrite)
	err = installSkillToDir(tempDir, true)
	if err != nil {
		t.Fatalf("Second installSkillToDir failed: %v", err)
	}

	// Verify the file was overwritten with original content
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("Failed to read SKILL.md after overwrite: %v", err)
	}

	// Content should not be the modified content
	if string(content) == string(modifiedContent) {
		t.Error("SKILL.md was not overwritten - still contains modified content")
	}

	// Content should contain original markers
	if !strings.Contains(string(content), "name: pagen") {
		t.Error("SKILL.md does not contain expected original content after overwrite")
	}

	// File size should match original
	newInfo, err := os.Stat(skillPath)
	if err != nil {
		t.Fatalf("Failed to stat file after overwrite: %v", err)
	}
	if newInfo.Size() != initialSize {
		t.Errorf("File size mismatch after overwrite: got %d, want %d", newInfo.Size(), initialSize)
	}
}

// TestInstallSkillCommand_FilePermissions verifies that the installed file
// has appropriate permissions.
func TestInstallSkillCommand_FilePermissions(t *testing.T) {
	tempDir := t.TempDir()

	err := installSkillToDir(tempDir, true)
	if err != nil {
		t.Fatalf("installSkillToDir failed: %v", err)
	}

	skillPath := filepath.Join(tempDir, ".claude", "skills", "pagen", "SKILL.md")
	info, err := os.Stat(skillPath)
	if err != nil {
		t.Fatalf("Failed to stat SKILL.md: %v", err)
	}

	// Check file is readable (at minimum)
	mode := info.Mode()
	if mode&0400 == 0 {
		t.Error("SKILL.md is not readable by owner")
	}
	if mode&0200 == 0 {
		t.Error("SKILL.md is not writable by owner")
	}
}

// TestInstallSkillCommand_DirectoryPermissions verifies that created directories
// have appropriate permissions.
func TestInstallSkillCommand_DirectoryPermissions(t *testing.T) {
	tempDir := t.TempDir()

	err := installSkillToDir(tempDir, true)
	if err != nil {
		t.Fatalf("installSkillToDir failed: %v", err)
	}

	pagenDir := filepath.Join(tempDir, ".claude", "skills", "pagen")
	info, err := os.Stat(pagenDir)
	if err != nil {
		t.Fatalf("Failed to stat pagen directory: %v", err)
	}

	// Check directory is accessible
	mode := info.Mode()
	if mode&0700 == 0 {
		t.Error("pagen directory does not have owner rwx permissions")
	}
}

// TestInstallSkillCommand_ExistingPartialStructure verifies installation works
// when some directories already exist.
func TestInstallSkillCommand_ExistingPartialStructure(t *testing.T) {
	tempDir := t.TempDir()

	// Pre-create partial directory structure
	claudeDir := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude directory: %v", err)
	}

	// Install should complete successfully
	err := installSkillToDir(tempDir, true)
	if err != nil {
		t.Fatalf("installSkillToDir failed with existing partial structure: %v", err)
	}

	// Verify file was created
	skillPath := filepath.Join(tempDir, ".claude", "skills", "pagen", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Error("SKILL.md was not created with existing partial structure")
	}
}

// TestInstallSkillCommand_ReadOnlyDirectory verifies appropriate error handling
// when the target directory cannot be created.
func TestInstallSkillCommand_ReadOnlyDirectory(t *testing.T) {
	// Skip on platforms where this test may not work as expected
	if os.Getuid() == 0 {
		t.Skip("Skipping test: running as root")
	}

	tempDir := t.TempDir()

	// Create .claude directory as read-only
	claudeDir := filepath.Join(tempDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0555); err != nil {
		t.Fatalf("Failed to create read-only .claude directory: %v", err)
	}

	// Attempt to install should fail
	err := installSkillToDir(tempDir, true)
	if err == nil {
		t.Error("Expected error when installing to read-only directory, got nil")
	}

	// Verify error message is helpful
	if err != nil && !strings.Contains(err.Error(), "failed to create skill directory") {
		t.Errorf("Expected error about directory creation, got: %v", err)
	}
}
