// ABOUTME: Charm KV client wrapper using transactional Do API
// ABOUTME: Short-lived connections to avoid lock contention with other MCP servers

package charm

import (
	"fmt"
	"os"

	"github.com/charmbracelet/charm/client"
	"github.com/charmbracelet/charm/kv"
	charmproto "github.com/charmbracelet/charm/proto"
)

// Client holds configuration for KV operations.
// Unlike the previous implementation, it does NOT hold a persistent connection.
// Each operation opens the database, performs the operation, and closes it.
type Client struct {
	dbName     string
	autoSync   bool
	testClient *testClient // Used for testing without server dependency
}

// Option configures a Client.
type Option func(*Client)

// WithDBName sets the database name.
func WithDBName(name string) Option {
	return func(c *Client) {
		c.dbName = name
	}
}

// WithAutoSync enables or disables auto-sync after writes.
func WithAutoSync(enabled bool) Option {
	return func(c *Client) {
		c.autoSync = enabled
	}
}

// NewClient creates a new client with the given options.
func NewClient(opts ...Option) (*Client, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// Set charm host if configured
	if cfg.Host != "" {
		if err := os.Setenv("CHARM_HOST", cfg.Host); err != nil {
			return nil, err
		}
	}

	c := &Client{
		dbName:   AppName,
		autoSync: cfg.AutoSync,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// Get retrieves a value by key (read-only, no lock contention).
func (c *Client) Get(key []byte) ([]byte, error) {
	if c.testClient != nil {
		return c.testClient.Get(key)
	}

	var val []byte
	err := kv.DoReadOnly(c.dbName, func(k *kv.KV) error {
		var err error
		val, err = k.Get(key)
		return err
	})
	return val, err
}

// Set stores a value with the given key.
func (c *Client) Set(key, value []byte) error {
	if c.testClient != nil {
		return c.testClient.Set(key, value)
	}

	return kv.Do(c.dbName, func(k *kv.KV) error {
		if err := k.Set(key, value); err != nil {
			return err
		}
		if c.autoSync {
			return k.Sync()
		}
		return nil
	})
}

// Delete removes a key.
func (c *Client) Delete(key []byte) error {
	if c.testClient != nil {
		return c.testClient.Delete(key)
	}

	return kv.Do(c.dbName, func(k *kv.KV) error {
		if err := k.Delete(key); err != nil {
			return err
		}
		if c.autoSync {
			return k.Sync()
		}
		return nil
	})
}

// Keys returns all keys in the database.
func (c *Client) Keys() ([][]byte, error) {
	if c.testClient != nil {
		return c.testClient.Keys()
	}

	var keys [][]byte
	err := kv.DoReadOnly(c.dbName, func(k *kv.KV) error {
		var err error
		keys, err = k.Keys()
		return err
	})
	return keys, err
}

// KeysWithPrefix returns all keys starting with the given prefix.
func (c *Client) KeysWithPrefix(prefix []byte) ([][]byte, error) {
	if c.testClient != nil {
		return c.testClient.KeysWithPrefix(prefix)
	}

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

// DoReadOnly executes a function with read-only database access.
// Use this for batch read operations that need multiple Gets.
func (c *Client) DoReadOnly(fn func(k *kv.KV) error) error {
	if c.testClient != nil {
		// For test client, we don't have a real KV to pass
		// This is okay because test code should use the individual methods
		return fmt.Errorf("DoReadOnly not supported with test client")
	}
	return kv.DoReadOnly(c.dbName, fn)
}

// Do executes a function with write access to the database.
// Use this for batch write operations.
func (c *Client) Do(fn func(k *kv.KV) error) error {
	if c.testClient != nil {
		// For test client, we don't have a real KV to pass
		return fmt.Errorf("Do not supported with test client")
	}
	return kv.Do(c.dbName, func(k *kv.KV) error {
		if err := fn(k); err != nil {
			return err
		}
		if c.autoSync {
			return k.Sync()
		}
		return nil
	})
}

// Sync triggers a manual sync with the charm server.
func (c *Client) Sync() error {
	if c.testClient != nil {
		return nil // No-op for test client
	}
	return kv.Do(c.dbName, func(k *kv.KV) error {
		return k.Sync()
	})
}

// Reset clears all data (nuclear option).
func (c *Client) Reset() error {
	if c.testClient != nil {
		return c.testClient.Reset()
	}
	return kv.Do(c.dbName, func(k *kv.KV) error {
		return k.Reset()
	})
}

// ID returns the charm user ID for this device.
func (c *Client) ID() (string, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return "", err
	}
	return cc.ID()
}

// User returns the current charm user information.
func (c *Client) User() (*charmproto.User, error) {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return nil, err
	}
	return cc.Bio()
}

// Link initiates the charm linking process for this device.
func (c *Client) Link() error {
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return err
	}
	_, err = cc.Bio()
	return err
}

// Unlink removes the charm account association from this device.
func (c *Client) Unlink() error {
	return c.Reset()
}

// Config returns the current configuration.
func (c *Client) Config() *Config {
	if c.testClient != nil {
		return c.testClient.Config()
	}
	cfg, _ := LoadConfig()
	return cfg
}

// IsConnected checks if the client can connect to charm cloud.
func (c *Client) IsConnected() bool {
	_, err := c.ID()
	return err == nil
}

// --- Legacy compatibility layer ---
// These functions maintain backwards compatibility with existing code.

var globalClient *Client

// InitClient initializes the global charm client.
// With the new architecture, this just creates a Client instance.
func InitClient() error {
	if globalClient != nil {
		return nil
	}
	var err error
	globalClient, err = NewClient()
	return err
}

// GetClient returns the global client, initializing if needed.
func GetClient() (*Client, error) {
	if err := InitClient(); err != nil {
		return nil, err
	}
	return globalClient, nil
}

// ResetClient resets the global client singleton.
func ResetClient() error {
	globalClient = nil
	return nil
}

// Close is a no-op for backwards compatibility.
// With Do API, connections are automatically closed after each operation.
func (c *Client) Close() error {
	return nil
}
