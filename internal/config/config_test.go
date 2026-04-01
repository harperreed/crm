// ABOUTME: Tests for CRM configuration loading, path expansion, and backend factory.
// ABOUTME: Covers defaults, custom data dirs, tilde expansion, and storage initialization.
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := &Config{}

	if got := cfg.GetBackend(); got != "sqlite" {
		t.Errorf("GetBackend() = %q, want %q", got, "sqlite")
	}

	// DataDir should fall back to XDG data home
	t.Setenv("XDG_DATA_HOME", "/tmp/test-xdg-data")
	expected := filepath.Join("/tmp/test-xdg-data", "crm")
	if got := cfg.GetDataDir(); got != expected {
		t.Errorf("GetDataDir() = %q, want %q", got, expected)
	}
}

func TestGetDataDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/test-data")
	cfg := &Config{}

	expected := filepath.Join("/tmp/test-data", "crm")
	if got := cfg.GetDataDir(); got != expected {
		t.Errorf("GetDataDir() = %q, want %q", got, expected)
	}
}

func TestGetDataDirCustom(t *testing.T) {
	cfg := &Config{DataDir: "/custom/path"}

	if got := cfg.GetDataDir(); got != "/custom/path" {
		t.Errorf("GetDataDir() = %q, want %q", got, "/custom/path")
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/Documents", filepath.Join(home, "Documents")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}
	for _, tt := range tests {
		got := ExpandPath(tt.input)
		if got != tt.want {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestOpenStorageSqlite(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Backend: "sqlite", DataDir: dir}

	s, err := cfg.OpenStorage()
	if err != nil {
		t.Fatalf("OpenStorage: %v", err)
	}
	defer func() { _ = s.Close() }()

	// Verify the database file was created
	dbPath := filepath.Join(dir, "crm.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("expected database file at %q", dbPath)
	}
}

func TestOpenStorageUnknown(t *testing.T) {
	cfg := &Config{Backend: "nosql"}

	_, err := cfg.OpenStorage()
	if err == nil {
		t.Fatal("expected error for unknown backend, got nil")
	}
}
