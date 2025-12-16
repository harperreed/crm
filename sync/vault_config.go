// ABOUTME: Vault configuration and credential management for suite sync integration
// ABOUTME: Handles vault config storage at XDG paths, environment variable overrides, and device ID generation
package sync

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/oklog/ulid/v2"
)

// VaultConfig stores vault server credentials and synchronization settings.
type VaultConfig struct {
	Server       string `json:"server"`
	UserID       string `json:"user_id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenExpires string `json:"token_expires,omitempty"`
	DerivedKey   string `json:"derived_key"` // hex-encoded seed, NOT the mnemonic
	DeviceID     string `json:"device_id"`
	VaultDB      string `json:"vault_db"`
	AutoSync     bool   `json:"auto_sync"`
}

// VaultConfigDir returns XDG-compliant directory for vault configuration.
func VaultConfigDir() string {
	return filepath.Join(xdg.DataHome, "pagen")
}

// VaultConfigPath returns XDG-compliant path for storing vault configuration.
func VaultConfigPath() string {
	return filepath.Join(VaultConfigDir(), "vault-config.json")
}

// LoadVaultConfig loads vault configuration from XDG data directory.
// Returns empty config with default VaultDB if file not found.
// Environment variables override file values:
// - PAGEN_VAULT_SERVER
// - PAGEN_VAULT_TOKEN
// - PAGEN_VAULT_USER_ID
// - PAGEN_VAULT_DEVICE_ID
// - PAGEN_VAULT_AUTO_SYNC.
func LoadVaultConfig() (*VaultConfig, error) {
	path := VaultConfigPath()

	// Initialize with defaults
	cfg := &VaultConfig{
		VaultDB:  "vault.db",
		AutoSync: true,
	}

	// Try to load from file
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - return default config
			applyEnvOverrides(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to open vault config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to decode vault config: %w", err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides to vault config.
func applyEnvOverrides(cfg *VaultConfig) {
	if server := os.Getenv("PAGEN_VAULT_SERVER"); server != "" {
		cfg.Server = server
	}
	if token := os.Getenv("PAGEN_VAULT_TOKEN"); token != "" {
		cfg.Token = token
	}
	if userID := os.Getenv("PAGEN_VAULT_USER_ID"); userID != "" {
		cfg.UserID = userID
	}
	if deviceID := os.Getenv("PAGEN_VAULT_DEVICE_ID"); deviceID != "" {
		cfg.DeviceID = deviceID
	}
	if autoSync := os.Getenv("PAGEN_VAULT_AUTO_SYNC"); autoSync != "" {
		cfg.AutoSync = autoSync == "true" || autoSync == "1"
	}
}

// SaveVaultConfig saves vault configuration to XDG data directory.
func SaveVaultConfig(cfg *VaultConfig) error {
	path := VaultConfigPath()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create vault config directory: %w", err)
	}

	// Write config file with restricted permissions
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create vault config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode vault config: %w", err)
	}

	return nil
}

// IsConfigured checks if vault is properly configured with required credentials.
// v0.3+ requires DeviceID for all authenticated requests.
func (c *VaultConfig) IsConfigured() bool {
	return c.DerivedKey != "" && c.Token != "" && c.Server != "" && c.UserID != "" && c.DeviceID != ""
}

// GenerateVaultDeviceID generates a new ULID for device identification.
func GenerateVaultDeviceID() string {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}
