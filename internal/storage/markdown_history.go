// ABOUTME: Stores committed Markdown-backend history events as strict JSON files.
// ABOUTME: Reads timelines with UUID prefix resolution, validation, ordering, and limits.
package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/history"
)

func (s *MarkdownStore) historyEventsDir() string {
	return filepath.Join(s.dataDir, "_history", "events")
}

func (s *MarkdownStore) historyPendingDir() string {
	return filepath.Join(s.dataDir, "_history", "pending")
}

func (s *MarkdownStore) readCommittedHistoryEvents() ([]*history.Event, error) {
	entries, err := os.ReadDir(s.historyEventsDir())
	if err != nil {
		return nil, fmt.Errorf("read committed history directory: %w", err)
	}
	events := make([]*history.Event, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.historyEventsDir(), entry.Name())
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%w: %s: symbolic links are not committed history events", ErrHistoryCorrupt, path)
		}
		data, err := os.ReadFile(path) //nolint:gosec // path comes from the private history directory.
		if err != nil {
			return nil, fmt.Errorf("read history event %q: %w", path, err)
		}
		event, err := decodeCommittedHistoryEvent(data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrHistoryCorrupt, path, err)
		}
		if entry.Name() != event.ID.String()+".json" {
			return nil, fmt.Errorf("%w: %s: filename does not match event ID %s", ErrHistoryCorrupt, path, event.ID)
		}
		events = append(events, event)
	}
	return events, nil
}

func decodeCommittedHistoryEvent(data []byte) (*history.Event, error) {
	var event history.Event
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		return nil, err
	}
	if err := ensureJSONDocumentEnd(decoder); err != nil {
		return nil, err
	}
	if err := event.Validate(); err != nil {
		return nil, err
	}
	return &event, nil
}

func ensureJSONDocumentEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("history event contains multiple JSON values")
		}
		return err
	}
	return nil
}

func (s *MarkdownStore) writeCommittedHistoryEvent(event *history.Event) (returnErr error) {
	data, err := marshalCommittedHistoryEvent(event)
	if err != nil {
		return err
	}

	temporary, err := os.CreateTemp(s.historyEventsDir(), "."+event.ID.String()+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary history event: %w", err)
	}
	temporaryPath := temporary.Name()
	temporaryOpen := true
	defer func() {
		if temporaryOpen {
			if err := temporary.Close(); err != nil {
				returnErr = errors.Join(returnErr, fmt.Errorf("close temporary history event during cleanup: %w", err))
			}
		}
		if err := os.Remove(temporaryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			returnErr = errors.Join(returnErr, fmt.Errorf("remove temporary history event: %w", err))
		}
	}()

	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("set temporary history event mode: %w", err)
	}
	written, err := temporary.Write(data)
	if err != nil {
		return fmt.Errorf("write temporary history event: %w", err)
	}
	if written != len(data) {
		return fmt.Errorf("write temporary history event: %w", io.ErrShortWrite)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary history event: %w", err)
	}
	closeErr := temporary.Close()
	temporaryOpen = false
	if closeErr != nil {
		return fmt.Errorf("close temporary history event: %w", closeErr)
	}

	path := filepath.Join(s.historyEventsDir(), event.ID.String()+".json")
	if err := os.Link(temporaryPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return compareCommittedHistoryEvent(path, event, data)
		}
		return fmt.Errorf("publish committed history event %q: %w", path, err)
	}
	return nil
}

func marshalCommittedHistoryEvent(event *history.Event) ([]byte, error) {
	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("validate history event: %w", err)
	}
	data, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode history event: %w", err)
	}
	data = append(data, '\n')
	return data, nil
}

