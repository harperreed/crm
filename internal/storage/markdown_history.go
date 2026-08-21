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
	"strconv"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/history"
	"github.com/harperreed/crm/internal/models"
	"gopkg.in/yaml.v3"
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
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.historyEventsDir(), entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("%w: inspect committed history event %s: %w", ErrHistoryCorrupt, path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%w: %s: path is not a regular committed history event", ErrHistoryCorrupt, path)
		}
		data, err := readFileWithoutFollowingSymlinks(path)
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
	return syncMarkdownHistoryDirectory(s.historyEventsDir())
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
		return syncMarkdownHistoryDirectory(filepath.Dir(path))
	}
	return fmt.Errorf("%w: event ID %s already has different committed content", ErrHistoryConflict, expected.ID)
}

func syncMarkdownHistoryDirectory(path string) (returnErr error) {
	directory, err := os.Open(path) //nolint:gosec // path is the store's trusted history events directory.
	if err != nil {
		return fmt.Errorf("open history events directory for sync: %w", err)
	}
	defer func() {
		if err := directory.Close(); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("close history events directory after sync: %w", err))
		}
	}()

	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync history events directory: %w", err)
	}
	return nil
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

func (s *MarkdownStore) commitHistoryEvent(event *history.Event) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate history event: %w", err)
	}
	if err := s.writePendingHistoryEvent(event); err != nil {
		return err
	}
	return s.recoverPendingEvent(event)
}

func (s *MarkdownStore) recoverPendingHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.historyPendingDir())
	if err != nil {
		return fmt.Errorf("read pending history directory: %w", err)
	}
	slices.SortFunc(entries, func(left, right os.DirEntry) int {
		return strings.Compare(left.Name(), right.Name())
	})
	for _, entry := range entries {
		path := filepath.Join(s.historyPendingDir(), entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("%w: inspect pending history event %s: %w", ErrHistoryCorrupt, path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
			return fmt.Errorf("%w: pending history path %s is not a regular file", ErrHistoryCorrupt, path)
		}
		if strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			return fmt.Errorf("%w: unexpected file in pending history directory: %s", ErrHistoryCorrupt, path)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%w: pending history path %s is not a regular file", ErrHistoryCorrupt, path)
		}
		data, err := readFileWithoutFollowingSymlinks(path)
		if err != nil {
			return fmt.Errorf("%w: read pending history event %s: %w", ErrHistoryCorrupt, path, err)
		}
		event, err := decodeCommittedHistoryEvent(data)
		if err != nil {
			return fmt.Errorf("%w: %s: %w", ErrHistoryCorrupt, path, err)
		}
		if entry.Name() != event.ID.String()+".json" {
			return fmt.Errorf("%w: %s: filename does not match event ID %s", ErrHistoryCorrupt, path, event.ID)
		}
		if err := s.recoverPendingEvent(event); err != nil {
			return err
		}
	}
	return nil
}

func (s *MarkdownStore) recoverPendingEvent(event *history.Event) error {
	pendingPath := filepath.Join(s.historyPendingDir(), event.ID.String()+".json")
	committed, err := s.committedHistoryEventExists(event)
	if err != nil {
		return err
	}
	current, err := s.currentHistorySnapshot(event)
	if err != nil {
		return err
	}
	equalsBefore, err := history.EqualSnapshots(event.EntityType, current, event.Before)
	if err != nil {
		return fmt.Errorf("compare current %s snapshot with event before: %w", event.EntityType, err)
	}
	equalsAfter, err := history.EqualSnapshots(event.EntityType, current, event.After)
	if err != nil {
		return fmt.Errorf("compare current %s snapshot with event after: %w", event.EntityType, err)
	}
	if committed {
		if !equalsAfter {
			return fmt.Errorf("%w: committed event %s has current state that does not match its after snapshot; pending file %s preserved", ErrHistoryConflict, event.ID, pendingPath)
		}
		return s.finalizePendingHistory(event)
	}
	if equalsBefore {
		if err := s.applyHistoryEvent(event); err != nil {
			return err
		}
	} else if !equalsAfter {
		return fmt.Errorf("%w: current %s %s matches neither pending event snapshot at %s", ErrHistoryConflict, event.EntityType, event.EntityID, pendingPath)
	}
	return s.finalizePendingHistory(event)
}

