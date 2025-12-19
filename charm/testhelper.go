// ABOUTME: Test utilities for creating isolated charm clients
// ABOUTME: Uses temporary directories with in-memory BadgerDB for test isolation

package charm

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/dgraph-io/badger/v3"
)

// testKV wraps BadgerDB to provide the same interface as charm/kv.KV
// for testing without requiring server connectivity.
type testKV struct {
	db *badger.DB
}

func (t *testKV) Get(key []byte) ([]byte, error) {
	var result []byte
	err := t.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		result, err = item.ValueCopy(nil)
		return err
	})
	return result, err
}

func (t *testKV) Set(key, value []byte) error {
	return t.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

func (t *testKV) Delete(key []byte) error {
	return t.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (t *testKV) Keys() ([][]byte, error) {
	var keys [][]byte
	err := t.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.KeyCopy(nil)
			keys = append(keys, key)
		}
		return nil
	})
	return keys, err
}

func (t *testKV) Sync() error {
	// No-op for test KV
	return nil
}

func (t *testKV) Reset() error {
	return t.db.DropAll()
}

func (t *testKV) Close() error {
	return t.db.Close()
}

// testClient wraps testKV to match the Client interface without the charm/kv dependency.
// The mutex provides thread safety for parallel test operations.
type testClient struct {
	tkv    *testKV
	config *Config
	mu     sync.RWMutex
}

func (c *testClient) Get(key []byte) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tkv.Get(key)
}

func (c *testClient) Set(key, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tkv.Set(key, value)
}

func (c *testClient) Delete(key []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tkv.Delete(key)
}

func (c *testClient) Keys() ([][]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tkv.Keys()
}

func (c *testClient) KeysWithPrefix(prefix []byte) ([][]byte, error) {
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

func (c *testClient) Config() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

func (c *testClient) Sync() error {
	return nil
}

func (c *testClient) Reset() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tkv.Reset()
}

// NewTestClient creates a charm client using a temporary directory for testing.
// The returned cleanup function should be deferred to remove the temp directory.
// This implementation uses BadgerDB directly, avoiding the charm server dependency.
func NewTestClient(t *testing.T) (*Client, func()) {
	t.Helper()

	// Create temp directory for test data
	tmpDir, err := os.MkdirTemp("", "pagen-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create the data directory
	dataDir := filepath.Join(tmpDir, "pagen")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create data dir: %v", err)
	}

	// Configure badger options for the temp directory
	opts := badger.DefaultOptions(dataDir).
		WithLogger(nil) // Suppress badger logs in tests

	// Open BadgerDB directly
	db, err := badger.Open(opts)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open badger: %v", err)
	}

	tkv := &testKV{db: db}

	// Create config with no sync for testing
	cfg := &Config{
		Host:     "localhost",
		AutoSync: false,
		Linked:   false,
	}

	// Create a wrapper that embeds testClient to satisfy the Client interface
	tc := &testClient{
		tkv:    tkv,
		config: cfg,
	}

	// Return a Client with nil kv but using the test implementation
	// We'll modify the Client struct to handle this
	c := newTestClientWrapper(tc)

	cleanup := func() {
		if db != nil {
			if err := db.Close(); err != nil {
				t.Logf("Warning: failed to close test database: %v", err)
			}
		}
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Warning: failed to remove temp directory %s: %v", tmpDir, err)
		}
	}

	return c, cleanup
}

// newTestClientWrapper creates a Client that uses the testClient for storage.
func newTestClientWrapper(tc *testClient) *Client {
	return &Client{
		kv:         nil, // Use test implementation
		config:     tc.config,
		testClient: tc, // Store test client
	}
}
