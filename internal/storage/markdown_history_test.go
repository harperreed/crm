// ABOUTME: Tests Markdown history directories, committed-event reads, prefix lookup, and limits.
// ABOUTME: Verifies corrupt event files fail loudly with their path and history sentinel.
package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownHistorySchema(t *testing.T) {
	store := newTestMarkdownStore(t)
	for _, dir := range []string{store.historyEventsDir(), store.historyPendingDir()} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("Stat(%q): %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%q is not a directory", dir)
		}
	}
}

func TestMarkdownListHistoryAndGetHistoryEvent(t *testing.T) {
	store := newTestMarkdownStore(t)
	runHistoryReadContract(t, historyReadBackend{
		list:  store.ListHistory,
		get:   store.GetHistoryEvent,
		write: store.writeCommittedHistoryEvent,
	})
}

func TestMarkdownListHistoryLimits(t *testing.T) {
	store := newTestMarkdownStore(t)
	runHistoryLimitContract(t, historyReadBackend{
		list:  store.ListHistory,
		write: store.writeCommittedHistoryEvent,
	})
}

func TestMarkdownGetHistoryEventReportsCorruptFile(t *testing.T) {
	store := newTestMarkdownStore(t)
	path := filepath.Join(store.historyEventsDir(), "bad-event.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":1,"extra":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := store.GetHistoryEvent("bad-ev")
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("GetHistoryEvent() error = %v, want ErrHistoryCorrupt", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("GetHistoryEvent() error = %q, want path %q", err, path)
	}
}

func TestMarkdownGetHistoryEventRejectsUnknownFields(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("writeCommittedHistoryEvent: %v", err)
	}
	path := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	data, err := os.ReadFile(path) //nolint:gosec // path is built from the temporary test store.
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	data = []byte(strings.Replace(string(data), `"schema_version": 1`, `"schema_version": 1, "unknown": true`, 1))
	if err := os.WriteFile(path, data, 0o600); err != nil { //nolint:gosec // path is built from the temporary test store.
		t.Fatalf("WriteFile: %v", err)
	}

	_, err = store.GetHistoryEvent(event.ID.String())
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("GetHistoryEvent() error = %v, want ErrHistoryCorrupt", err)
	}
}

func TestMarkdownGetHistoryEventRejectsMismatchedFilename(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("writeCommittedHistoryEvent: %v", err)
	}
	original := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	mismatched := filepath.Join(store.historyEventsDir(), "80000100-0000-0000-0000-000000000008.json")
	if err := os.Rename(original, mismatched); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	_, err := store.GetHistoryEvent(event.ID.String())
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("GetHistoryEvent() error = %v, want ErrHistoryCorrupt", err)
	}
	if !strings.Contains(err.Error(), mismatched) {
		t.Fatalf("GetHistoryEvent() error = %q, want path %q", err, mismatched)
	}
}

func TestMarkdownGetHistoryEventRejectsSymlink(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("writeCommittedHistoryEvent: %v", err)
	}
	committedPath := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	externalPath := filepath.Join(t.TempDir(), event.ID.String()+".json")
	if err := os.Rename(committedPath, externalPath); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if err := os.Symlink(externalPath, committedPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	_, err := store.GetHistoryEvent(event.ID.String())
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("GetHistoryEvent() error = %v, want ErrHistoryCorrupt", err)
	}
}
