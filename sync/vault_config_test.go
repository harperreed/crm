// ABOUTME: Tests for vault configuration management and credential handling
// ABOUTME: Covers XDG path handling, config persistence, and device ID generation
package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVaultConfigDir(t *testing.T) {
	dir := VaultConfigDir()

	expectedBase := filepath.Join(xdg.DataHome, "pagen")
	assert.Equal(t, expectedBase, dir, "VaultConfigDir should return XDG data home path")
}

func TestVaultConfigPath(t *testing.T) {
	path := VaultConfigPath()

	expectedBase := filepath.Join(xdg.DataHome, "pagen")
	assert.True(t, strings.HasPrefix(path, expectedBase), "path should be under XDG data home")
	assert.Equal(t, "vault-config.json", filepath.Base(path), "config filename should be vault-config.json")
}

func TestLoadVaultConfig_NotFound(t *testing.T) {
	// Use temp dir to ensure config doesn't exist
	origHome := xdg.DataHome
	tmpDir := t.TempDir()
	xdg.DataHome = tmpDir
	defer func() { xdg.DataHome = origHome }()

	cfg, err := LoadVaultConfig()
	require.NoError(t, err, "LoadVaultConfig should not error when file not found")
	require.NotNil(t, cfg, "should return non-nil config")

	// Check defaults
	assert.Equal(t, "vault.db", cfg.VaultDB, "should have default VaultDB")
	assert.Empty(t, cfg.Server, "Server should be empty")
	assert.Empty(t, cfg.UserID, "UserID should be empty")
	assert.Empty(t, cfg.Token, "Token should be empty")
	assert.Empty(t, cfg.DerivedKey, "DerivedKey should be empty")
	assert.Empty(t, cfg.DeviceID, "DeviceID should be empty")
	assert.True(t, cfg.AutoSync, "AutoSync should default to true")
}

func TestSaveAndLoadVaultConfig(t *testing.T) {
	// Use temp dir for config
	origHome := xdg.DataHome
	tmpDir := t.TempDir()
	xdg.DataHome = tmpDir
	defer func() { xdg.DataHome = origHome }()

	// Create test config
	original := &VaultConfig{
		Server:       "https://vault.example.com",
		UserID:       "user123",
		Token:        "token456",
		RefreshToken: "refresh789",
		TokenExpires: "2025-12-31T23:59:59Z",
		DerivedKey:   "deadbeef",
		DeviceID:     "device001",
		VaultDB:      "custom.db",
		AutoSync:     true,
	}

	// Save config
	err := SaveVaultConfig(original)
	require.NoError(t, err, "SaveVaultConfig should succeed")

	// Verify file was created
	configPath := VaultConfigPath()
	_, err = os.Stat(configPath)
	require.NoError(t, err, "config file should exist")

	// Verify file permissions (should be user-only)
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	mode := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0600), mode, "config file should have 0600 permissions")

	// Load config back
	loaded, err := LoadVaultConfig()
	require.NoError(t, err, "LoadVaultConfig should succeed")
	require.NotNil(t, loaded)

	// Verify all fields match
	assert.Equal(t, original.Server, loaded.Server)
	assert.Equal(t, original.UserID, loaded.UserID)
	assert.Equal(t, original.Token, loaded.Token)
	assert.Equal(t, original.RefreshToken, loaded.RefreshToken)
	assert.Equal(t, original.TokenExpires, loaded.TokenExpires)
	assert.Equal(t, original.DerivedKey, loaded.DerivedKey)
	assert.Equal(t, original.DeviceID, loaded.DeviceID)
	assert.Equal(t, original.VaultDB, loaded.VaultDB)
	assert.Equal(t, original.AutoSync, loaded.AutoSync)
}

