// ABOUTME: Tests Markdown history directories, committed-event reads, prefix lookup, and limits.
// ABOUTME: Verifies corrupt event files fail loudly with their path and history sentinel.
package storage

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harperreed/crm/internal/history"
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

func TestSyncMarkdownHistoryDirectory(t *testing.T) {
	t.Run("syncs existing directory", func(t *testing.T) {
		store := newTestMarkdownStore(t)
		if err := syncMarkdownHistoryDirectory(store.historyEventsDir()); err != nil {
			t.Fatalf("syncMarkdownHistoryDirectory: %v", err)
		}
	})

	t.Run("reports open error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing")
		err := syncMarkdownHistoryDirectory(path)
		if err == nil {
			t.Fatal("syncMarkdownHistoryDirectory() error = nil")
		}
		if !strings.Contains(err.Error(), "open history events directory") {
			t.Fatalf("syncMarkdownHistoryDirectory() error = %q", err)
		}
	})
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

func TestMarkdownWriteCommittedHistoryEventIsIdempotent(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("first writeCommittedHistoryEvent: %v", err)
	}
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("second writeCommittedHistoryEvent: %v", err)
	}
	assertNoMarkdownHistoryTempFiles(t, store)
}

func TestMarkdownWriteCommittedHistoryEventPreservesConflictingEvent(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("writeCommittedHistoryEvent: %v", err)
	}
	path := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	want, err := os.ReadFile(path) //nolint:gosec // path is built from the temporary test store.
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	conflicting := *event
	conflicting.Source = history.SourceMCP

	err = store.writeCommittedHistoryEvent(&conflicting)
	if !errors.Is(err, ErrHistoryConflict) {
		t.Fatalf("writeCommittedHistoryEvent() error = %v, want ErrHistoryConflict", err)
	}
	got, err := os.ReadFile(path) //nolint:gosec // path is built from the temporary test store.
	if err != nil {
		t.Fatalf("ReadFile after conflict: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("conflicting write changed committed event")
	}
	assertNoMarkdownHistoryTempFiles(t, store)
}

func TestMarkdownWriteCommittedHistoryEventPreservesCorruptEvent(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	path := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	want := []byte("truncated event")
	if err := os.WriteFile(path, want, 0o600); err != nil { //nolint:gosec // path is built from the temporary test store.
		t.Fatalf("WriteFile: %v", err)
	}

	err := store.writeCommittedHistoryEvent(event)
	if !errors.Is(err, ErrHistoryCorrupt) {
		t.Fatalf("writeCommittedHistoryEvent() error = %v, want ErrHistoryCorrupt", err)
	}
	got, err := os.ReadFile(path) //nolint:gosec // path is built from the temporary test store.
	if err != nil {
		t.Fatalf("ReadFile after corrupt conflict: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("write changed corrupt committed event")
	}
	assertNoMarkdownHistoryTempFiles(t, store)
}

func TestMarkdownWriteCommittedHistoryEventRejectsFinalSymlink(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	path := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	target := filepath.Join(t.TempDir(), "outside.json")
	want := []byte("outside target")
	if err := os.WriteFile(target, want, 0o600); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	if err := store.writeCommittedHistoryEvent(event); err == nil {
		t.Fatal("writeCommittedHistoryEvent() error = nil")
	}
	got, err := os.ReadFile(target) //nolint:gosec // target is built from a temporary test directory.
	if err != nil {
		t.Fatalf("ReadFile target: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("committed write changed symlink target")
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("committed write replaced final symlink")
	}
	assertNoMarkdownHistoryTempFiles(t, store)
}

func TestMarkdownWriteCommittedHistoryEventUsesPrivateMode(t *testing.T) {
	store := newTestMarkdownStore(t)
	event := testContactHistoryEvent(t, testHistoryEntityA, testHistoryEventA, testHistoryTime)
	if err := store.writeCommittedHistoryEvent(event); err != nil {
		t.Fatalf("writeCommittedHistoryEvent: %v", err)
	}
	path := filepath.Join(store.historyEventsDir(), event.ID.String()+".json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("committed mode = %04o, want 0600", got)
	}
	assertNoMarkdownHistoryTempFiles(t, store)
}

func TestMarkdownHistoryReadsIgnoreTemporaryFiles(t *testing.T) {
	store := newTestMarkdownStore(t)
	path := filepath.Join(store.historyEventsDir(), ".interrupted-event.tmp")
	if err := os.WriteFile(path, []byte(`{"partial":`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := store.ListHistory(testHistoryEntityA.String(), 0)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListHistory() len = %d, want 0", len(got))
	}
}

func assertNoMarkdownHistoryTempFiles(t *testing.T, store *MarkdownStore) {
	t.Helper()
	entries, err := os.ReadDir(store.historyEventsDir())
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			t.Fatalf("temporary history file remains: %s", entry.Name())
		}
	}
}