func compareCommittedHistoryEvent(path string, expected *history.Event, expectedData []byte) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect existing committed history event %q: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: committed history path %q is a symbolic link", ErrHistoryConflict, path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: committed history path %q is not a regular file", ErrHistoryConflict, path)
	}

	existingData, err := readFileWithoutFollowingSymlinks(path)
	if err != nil {
		return fmt.Errorf("read existing committed history event %q: %w", path, err)
	}
	existing, err := decodeCommittedHistoryEvent(existingData)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrHistoryCorrupt, path, err)
	}
	if filepath.Base(path) != existing.ID.String()+".json" {
		return fmt.Errorf("%w: %s: filename does not match event ID %s", ErrHistoryCorrupt, path, existing.ID)
	}
	canonicalExisting, err := marshalCommittedHistoryEvent(existing)
	if err != nil {
		return fmt.Errorf("%w: canonicalize %s: %w", ErrHistoryCorrupt, path, err)
	}
	if bytes.Equal(canonicalExisting, expectedData) && existing.ID == expected.ID {
		return nil
	}
	return fmt.Errorf("%w: event ID %s already has different committed content", ErrHistoryConflict, expected.ID)
}

func readFileWithoutFollowingSymlinks(path string) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, errors.New("open committed history event")
	}
	defer func() { _ = file.Close() }()
	return io.ReadAll(file)
}

func (s *MarkdownStore) ListHistory(entityIDOrPrefix string, limit int) ([]*history.Summary, error) {
	normalizedLimit, err := history.NormalizeLimit(limit)
	if err != nil {
		return nil, err
	}
	if len(entityIDOrPrefix) < 6 {
		return nil, ErrPrefixTooShort
	}
	events, err := s.readCommittedHistoryEvents()
	if err != nil {
		return nil, err
	}
	entityID, err := resolveMarkdownHistoryEntityID(events, entityIDOrPrefix)
	if err != nil {
		return nil, err
	}
	if entityID == uuid.Nil {
		return []*history.Summary{}, nil
	}

	matching := make([]*history.Event, 0, len(events))
	for _, event := range events {
		if slices.Contains(event.RelatedEntityIDs, entityID) {
			matching = append(matching, event)
		}
	}
	sortHistoryEvents(matching)
	if len(matching) > normalizedLimit {
		matching = matching[:normalizedLimit]
	}
	summaries := make([]*history.Summary, 0, len(matching))
	for _, event := range matching {
		summary, err := event.Summary()
		if err != nil {
			return nil, fmt.Errorf("%w: summarize history event %s: %w", ErrHistoryCorrupt, event.ID, err)
		}
		summaries = append(summaries, &summary)
	}
	return summaries, nil
}

func resolveMarkdownHistoryEntityID(events []*history.Event, value string) (uuid.UUID, error) {
	if id, err := uuid.Parse(value); err == nil {
		return id, nil
	}
	value = normalizeHistoryIDPrefix(value)
	matches := make(map[uuid.UUID]struct{})
	for _, event := range events {
		for _, id := range event.RelatedEntityIDs {
			if strings.HasPrefix(id.String(), value) {
				matches[id] = struct{}{}
			}
		}
	}
	if len(matches) == 0 {
		return uuid.Nil, nil
	}
	if len(matches) > 1 {
		return uuid.Nil, ErrAmbiguousPrefix
	}
	for id := range matches {
		return id, nil
	}
	return uuid.Nil, nil
}

func (s *MarkdownStore) GetHistoryEvent(eventIDOrPrefix string) (*history.Event, error) {
	if len(eventIDOrPrefix) < 6 {
		return nil, ErrPrefixTooShort
	}
	events, err := s.readCommittedHistoryEvents()
	if err != nil {
		return nil, err
	}
	var matches []*history.Event
	if eventID, parseErr := uuid.Parse(eventIDOrPrefix); parseErr == nil {
		for _, event := range events {
			if event.ID == eventID {
				return event, nil
			}
		}
		return nil, ErrHistoryNotFound
	}
	eventIDOrPrefix = normalizeHistoryIDPrefix(eventIDOrPrefix)
	for _, event := range events {
		if strings.HasPrefix(event.ID.String(), eventIDOrPrefix) {
			matches = append(matches, event)
		}
	}
	switch len(matches) {
	case 0:
		return nil, ErrHistoryNotFound
	case 1:
		return matches[0], nil
	default:
		return nil, ErrAmbiguousPrefix
	}
}

func sortHistoryEvents(events []*history.Event) {
	slices.SortFunc(events, func(left, right *history.Event) int {
		if !left.OccurredAt.Equal(right.OccurredAt) {
			if left.OccurredAt.After(right.OccurredAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(left.ID.String(), right.ID.String())
	})
}
