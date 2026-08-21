// ABOUTME: Exercises user-visible behavior of the root CRM command.
// ABOUTME: Verifies Cobra's version flag reports the configured build version.

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/harperreed/crm/internal/history"
	"github.com/spf13/cobra"
)

func TestHistorySourceForCommand(t *testing.T) {
	tests := []struct {
		name string
		want history.Source
	}{
		{name: "mcp", want: history.SourceMCP},
		{name: "contact", want: history.SourceCLI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: tt.name}
			if got := historySourceForCommand(cmd); got != tt.want {
				t.Fatalf("historySourceForCommand(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestRootVersionFlag(t *testing.T) {
	var output bytes.Buffer
	var errorOutput bytes.Buffer
	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "config")
	dataHome := filepath.Join(tempDir, "data")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", dataHome)
	rootCmd.SetArgs([]string{"--version"})
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&errorOutput)
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		versionFlag := rootCmd.Flags().Lookup("version")
		if versionFlag == nil {
			t.Error("version flag missing during cleanup")
			return
		}
		if err := versionFlag.Value.Set("false"); err != nil {
			t.Errorf("reset version flag: %v", err)
		}
		versionFlag.Changed = false
	})

	if err := Execute(); err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	want := fmt.Sprintf("crm version %s\n", version)
	if got := output.String(); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
	if got := errorOutput.String(); got != "" {
		t.Errorf("error output = %q, want empty", got)
	}
	for _, path := range []string{configHome, dataHome} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("os.Stat(%q) error = %v, want path not to exist", path, err)
		}
	}
}