func (s *MarkdownStore) writePendingHistoryEvent(event *history.Event) error {
	data, err := marshalCommittedHistoryEvent(event)
	if err != nil {
		return err
	}
	path := filepath.Join(s.historyPendingDir(), event.ID.String()+".json")
	if err := publishMarkdownHistoryFile(s.historyPendingDir(), path, event, data); err != nil {
		return fmt.Errorf("publish pending history event: %w", err)
	}
	return nil
}

func publishMarkdownHistoryFile(dir, path string, event *history.Event, data []byte) (returnErr error) {
	temporary, err := os.CreateTemp(dir, "."+event.ID.String()+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary history event: %w", err)
	}
	temporaryPath := temporary.Name()
	temporaryOpen := true
	defer func() {
		if temporaryOpen {
			returnErr = errors.Join(returnErr, temporary.Close())
		}
		if err := os.Remove(temporaryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			returnErr = errors.Join(returnErr, fmt.Errorf("remove temporary history event: %w", err))
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("set temporary history event mode: %w", err)
	}
	if written, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write temporary history event: %w", err)
	} else if written != len(data) {
		return fmt.Errorf("write temporary history event: %w", io.ErrShortWrite)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary history event: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary history event: %w", err)
	}
	temporaryOpen = false
	if err := os.Link(temporaryPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return comparePendingHistoryEvent(path, event, data)
		}
		return fmt.Errorf("publish history event %q: %w", path, err)
	}
	return syncMarkdownHistoryDirectory(dir)
}

func comparePendingHistoryEvent(path string, expected *history.Event, expectedData []byte) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect pending history event %q: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%w: pending history path %q is not a regular file", ErrHistoryCorrupt, path)
	}
	existingData, err := readFileWithoutFollowingSymlinks(path)
	if err != nil {
		return fmt.Errorf("read pending history event %q: %w", path, err)
	}
	existing, err := decodeCommittedHistoryEvent(existingData)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrHistoryCorrupt, path, err)
	}
	if filepath.Base(path) != existing.ID.String()+".json" {
		return fmt.Errorf("%w: %s: filename does not match event ID %s", ErrHistoryCorrupt, path, existing.ID)
	}
	canonical, err := marshalCommittedHistoryEvent(existing)
	if err != nil {
		return fmt.Errorf("%w: canonicalize %s: %w", ErrHistoryCorrupt, path, err)
	}
	if bytes.Equal(canonical, expectedData) && existing.ID == expected.ID {
		return syncMarkdownHistoryDirectory(filepath.Dir(path))
	}
	return fmt.Errorf("%w: event ID %s already has different pending content", ErrHistoryConflict, expected.ID)
}

func (s *MarkdownStore) committedHistoryEventExists(event *history.Event) (bool, error) {
	path := filepath.Join(s.historyEventsDir(), event.ID.String()+".json")
	_, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect committed history event %q: %w", path, err)
	}
	data, err := marshalCommittedHistoryEvent(event)
	if err != nil {
		return false, err
	}
	if err := compareCommittedHistoryEvent(path, event, data); err != nil {
		return false, err
	}
	return true, nil
}

func (s *MarkdownStore) finalizePendingHistory(event *history.Event) error {
	if err := s.writeCommittedHistoryEvent(event); err != nil {
		return err
	}
	path := filepath.Join(s.historyPendingDir(), event.ID.String()+".json")
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove pending history event %q: %w", path, err)
	}
	return syncMarkdownHistoryDirectory(s.historyPendingDir())
}

