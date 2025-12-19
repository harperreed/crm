// ABOUTME: Configuration for Charm KV backend connection
// ABOUTME: Handles server settings and auto-sync preferences

package charm

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

const (
	// DefaultCharmHost is the self-hosted 2389 research server.
	DefaultCharmHost = "charm.2389.dev"

	// AppName is the application name for Charm KV database.
	AppName = "pagen"

	// ConfigFileName is where we store local config.
	ConfigFileName = "charm-config.json"
)

// Config holds charm connection settings.
type Config struct {
	// Host is the charm server hostname (default: charm.2389.dev)
	Host string `json:"host,omitempty"`

	// AutoSync enables automatic sync after every write operation
	AutoSync bool `json:"auto_sync"`

	// Linked indicates the device has been linked via `charm link`
	Linked bool `json:"linked"`
}

// DefaultConfig returns a new config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Host:     DefaultCharmHost,
		AutoSync: true,
		Linked:   false,
	}
}

// configPath returns the path to the config file.
func configPath() (string, error) {
	dataDir := filepath.Join(xdg.DataHome, AppName)
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dataDir, ConfigFileName), nil
}

// LoadConfig loads config from disk, or returns defaults if not found.
func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		// Can't determine config path, use defaults
		return DefaultConfig(), nil //nolint:nilerr // Intentionally returning defaults on path error
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Invalid config, use defaults
		return DefaultConfig(), nil //nolint:nilerr // Intentionally returning defaults on parse error
	}

	// Apply defaults for missing fields
	if cfg.Host == "" {
		cfg.Host = DefaultCharmHost
	}

	return &cfg, nil
}

// Save persists the config to disk.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// IsConfigured returns true if the device has been linked.
func (c *Config) IsConfigured() bool {
	return c.Linked
}

// SetHost sets the charm server host and saves.
func (c *Config) SetHost(host string) error {
	c.Host = host
	return c.Save()
}

// SetAutoSync enables or disables auto-sync and saves.
func (c *Config) SetAutoSync(enabled bool) error {
	c.AutoSync = enabled
	return c.Save()
}

// MarkLinked marks the device as linked and saves.
func (c *Config) MarkLinked() error {
	c.Linked = true
	return c.Save()
}

// MarkUnlinked marks the device as unlinked and saves.
func (c *Config) MarkUnlinked() error {
	c.Linked = false
	return c.Save()
}
