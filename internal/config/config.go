// ABOUTME: Application configuration with XDG-compliant paths and backend selection.
// ABOUTME: Loads JSON config from XDG config home and provides a storage factory.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harperreed/crm/internal/storage"
)

// Config holds application-level settings persisted as JSON.
type Config struct {
	Backend string `json:"backend,omitempty"` // "sqlite" or "markdown", default "sqlite"
	DataDir string `json:"data_dir,omitempty"`
}

// GetBackend returns the configured storage backend, defaulting to "sqlite".
func (c *Config) GetBackend() string {
	if c.Backend == "" {
		return "sqlite"
	}
	return c.Backend
}

// GetDataDir returns the effective data directory, expanding ~ to the user's
// home directory. Falls back to the XDG data directory when no override is set.
func (c *Config) GetDataDir() string {
	if c.DataDir != "" {
		return ExpandPath(c.DataDir)
	}
	return storage.DataDir()
}

// ExpandPath replaces a leading ~ with the user's home directory.
func ExpandPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

// OpenStorage creates and returns a Storage implementation based on the
// configured backend.
func (c *Config) OpenStorage() (storage.Storage, error) {
	switch c.GetBackend() {
	case "sqlite":
		dbPath := filepath.Join(c.GetDataDir(), "crm.db")
		return storage.NewSqliteStore(dbPath)
	case "markdown":
		// NewMarkdownStore returns an error until the markdown backend is implemented.
		_, err := storage.NewMarkdownStore(c.GetDataDir())
		return nil, err
	default:
		return nil, fmt.Errorf("unknown storage backend: %q", c.GetBackend())
	}
}

// GetConfigPath returns the path to the CRM config file under XDG config home.
func GetConfigPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "crm", "config.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "crm", "config.json")
}

// Load reads the config file from the XDG config path. If the file does not
// exist, it returns a zero-value Config (which defaults to sqlite backend).
func Load() (*Config, error) {
	path := filepath.Clean(GetConfigPath())
	data, err := os.ReadFile(path) //nolint:gosec // path is derived from XDG env or home dir, not user input
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}