func (s *MarkdownStore) currentHistorySnapshot(event *history.Event) (json.RawMessage, error) {
	switch event.EntityType {
	case history.EntityContact:
		_, contact, err := s.findContactFileStrict(event.EntityID)
		if err != nil {
			return nil, err
		}
		if contact == nil {
			return nil, nil
		}
		return markdownContactSnapshot(contact)
	case history.EntityCompany:
		_, company, err := s.findCompanyFileStrict(event.EntityID)
		if err != nil {
			return nil, err
		}
		if company == nil {
			return nil, nil
		}
		return markdownCompanySnapshot(company)
	case history.EntityRelationship:
		relationship, err := s.findRelationshipStrict(event.EntityID)
		if errors.Is(err, ErrRelationshipNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if relationship == nil {
			return nil, nil
		}
		return markdownRelationshipSnapshot(relationship)
	default:
		return nil, fmt.Errorf("unsupported history entity type %q", event.EntityType)
	}
}

func markdownContactSnapshot(contact *models.Contact) (json.RawMessage, error) {
	normalized := *contact
	normalized.CreatedAt = contact.CreatedAt.UTC()
	normalized.UpdatedAt = contact.UpdatedAt.UTC()
	return history.SnapshotContact(&normalized)
}

func canonicalMarkdownContact(contact *models.Contact) *models.Contact {
	candidate := *contact
	canonicalizeMarkdownContactCollections(&candidate)
	return &candidate
}

func canonicalizeMarkdownContactCollections(contact *models.Contact) {
	if contact.Fields == nil {
		contact.Fields = make(map[string]any)
	}
	if contact.Tags == nil {
		contact.Tags = []string{}
	}
}

func markdownCompanySnapshot(company *models.Company) (json.RawMessage, error) {
	normalized := *company
	normalized.CreatedAt = company.CreatedAt.UTC()
	normalized.UpdatedAt = company.UpdatedAt.UTC()
	return history.SnapshotCompany(&normalized)
}

func canonicalMarkdownCompany(company *models.Company) *models.Company {
	candidate := *company
	canonicalizeMarkdownCompanyCollections(&candidate)
	return &candidate
}

func canonicalizeMarkdownCompanyCollections(company *models.Company) {
	if company.Fields == nil {
		company.Fields = make(map[string]any)
	}
	if company.Tags == nil {
		company.Tags = []string{}
	}
}

func markdownRelationshipSnapshot(relationship *models.Relationship) (json.RawMessage, error) {
	normalized := *relationship
	normalized.CreatedAt = relationship.CreatedAt.UTC()
	return history.SnapshotRelationship(&normalized)
}

func (s *MarkdownStore) applyHistoryEvent(event *history.Event) error {
	switch event.EntityType {
	case history.EntityContact:
		return s.applyContactHistoryEvent(event)
	case history.EntityCompany:
		return s.applyCompanyHistoryEvent(event)
	case history.EntityRelationship:
		return s.applyRelationshipHistoryEvent(event)
	default:
		return fmt.Errorf("unsupported history entity type %q", event.EntityType)
	}
}

func (s *MarkdownStore) applyContactHistoryEvent(event *history.Event) error {
	path, current, err := s.findContactFileStrict(event.EntityID)
	if err != nil {
		return err
	}
	if event.Action == history.ActionDelete {
		if current == nil {
			return ErrContactNotFound
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove contact %s: %w", event.EntityID, err)
		}
		return syncMarkdownHistoryDirectory(s.contactsDir())
	}
	contact, err := history.ContactFromSnapshot(event.After)
	if err != nil {
		return fmt.Errorf("decode contact history snapshot: %w", err)
	}
	contact.Fields = historyFieldsForYAML(contact.Fields)
	if event.Action == history.ActionCreate {
		if current != nil {
			return fmt.Errorf("%w: contact %s already exists", ErrHistoryConflict, event.EntityID)
		}
		filename := slugForName(contact.Name, contact.ID.String(), s.contactsDir())
		if err := s.writeContact(contact, filename); err != nil {
			return fmt.Errorf("write contact %s: %w", event.EntityID, err)
		}
		return syncMarkdownHistoryDirectory(s.contactsDir())
	}
	if current == nil {
		return ErrContactNotFound
	}
	oldName := current.Name
	if err := s.writeContact(contact, filepath.Base(path)); err != nil {
		return fmt.Errorf("write updated contact %s: %w", event.EntityID, err)
	}
	if err := syncMarkdownHistoryDirectory(s.contactsDir()); err != nil {
		return err
	}
	if oldName != contact.Name {
		newFilename := slugForName(contact.Name, contact.ID.String(), s.contactsDir())
		newPath := filepath.Join(s.contactsDir(), newFilename)
		if newPath != path {
			if err := os.Rename(path, newPath); err != nil {
				return fmt.Errorf("rename updated contact %s: %w", event.EntityID, err)
			}
		}
	}
	return syncMarkdownHistoryDirectory(s.contactsDir())
}

func (s *MarkdownStore) applyCompanyHistoryEvent(event *history.Event) error {
	path, current, err := s.findCompanyFileStrict(event.EntityID)
	if err != nil {
		return err
	}
	if event.Action == history.ActionDelete {
		if current == nil {
			return ErrCompanyNotFound
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove company %s: %w", event.EntityID, err)
		}
		return syncMarkdownHistoryDirectory(s.companiesDir())
	}
	company, err := history.CompanyFromSnapshot(event.After)
	if err != nil {
		return fmt.Errorf("decode company history snapshot: %w", err)
	}
	company.Fields = historyFieldsForYAML(company.Fields)
	if event.Action == history.ActionCreate {
		if current != nil {
			return fmt.Errorf("%w: company %s already exists", ErrHistoryConflict, event.EntityID)
		}
		filename := slugForName(company.Name, company.ID.String(), s.companiesDir())
		if err := s.writeCompany(company, filename); err != nil {
			return fmt.Errorf("write company %s: %w", event.EntityID, err)
		}
		return syncMarkdownHistoryDirectory(s.companiesDir())
	}
	if current == nil {
		return ErrCompanyNotFound
	}
	oldName := current.Name
	if err := s.writeCompany(company, filepath.Base(path)); err != nil {
		return fmt.Errorf("write updated company %s: %w", event.EntityID, err)
	}
	if err := syncMarkdownHistoryDirectory(s.companiesDir()); err != nil {
		return err
	}
	if oldName != company.Name {
		newFilename := slugForName(company.Name, company.ID.String(), s.companiesDir())
		newPath := filepath.Join(s.companiesDir(), newFilename)
		if newPath != path {
			if err := os.Rename(path, newPath); err != nil {
				return fmt.Errorf("rename updated company %s: %w", event.EntityID, err)
			}
		}
	}
	return syncMarkdownHistoryDirectory(s.companiesDir())
}

func (s *MarkdownStore) applyRelationshipHistoryEvent(event *history.Event) error {
	entries, relationships, err := s.readRelationshipsStrict()
	if err != nil {
		return err
	}
	index := -1
	for i, relationship := range relationships {
		if relationship.ID == event.EntityID {
			index = i
			break
		}
	}
	if event.Action == history.ActionDelete {
		if index < 0 {
			return ErrRelationshipNotFound
		}
		entries = append(entries[:index], entries[index+1:]...)
		return s.writeRelationships(entries)
	}
	if event.Action != history.ActionCreate {
		return fmt.Errorf("unsupported relationship history action %q", event.Action)
	}
	if index >= 0 {
		return fmt.Errorf("%w: relationship %s already exists", ErrHistoryConflict, event.EntityID)
	}
	relationship, err := history.RelationshipFromSnapshot(event.After)
	if err != nil {
		return fmt.Errorf("decode relationship history snapshot: %w", err)
	}
	entries = append(entries, relationshipToEntry(relationship))
	return s.writeRelationships(entries)
}

func (s *MarkdownStore) findContactFileStrict(id uuid.UUID) (string, *models.Contact, error) {
	entries, err := os.ReadDir(s.contactsDir())
	if err != nil {
		return "", nil, fmt.Errorf("read contacts directory: %w", err)
	}
	seen := make(map[uuid.UUID]string)
	var foundPath string
	var found *models.Contact
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(s.contactsDir(), entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return "", nil, fmt.Errorf("%w: inspect contact path %s: %w", ErrHistoryCorrupt, path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return "", nil, fmt.Errorf("%w: contact path %s is not a regular file", ErrHistoryCorrupt, path)
		}
		contact, err := readContactFileStrict(path)
		if err != nil {
			return "", nil, fmt.Errorf("%w: %s: %w", ErrHistoryCorrupt, path, err)
		}
		if previous, ok := seen[contact.ID]; ok {
			return "", nil, fmt.Errorf("%w: duplicate contact ID %s in %s and %s", ErrHistoryConflict, contact.ID, previous, path)
		}
		seen[contact.ID] = path
		if contact.ID == id {
			foundPath, found = path, contact
		}
	}
	return foundPath, found, nil
}

func (s *MarkdownStore) findCompanyFileStrict(id uuid.UUID) (string, *models.Company, error) {
	entries, err := os.ReadDir(s.companiesDir())
	if err != nil {
		return "", nil, fmt.Errorf("read companies directory: %w", err)
	}
	seen := make(map[uuid.UUID]string)
	var foundPath string
	var found *models.Company
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(s.companiesDir(), entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return "", nil, fmt.Errorf("%w: inspect company path %s: %w", ErrHistoryCorrupt, path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return "", nil, fmt.Errorf("%w: company path %s is not a regular file", ErrHistoryCorrupt, path)
		}
		company, err := readCompanyFileStrict(path)
		if err != nil {
			return "", nil, fmt.Errorf("%w: %s: %w", ErrHistoryCorrupt, path, err)
		}
		if previous, ok := seen[company.ID]; ok {
			return "", nil, fmt.Errorf("%w: duplicate company ID %s in %s and %s", ErrHistoryConflict, company.ID, previous, path)
		}
		seen[company.ID] = path
		if company.ID == id {
			foundPath, found = path, company
		}
	}
	return foundPath, found, nil
}

func readContactFileStrict(path string) (*models.Contact, error) {
	data, err := readFileWithoutFollowingSymlinks(path)
	if err != nil {
		return nil, err
	}
	yamlText, err := parseStrictMarkdownFrontmatter(data)
	if err != nil {
		return nil, err
	}
	var frontmatter contactFrontmatter
	if err := strictYAMLUnmarshal(yamlText, &frontmatter); err != nil {
		return nil, err
	}
	return frontmatterToContact(frontmatter)
}

func readCompanyFileStrict(path string) (*models.Company, error) {
	data, err := readFileWithoutFollowingSymlinks(path)
	if err != nil {
		return nil, err
	}
	yamlText, err := parseStrictMarkdownFrontmatter(data)
	if err != nil {
		return nil, err
	}
	var frontmatter companyFrontmatter
	if err := strictYAMLUnmarshal(yamlText, &frontmatter); err != nil {
		return nil, err
	}
	return frontmatterToCompany(frontmatter)
}

func parseStrictMarkdownFrontmatter(data []byte) ([]byte, error) {
	normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return nil, errors.New("markdown file must start with an exact --- delimiter line")
	}
	closing := -1
	for index := 1; index < len(lines); index++ {
		if lines[index] == "---" {
			closing = index
			break
		}
		if strings.HasPrefix(lines[index], "---") {
			return nil, fmt.Errorf("malformed frontmatter delimiter on line %d", index+1)
		}
	}
	if closing < 0 {
		return nil, errors.New("markdown frontmatter has no exact closing --- delimiter line")
	}
	yamlText := strings.Join(lines[1:closing], "\n")
	if strings.TrimSpace(yamlText) == "" {
		return nil, errors.New("markdown frontmatter is empty")
	}
	return []byte(yamlText), nil
}

