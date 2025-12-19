// ABOUTME: Charm KV client wrapper with automatic sync support
// ABOUTME: Thread-safe singleton initialization using sync.Once

package charm

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/kv"
)

var (
	globalClient *Client
	clientOnce   sync.Once
	clientErr    error
)

// connectionCheckTTL is how long to cache the connection status.
const connectionCheckTTL = 30 * time.Second

// Client wraps charm KV with config and sync helpers.
type Client struct {
	kv         *kv.KV
	config     *Config
	mu         sync.RWMutex
	testClient *testClient // Used for testing without server dependency

	// Cached connection state to avoid repeated network calls
	cachedConnected   bool
	cachedConnectedAt time.Time
	cachedUserID      string
}

// InitClient initializes the global charm client (thread-safe, only runs once).
func InitClient() error {
	clientOnce.Do(func() {
		cfg, err := LoadConfig()
		if err != nil {
			clientErr = fmt.Errorf("failed to load config: %w", err)
			return
		}

		// Set charm host before opening KV
		_ = os.Setenv("CHARM_HOST", cfg.Host)

		db, err := kv.OpenWithDefaultsFallback(AppName)
		if err != nil {
			clientErr = fmt.Errorf("failed to open charm kv: %w", err)
			return
		}

		globalClient = &Client{
			kv:     db,
			config: cfg,
		}

		// Sync on startup to pull remote changes (skip if read-only)
		if cfg.AutoSync && !db.IsReadOnly() {
			_ = db.Sync()
		}
	})
	return clientErr
}

// GetClient returns the global client, initializing if needed.
func GetClient() (*Client, error) {
	if err := InitClient(); err != nil {
		return nil, err
	}
	if globalClient == nil {
		return nil, fmt.Errorf("client not initialized")
	}
	return globalClient, nil
}

// NewClient creates a fresh client (for testing or special cases).
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	_ = os.Setenv("CHARM_HOST", cfg.Host)

	db, err := kv.OpenWithDefaultsFallback(AppName)
	if err != nil {
		return nil, fmt.Errorf("failed to open charm kv: %w", err)
	}

	c := &Client{
		kv:     db,
		config: cfg,
	}

	// Sync on startup to pull remote changes (skip if read-only)
	if cfg.AutoSync && !db.IsReadOnly() {
		_ = db.Sync()
	}

	return c, nil
}

// Close closes the KV store.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Note: charm/kv doesn't expose Close() directly
	// The underlying BadgerDB will be cleaned up on process exit
	return nil
}

// Config returns a copy of the client's config (thread-safe).
func (c *Client) Config() *Config {
	if c.testClient != nil {
		return c.testClient.Config()
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Return a copy to prevent race conditions
	return &Config{
		Host:     c.config.Host,
		AutoSync: c.config.AutoSync,
	}
}

// ID returns the charm user ID for this device.
// Results are cached to avoid repeated network calls.
func (c *Client) ID() (string, error) {
	if c.testClient != nil {
		return "test-user-id", nil
	}

	c.mu.RLock()
	// Return cached ID if still valid
	if c.cachedUserID != "" && time.Since(c.cachedConnectedAt) < connectionCheckTTL {
		c.mu.RUnlock()
		return c.cachedUserID, nil
	}
	c.mu.RUnlock()

	// Need to refresh - acquire write lock
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.cachedUserID != "" && time.Since(c.cachedConnectedAt) < connectionCheckTTL {
		return c.cachedUserID, nil
	}

	// Make the network call
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		c.cachedConnected = false
		c.cachedConnectedAt = time.Now()
		c.cachedUserID = ""
		return "", fmt.Errorf("failed to create charm client: %w", err)
	}

	id, err := cc.ID()
	if err != nil {
		c.cachedConnected = false
		c.cachedConnectedAt = time.Now()
		c.cachedUserID = ""
		return "", err
	}

	// Cache the successful result
	c.cachedConnected = true
	c.cachedConnectedAt = time.Now()
	c.cachedUserID = id
	return id, nil
}

// IsConnected checks if the client can connect to charm cloud.
// Uses cached connection status to avoid repeated network calls.
func (c *Client) IsConnected() bool {
	if c.testClient != nil {
		return true // Test client is always "connected"
	}

	c.mu.RLock()
	// Return cached status if still valid
	if !c.cachedConnectedAt.IsZero() && time.Since(c.cachedConnectedAt) < connectionCheckTTL {
		connected := c.cachedConnected
		c.mu.RUnlock()
		return connected
	}
	c.mu.RUnlock()

	// Cache is stale, refresh by calling ID()
	_, err := c.ID()
	return err == nil
}

// IsReadOnly returns true if the database is open in read-only mode.
// This happens when another process (like an MCP server) holds the lock.
func (c *Client) IsReadOnly() bool {
	if c.testClient != nil {
		return false // Test client is never read-only
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.kv.IsReadOnly()
}

// Sync performs a manual sync with the charm server.
func (c *Client) Sync() error {
	if c.testClient != nil {
		return nil // No-op for test client
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.kv.IsReadOnly() {
		return c.kv.Sync()
	}
	return nil
}

// Get retrieves a value by key.
func (c *Client) Get(key []byte) ([]byte, error) {
	if c.testClient != nil {
		return c.testClient.Get(key)
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.kv.Get(key)
}

// Set stores a value and syncs if enabled.
func (c *Client) Set(key, value []byte) error {
	if c.testClient != nil {
		return c.testClient.Set(key, value)
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.kv.IsReadOnly() {
		return fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}

	if err := c.kv.Set(key, value); err != nil {
		return err
	}

	// Sync while still holding lock to avoid race condition
	if c.config.AutoSync {
		_ = c.kv.Sync()
	}
	return nil
}

// Delete removes a key and syncs if enabled.
func (c *Client) Delete(key []byte) error {
	if c.testClient != nil {
		return c.testClient.Delete(key)
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.kv.IsReadOnly() {
		return fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}

	if err := c.kv.Delete(key); err != nil {
		return err
	}

	// Sync while still holding lock to avoid race condition
	if c.config.AutoSync {
		_ = c.kv.Sync()
	}
	return nil
}

// Keys returns all keys (for debugging/admin).
func (c *Client) Keys() ([][]byte, error) {
	if c.testClient != nil {
		return c.testClient.Keys()
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.kv.Keys()
}

// KeysWithPrefix returns all keys starting with the given prefix.
func (c *Client) KeysWithPrefix(prefix []byte) ([][]byte, error) {
	allKeys, err := c.Keys()
	if err != nil {
		return nil, err
	}

	var matched [][]byte
	for _, k := range allKeys {
		if len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix) {
			matched = append(matched, k)
		}
	}
	return matched, nil
}

// Reset wipes all data from the KV store (use with caution!)
func (c *Client) Reset() error {
	if c.testClient != nil {
		return c.testClient.Reset()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.kv.IsReadOnly() {
		return fmt.Errorf("cannot write: database is locked by another process (MCP server?)")
	}
	return c.kv.Reset()
}