func TestVaultConfigIsConfigured(t *testing.T) {
	tests := []struct {
		name     string
		config   *VaultConfig
		expected bool
	}{
		{
			name:     "empty config",
			config:   &VaultConfig{},
			expected: false,
		},
		{
			name: "missing server",
			config: &VaultConfig{
				DerivedKey: "key",
				Token:      "token",
				UserID:     "user",
			},
			expected: false,
		},
		{
			name: "missing token",
			config: &VaultConfig{
				DerivedKey: "key",
				Server:     "server",
				UserID:     "user",
			},
			expected: false,
		},
		{
			name: "missing user ID",
			config: &VaultConfig{
				DerivedKey: "key",
				Token:      "token",
				Server:     "server",
			},
			expected: false,
		},
		{
			name: "missing derived key",
			config: &VaultConfig{
				Token:    "token",
				Server:   "server",
				UserID:   "user",
				DeviceID: "device",
			},
			expected: false,
		},
		{
			name: "missing device ID (v0.3+ required)",
			config: &VaultConfig{
				DerivedKey: "key",
				Token:      "token",
				Server:     "server",
				UserID:     "user",
			},
			expected: false,
		},
		{
			name: "fully configured",
			config: &VaultConfig{
				DerivedKey: "key",
				Token:      "token",
				Server:     "server",
				UserID:     "user",
				DeviceID:   "device",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsConfigured()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateVaultDeviceID(t *testing.T) {
	deviceID := GenerateVaultDeviceID()

	// Should not be empty
	assert.NotEmpty(t, deviceID, "device ID should not be empty")

	// Should be a valid ULID
	_, err := ulid.Parse(deviceID)
	require.NoError(t, err, "device ID should be a valid ULID")

	// Generate another one - should be different
	deviceID2 := GenerateVaultDeviceID()
	assert.NotEqual(t, deviceID, deviceID2, "successive device IDs should be unique")
}

func TestLoadVaultConfig_EnvOverrides(t *testing.T) {
	// Use temp dir for config
	origHome := xdg.DataHome
	tmpDir := t.TempDir()
	xdg.DataHome = tmpDir
	defer func() { xdg.DataHome = origHome }()

	// Create base config
	baseConfig := &VaultConfig{
		Server:   "https://file.example.com",
		UserID:   "file-user",
		Token:    "file-token",
		DeviceID: "file-device",
		VaultDB:  "vault.db",
		AutoSync: false,
	}
	err := SaveVaultConfig(baseConfig)
	require.NoError(t, err)

	// Set environment variables (t.Setenv auto-cleans up after test)
	t.Setenv("PAGEN_VAULT_SERVER", "https://env.example.com")
	t.Setenv("PAGEN_VAULT_USER_ID", "env-user")
	t.Setenv("PAGEN_VAULT_TOKEN", "env-token")
	t.Setenv("PAGEN_VAULT_DEVICE_ID", "env-device")
	t.Setenv("PAGEN_VAULT_AUTO_SYNC", "true")

	// Load config - env vars should override file values
	cfg, err := LoadVaultConfig()
	require.NoError(t, err)

	assert.Equal(t, "https://env.example.com", cfg.Server, "Server should be overridden by env")
	assert.Equal(t, "env-user", cfg.UserID, "UserID should be overridden by env")
	assert.Equal(t, "env-token", cfg.Token, "Token should be overridden by env")
	assert.Equal(t, "env-device", cfg.DeviceID, "DeviceID should be overridden by env")
	assert.True(t, cfg.AutoSync, "AutoSync should be overridden by env")
}

func TestLoadVaultConfig_InvalidJSON(t *testing.T) {
	// Use temp dir for config
	origHome := xdg.DataHome
	tmpDir := t.TempDir()
	xdg.DataHome = tmpDir
	defer func() { xdg.DataHome = origHome }()

	// Create invalid JSON file
	configDir := filepath.Join(tmpDir, "pagen")
	err := os.MkdirAll(configDir, 0700)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "vault-config.json")
	err = os.WriteFile(configPath, []byte("invalid json {{{"), 0600)
	require.NoError(t, err)

	// Loading should fail
	_, err = LoadVaultConfig()
	assert.Error(t, err, "LoadVaultConfig should error on invalid JSON")
	assert.Contains(t, err.Error(), "failed to decode", "error should mention decoding failure")
}

func TestSaveVaultConfig_JSONFormatting(t *testing.T) {
	// Use temp dir for config
	origHome := xdg.DataHome
	tmpDir := t.TempDir()
	xdg.DataHome = tmpDir
	defer func() { xdg.DataHome = origHome }()

	cfg := &VaultConfig{
		Server:   "https://vault.example.com",
		UserID:   "user123",
		Token:    "token456",
		VaultDB:  "vault.db",
		AutoSync: true,
	}

	err := SaveVaultConfig(cfg)
	require.NoError(t, err)

	// Read the raw JSON file
	configPath := VaultConfigPath()
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// Check that it's valid JSON
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Check that it contains indentation (formatted, not compact)
	dataStr := string(data)
	assert.Contains(t, dataStr, "\n  ", "JSON should be formatted with indentation")
	assert.Contains(t, dataStr, "\"server\":", "should contain server field")
	assert.Contains(t, dataStr, "\"vault_db\":", "should contain vault_db field")
}

func TestVaultConfig_OmitEmptyFields(t *testing.T) {
	cfg := &VaultConfig{
		Server:   "https://vault.example.com",
		UserID:   "user123",
		Token:    "token456",
		VaultDB:  "vault.db",
		AutoSync: true,
		// RefreshToken and TokenExpires intentionally not set
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	// Parse back to check fields
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// RefreshToken and TokenExpires should be omitted
	_, hasRefreshToken := parsed["refresh_token"]
	_, hasTokenExpires := parsed["token_expires"]

	assert.False(t, hasRefreshToken, "refresh_token should be omitted when empty")
	assert.False(t, hasTokenExpires, "token_expires should be omitted when empty")
}