func (s *MarkdownStore) readRelationshipsStrict() ([]relationshipEntry, []*models.Relationship, error) {
	path := s.relationshipsFile()
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("inspect relationships file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%w: relationships path %s is not a regular file", ErrHistoryCorrupt, path)
	}
	data, err := readFileWithoutFollowingSymlinks(path)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: read relationships %s: %w", ErrHistoryCorrupt, path, err)
	}
	var entries []relationshipEntry
	if err := strictYAMLUnmarshal(data, &entries); err != nil {
		return nil, nil, fmt.Errorf("%w: %s: %w", ErrHistoryCorrupt, path, err)
	}
	seen := make(map[uuid.UUID]int)
	relationships := make([]*models.Relationship, 0, len(entries))
	for index, entry := range entries {
		relationship, err := entryToRelationship(entry)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %s entry %d: %w", ErrHistoryCorrupt, path, index, err)
		}
		if previous, ok := seen[relationship.ID]; ok {
			return nil, nil, fmt.Errorf("%w: duplicate relationship ID %s at entries %d and %d", ErrHistoryConflict, relationship.ID, previous, index)
		}
		seen[relationship.ID] = index
		relationships = append(relationships, relationship)
	}
	return entries, relationships, nil
}

func (s *MarkdownStore) findRelationshipStrict(id uuid.UUID) (*models.Relationship, error) {
	_, relationships, err := s.readRelationshipsStrict()
	if err != nil {
		return nil, err
	}
	for _, relationship := range relationships {
		if relationship.ID == id {
			return relationship, nil
		}
	}
	return nil, ErrRelationshipNotFound
}

func strictYAMLUnmarshal(data []byte, destination any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("YAML contains multiple documents")
		}
		return err
	}
	return nil
}

func historyFieldsForYAML(fields map[string]any) map[string]any {
	converted, _ := historyValueForYAML(fields).(map[string]any)
	return converted
}

func historyValueForYAML(value any) any {
	switch value := value.(type) {
	case json.Number:
		if integer, err := strconv.ParseInt(value.String(), 10, 64); err == nil {
			return integer
		}
		if integer, err := strconv.ParseUint(value.String(), 10, 64); err == nil {
			return integer
		}
		if decimal, err := strconv.ParseFloat(value.String(), 64); err == nil {
			return decimal
		}
		return value.String()
	case map[string]any:
		converted := make(map[string]any, len(value))
		for key, item := range value {
			converted[key] = historyValueForYAML(item)
		}
		return converted
	case []any:
		converted := make([]any, len(value))
		for index, item := range value {
			converted[index] = historyValueForYAML(item)
		}
		return converted
	default:
		return value
	}
}
