# Follow-Up System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add comprehensive follow-up tracking to help users maintain their personal network by tracking interaction cadence, computing priority scores, and surfacing reminders across all interfaces.

**Architecture:** Extend existing data model with `contact_cadence` and `interaction_log` tables. Add priority scoring algorithm that runs on-write. Integrate follow-up views into TUI (new tab), CLI (new subcommand), Web (new page), and MCP (new tools). Keep digest functionality CLI-based for cron integration.

**Tech Stack:** Go 1.24, SQLite, bubbletea (TUI), Go templates + HTMX (Web), MCP SDK

---

## Task 1: Data Models

**Files:**
- Modify: `models/types.go:72` (after Relationship constants)
- Create: `models/types_test.go`

**Step 1: Write test for ContactCadence model**

```go
// models/types_test.go
package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestContactCadenceDefaults(t *testing.T) {
	cadence := &ContactCadence{
		ContactID:           uuid.New(),
		CadenceDays:         30,
		RelationshipStrength: StrengthMedium,
	}

	if cadence.CadenceDays != 30 {
		t.Errorf("expected default cadence 30, got %d", cadence.CadenceDays)
	}
	if cadence.RelationshipStrength != StrengthMedium {
		t.Errorf("expected medium strength, got %s", cadence.RelationshipStrength)
	}
}

func TestInteractionLogCreation(t *testing.T) {
	log := &InteractionLog{
		ID:              uuid.New(),
		ContactID:       uuid.New(),
		InteractionType: InteractionMeeting,
		Timestamp:       time.Now(),
		Notes:           "Coffee chat",
	}

	if log.InteractionType != InteractionMeeting {
		t.Errorf("expected meeting type, got %s", log.InteractionType)
	}
}

func TestComputePriorityScore(t *testing.T) {
	lastContact := time.Now().AddDate(0, 0, -45) // 45 days ago
	cadence := &ContactCadence{
		ContactID:            uuid.New(),
		CadenceDays:          30,
		RelationshipStrength: StrengthStrong,
		LastInteractionDate:  &lastContact,
	}

	score := cadence.ComputePriorityScore()

	// 45 - 30 = 15 days overdue
	// 15 * 2 = 30 base score
	// 30 * 2.0 (strong multiplier) = 60
	expected := 60.0
	if score != expected {
		t.Errorf("expected priority score %.1f, got %.1f", expected, score)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/harper/workspace/personal/pagen
go test ./models -v -run TestContactCadence
```

Expected: FAIL - undefined types

**Step 3: Add ContactCadence and InteractionLog models**

In `models/types.go` after line 72 (after Relationship constants):

```go
// RelationshipStrength constants
const (
	StrengthWeak   = "weak"
	StrengthMedium = "medium"
	StrengthStrong = "strong"
)

// InteractionType constants
const (
	InteractionMeeting = "meeting"
	InteractionCall    = "call"
	InteractionEmail   = "email"
	InteractionMessage = "message"
	InteractionEvent   = "event"
)

// Sentiment constants
const (
	SentimentPositive = "positive"
	SentimentNeutral  = "neutral"
	SentimentNegative = "negative"
)

type ContactCadence struct {
	ContactID            uuid.UUID  `json:"contact_id"`
	CadenceDays          int        `json:"cadence_days"`
	RelationshipStrength string     `json:"relationship_strength"`
	PriorityScore        float64    `json:"priority_score"`
	LastInteractionDate  *time.Time `json:"last_interaction_date,omitempty"`
	NextFollowupDate     *time.Time `json:"next_followup_date,omitempty"`
}

// ComputePriorityScore calculates the priority score for a contact
// Based on days overdue and relationship strength
func (c *ContactCadence) ComputePriorityScore() float64 {
	if c.LastInteractionDate == nil {
		return 0.0
	}

	daysSinceContact := int(time.Since(*c.LastInteractionDate).Hours() / 24)
	daysOverdue := daysSinceContact - c.CadenceDays

	if daysOverdue <= 0 {
		return 0.0
	}

	baseScore := float64(daysOverdue * 2)

	// Apply relationship multiplier
	multiplier := 1.0
	switch c.RelationshipStrength {
	case StrengthStrong:
		multiplier = 2.0
	case StrengthMedium:
		multiplier = 1.5
	case StrengthWeak:
		multiplier = 1.0
	}

	return baseScore * multiplier
}

// UpdateNextFollowup sets the next followup date based on last interaction and cadence
func (c *ContactCadence) UpdateNextFollowup() {
	if c.LastInteractionDate != nil {
		next := c.LastInteractionDate.AddDate(0, 0, c.CadenceDays)
		c.NextFollowupDate = &next
	}
}

type InteractionLog struct {
	ID              uuid.UUID  `json:"id"`
	ContactID       uuid.UUID  `json:"contact_id"`
	InteractionType string     `json:"interaction_type"`
	Timestamp       time.Time  `json:"timestamp"`
	Notes           string     `json:"notes,omitempty"`
	Sentiment       *string    `json:"sentiment,omitempty"`
}

// FollowupContact combines Contact with cadence info for follow-up views
type FollowupContact struct {
	Contact
	CadenceDays          int        `json:"cadence_days"`
	RelationshipStrength string     `json:"relationship_strength"`
	PriorityScore        float64    `json:"priority_score"`
	DaysSinceContact     int        `json:"days_since_contact"`
	NextFollowupDate     *time.Time `json:"next_followup_date,omitempty"`
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./models -v -run TestContactCadence
go test ./models -v -run TestInteractionLog
go test ./models -v -run TestComputePriorityScore
```

Expected: PASS

**Step 5: Commit**

```bash
git add models/types.go models/types_test.go
git commit -m "feat: add ContactCadence and InteractionLog models

Add follow-up tracking data models with priority scoring algorithm.
Includes relationship strength and interaction type constants."
```

---

## Task 2: Database Schema Migration

**Files:**
- Modify: `db/schema.go:81` (after relationships table)
- Create: `db/schema_test.go` (enhance existing)

**Step 1: Write test for new tables**

In `db/schema_test.go`:

```go
func TestSchemaIncludesFollowupTables(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Check contact_cadence table exists
	row := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='contact_cadence'`)
	var tableName string
	err := row.Scan(&tableName)
	if err != nil {
		t.Fatalf("contact_cadence table not found: %v", err)
	}

	// Check interaction_log table exists
	row = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='interaction_log'`)
	err = row.Scan(&tableName)
	if err != nil {
		t.Fatalf("interaction_log table not found: %v", err)
	}

	// Check indexes exist
	row = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name='idx_contact_cadence_priority'`)
	err = row.Scan(&tableName)
	if err != nil {
		t.Fatalf("priority index not found: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./db -v -run TestSchemaIncludesFollowupTables
```

Expected: FAIL - tables not found

**Step 3: Add schema for new tables**

In `db/schema.go` after line 80 (after relationships table):

```go
CREATE TABLE IF NOT EXISTS contact_cadence (
	contact_id TEXT PRIMARY KEY,
	cadence_days INTEGER NOT NULL DEFAULT 30,
	relationship_strength TEXT NOT NULL DEFAULT 'medium' CHECK(relationship_strength IN ('weak', 'medium', 'strong')),
	priority_score REAL NOT NULL DEFAULT 0,
	last_interaction_date DATETIME,
	next_followup_date DATETIME,
	FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_contact_cadence_priority ON contact_cadence(priority_score DESC);
CREATE INDEX IF NOT EXISTS idx_contact_cadence_next_followup ON contact_cadence(next_followup_date);

CREATE TABLE IF NOT EXISTS interaction_log (
	id TEXT PRIMARY KEY,
	contact_id TEXT NOT NULL,
	interaction_type TEXT NOT NULL CHECK(interaction_type IN ('meeting', 'call', 'email', 'message', 'event')),
	timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	notes TEXT,
	sentiment TEXT CHECK(sentiment IN ('positive', 'neutral', 'negative')),
	FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_interaction_log_contact ON interaction_log(contact_id);
CREATE INDEX IF NOT EXISTS idx_interaction_log_timestamp ON interaction_log(timestamp DESC);
```

**Step 4: Run test to verify it passes**

```bash
go test ./db -v -run TestSchemaIncludesFollowupTables
```

Expected: PASS

**Step 5: Commit**

```bash
git add db/schema.go db/schema_test.go
git commit -m "feat: add follow-up tracking schema

Add contact_cadence and interaction_log tables with indexes
for efficient priority-based queries."
```

---

## Task 3: Database Operations - Contact Cadence

**Files:**
- Create: `db/followups.go`
- Create: `db/followups_test.go`

**Step 1: Write test for creating contact cadence**

```go
// db/followups_test.go
package db

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

func TestCreateContactCadence(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create contact first
	contact := &models.Contact{Name: "Alice"}
	if err := CreateContact(db, contact); err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	cadence := &models.ContactCadence{
		ContactID:            contact.ID,
		CadenceDays:          30,
		RelationshipStrength: models.StrengthMedium,
	}

	err := CreateContactCadence(db, cadence)
	if err != nil {
		t.Fatalf("failed to create cadence: %v", err)
	}

	// Verify it was created
	retrieved, err := GetContactCadence(db, contact.ID)
	if err != nil {
		t.Fatalf("failed to get cadence: %v", err)
	}
	if retrieved.CadenceDays != 30 {
		t.Errorf("expected 30 days, got %d", retrieved.CadenceDays)
	}
}

func TestGetFollowupList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create contacts with different cadences
	contact1 := &models.Contact{Name: "Alice"}
	CreateContact(db, contact1)
	lastContact := time.Now().AddDate(0, 0, -45)
	cadence1 := &models.ContactCadence{
		ContactID:            contact1.ID,
		CadenceDays:          30,
		RelationshipStrength: models.StrengthStrong,
		LastInteractionDate:  &lastContact,
		PriorityScore:        60.0,
	}
	CreateContactCadence(db, cadence1)

	contact2 := &models.Contact{Name: "Bob"}
	CreateContact(db, contact2)

	// Get followup list
	followups, err := GetFollowupList(db, 10)
	if err != nil {
		t.Fatalf("failed to get followup list: %v", err)
	}

	if len(followups) == 0 {
		t.Error("expected at least one followup")
	}

	// Should be sorted by priority score descending
	if len(followups) > 1 && followups[0].PriorityScore < followups[1].PriorityScore {
		t.Error("followups not sorted by priority")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./db -v -run TestCreateContactCadence
go test ./db -v -run TestGetFollowupList
```

Expected: FAIL - undefined functions

**Step 3: Implement database operations**

```go
// db/followups.go
// ABOUTME: Database operations for follow-up tracking
// ABOUTME: Handles contact cadence, interaction logging, and follow-up queries
package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

// CreateContactCadence creates or updates a contact's follow-up cadence
func CreateContactCadence(db *sql.DB, cadence *models.ContactCadence) error {
	query := `
		INSERT INTO contact_cadence (
			contact_id, cadence_days, relationship_strength,
			priority_score, last_interaction_date, next_followup_date
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(contact_id) DO UPDATE SET
			cadence_days = excluded.cadence_days,
			relationship_strength = excluded.relationship_strength,
			priority_score = excluded.priority_score,
			last_interaction_date = excluded.last_interaction_date,
			next_followup_date = excluded.next_followup_date
	`

	_, err := db.Exec(query,
		cadence.ContactID.String(),
		cadence.CadenceDays,
		cadence.RelationshipStrength,
		cadence.PriorityScore,
		cadence.LastInteractionDate,
		cadence.NextFollowupDate,
	)
	return err
}

// GetContactCadence retrieves cadence info for a contact
func GetContactCadence(db *sql.DB, contactID uuid.UUID) (*models.ContactCadence, error) {
	query := `
		SELECT contact_id, cadence_days, relationship_strength,
		       priority_score, last_interaction_date, next_followup_date
		FROM contact_cadence
		WHERE contact_id = ?
	`

	cadence := &models.ContactCadence{}
	err := db.QueryRow(query, contactID.String()).Scan(
		&cadence.ContactID,
		&cadence.CadenceDays,
		&cadence.RelationshipStrength,
		&cadence.PriorityScore,
		&cadence.LastInteractionDate,
		&cadence.NextFollowupDate,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return cadence, err
}

// GetFollowupList returns contacts needing follow-up, sorted by priority
func GetFollowupList(db *sql.DB, limit int) ([]models.FollowupContact, error) {
	query := `
		SELECT
			c.id, c.name, c.email, c.phone, c.company_id, c.notes,
			c.last_contacted_at, c.created_at, c.updated_at,
			cc.cadence_days, cc.relationship_strength, cc.priority_score,
			cc.next_followup_date,
			CAST((julianday('now') - julianday(cc.last_interaction_date)) AS INTEGER) as days_since
		FROM contacts c
		INNER JOIN contact_cadence cc ON c.id = cc.contact_id
		WHERE cc.priority_score > 0
		ORDER BY cc.priority_score DESC
		LIMIT ?
	`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var followups []models.FollowupContact
	for rows.Next() {
		var f models.FollowupContact
		err := rows.Scan(
			&f.ID, &f.Name, &f.Email, &f.Phone, &f.CompanyID, &f.Notes,
			&f.LastContactedAt, &f.CreatedAt, &f.UpdatedAt,
			&f.CadenceDays, &f.RelationshipStrength, &f.PriorityScore,
			&f.NextFollowupDate, &f.DaysSinceContact,
		)
		if err != nil {
			return nil, err
		}
		followups = append(followups, f)
	}

	return followups, rows.Err()
}

// UpdateCadenceAfterInteraction updates cadence when interaction is logged
func UpdateCadenceAfterInteraction(db *sql.DB, contactID uuid.UUID, timestamp time.Time) error {
	// Get or create cadence
	cadence, err := GetContactCadence(db, contactID)
	if err != nil {
		return err
	}

	if cadence == nil {
		// Create default cadence
		cadence = &models.ContactCadence{
			ContactID:            contactID,
			CadenceDays:          30,
			RelationshipStrength: models.StrengthMedium,
		}
	}

	// Update timestamps
	cadence.LastInteractionDate = &timestamp
	cadence.UpdateNextFollowup()
	cadence.PriorityScore = cadence.ComputePriorityScore()

	return CreateContactCadence(db, cadence)
}

// SetContactCadence sets or updates a contact's cadence settings
func SetContactCadence(db *sql.DB, contactID uuid.UUID, days int, strength string) error {
	cadence, err := GetContactCadence(db, contactID)
	if err != nil {
		return err
	}

	if cadence == nil {
		cadence = &models.ContactCadence{
			ContactID: contactID,
		}
	}

	cadence.CadenceDays = days
	cadence.RelationshipStrength = strength
	cadence.PriorityScore = cadence.ComputePriorityScore()
	cadence.UpdateNextFollowup()

	return CreateContactCadence(db, cadence)
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./db -v -run TestCreateContactCadence
go test ./db -v -run TestGetFollowupList
```

Expected: PASS

**Step 5: Commit**

```bash
git add db/followups.go db/followups_test.go
git commit -m "feat: add follow-up database operations

Implement contact cadence CRUD and follow-up list queries
with priority-based sorting."
```

---

## Task 4: Database Operations - Interaction Logging

**Files:**
- Modify: `db/followups.go` (add functions)
- Modify: `db/followups_test.go` (add tests)

**Step 1: Write test for logging interactions**

Add to `db/followups_test.go`:

```go
func TestLogInteraction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create contact
	contact := &models.Contact{Name: "Alice"}
	CreateContact(db, contact)

	// Create initial cadence
	lastContact := time.Now().AddDate(0, 0, -10)
	cadence := &models.ContactCadence{
		ContactID:            contact.ID,
		CadenceDays:          30,
		RelationshipStrength: models.StrengthMedium,
		LastInteractionDate:  &lastContact,
	}
	CreateContactCadence(db, cadence)

	// Log interaction
	interaction := &models.InteractionLog{
		ContactID:       contact.ID,
		InteractionType: models.InteractionMeeting,
		Timestamp:       time.Now(),
		Notes:           "Coffee chat",
	}

	err := LogInteraction(db, interaction)
	if err != nil {
		t.Fatalf("failed to log interaction: %v", err)
	}

	// Verify cadence was updated
	updated, err := GetContactCadence(db, contact.ID)
	if err != nil {
		t.Fatalf("failed to get updated cadence: %v", err)
	}

	if updated.LastInteractionDate.Before(lastContact) {
		t.Error("last interaction date was not updated")
	}

	// Priority score should be lower now
	if updated.PriorityScore >= cadence.PriorityScore {
		t.Error("priority score should decrease after interaction")
	}
}

func TestGetInteractionHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	contact := &models.Contact{Name: "Alice"}
	CreateContact(db, contact)

	// Log multiple interactions
	for i := 0; i < 3; i++ {
		interaction := &models.InteractionLog{
			ContactID:       contact.ID,
			InteractionType: models.InteractionEmail,
			Timestamp:       time.Now().AddDate(0, 0, -i),
		}
		LogInteraction(db, interaction)
	}

	history, err := GetInteractionHistory(db, contact.ID, 10)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("expected 3 interactions, got %d", len(history))
	}

	// Should be sorted newest first
	if len(history) > 1 && history[0].Timestamp.Before(history[1].Timestamp) {
		t.Error("interactions not sorted by timestamp descending")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./db -v -run TestLogInteraction
go test ./db -v -run TestGetInteractionHistory
```

Expected: FAIL - undefined functions

**Step 3: Add interaction logging functions**

Add to `db/followups.go`:

```go
// LogInteraction records a new interaction and updates contact cadence
func LogInteraction(db *sql.DB, interaction *models.InteractionLog) error {
	// Generate ID if not set
	if interaction.ID == uuid.Nil {
		interaction.ID = uuid.New()
	}

	// Insert interaction
	query := `
		INSERT INTO interaction_log (
			id, contact_id, interaction_type, timestamp, notes, sentiment
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query,
		interaction.ID.String(),
		interaction.ContactID.String(),
		interaction.InteractionType,
		interaction.Timestamp,
		interaction.Notes,
		interaction.Sentiment,
	)
	if err != nil {
		return err
	}

	// Update contact's last_contacted_at
	updateContact := `UPDATE contacts SET last_contacted_at = ? WHERE id = ?`
	_, err = db.Exec(updateContact, interaction.Timestamp, interaction.ContactID.String())
	if err != nil {
		return err
	}

	// Update cadence
	return UpdateCadenceAfterInteraction(db, interaction.ContactID, interaction.Timestamp)
}

// GetInteractionHistory retrieves interaction history for a contact
func GetInteractionHistory(db *sql.DB, contactID uuid.UUID, limit int) ([]models.InteractionLog, error) {
	query := `
		SELECT id, contact_id, interaction_type, timestamp, notes, sentiment
		FROM interaction_log
		WHERE contact_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := db.Query(query, contactID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interactions []models.InteractionLog
	for rows.Next() {
		var i models.InteractionLog
		var id, contactID string
		err := rows.Scan(&id, &contactID, &i.InteractionType, &i.Timestamp, &i.Notes, &i.Sentiment)
		if err != nil {
			return nil, err
		}
		i.ID, _ = uuid.Parse(id)
		i.ContactID, _ = uuid.Parse(contactID)
		interactions = append(interactions, i)
	}

	return interactions, rows.Err()
}

// GetRecentInteractions gets all recent interactions across all contacts
func GetRecentInteractions(db *sql.DB, days int, limit int) ([]models.InteractionLog, error) {
	query := `
		SELECT id, contact_id, interaction_type, timestamp, notes, sentiment
		FROM interaction_log
		WHERE timestamp >= datetime('now', '-' || ? || ' days')
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := db.Query(query, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interactions []models.InteractionLog
	for rows.Next() {
		var i models.InteractionLog
		var id, contactID string
		err := rows.Scan(&id, &contactID, &i.InteractionType, &i.Timestamp, &i.Notes, &i.Sentiment)
		if err != nil {
			return nil, err
		}
		i.ID, _ = uuid.Parse(id)
		i.ContactID, _ = uuid.Parse(contactID)
		interactions = append(interactions, i)
	}

	return interactions, rows.Err()
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./db -v -run TestLogInteraction
go test ./db -v -run TestGetInteractionHistory
```

Expected: PASS

**Step 5: Commit**

```bash
git add db/followups.go db/followups_test.go
git commit -m "feat: add interaction logging with automatic cadence updates

Log interactions and auto-update contact cadence and priority scores."
```

---

## Task 5: CLI Commands - Followup List

**Files:**
- Create: `cli/followups.go`
- Modify: `main.go` (register commands) - will do after all CLI commands

**Step 1: Write basic structure test**

Since CLI is harder to unit test, we'll do integration-style testing:

```go
// cli/followups_test.go
package cli

import (
	"bytes"
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/harperreed/pagen/db"
)

func setupTestCLI(t *testing.T) *sql.DB {
	tmpDB, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpDB.Close()
	t.Cleanup(func() { os.Remove(tmpDB.Name()) })

	database, err := db.OpenDatabase(tmpDB.Name())
	if err != nil {
		t.Fatal(err)
	}

	return database
}

func TestFollowupListCommand(t *testing.T) {
	database := setupTestCLI(t)
	defer database.Close()

	// Will test that command runs without error
	// Detailed output testing will be manual
	err := FollowupListCommand(database, []string{})
	if err != nil {
		t.Errorf("FollowupListCommand failed: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./cli -v -run TestFollowupListCommand
```

Expected: FAIL - undefined function

**Step 3: Implement followup list command**

```go
// cli/followups.go
// ABOUTME: Follow-up tracking CLI commands
// ABOUTME: Commands for listing follow-ups, logging interactions, setting cadence
package cli

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
)

// FollowupListCommand lists contacts needing follow-up
func FollowupListCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	overdueOnly := fs.Bool("overdue-only", false, "Show only overdue contacts")
	strength := fs.String("strength", "", "Filter by relationship strength (weak/medium/strong)")
	limit := fs.Int("limit", 10, "Maximum number of contacts to show")
	_ = fs.Parse(args)

	followups, err := db.GetFollowupList(database, *limit)
	if err != nil {
		return fmt.Errorf("failed to get followup list: %w", err)
	}

	// Apply filters
	var filtered []models.FollowupContact
	for _, f := range followups {
		if *overdueOnly && f.PriorityScore <= 0 {
			continue
		}
		if *strength != "" && f.RelationshipStrength != *strength {
			continue
		}
		filtered = append(filtered, f)
	}

	// Print results
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDAYS SINCE\tPRIORITY\tSTRENGTH\tEMAIL")
	fmt.Fprintln(w, "----\t----------\t--------\t--------\t-----")

	for _, f := range filtered {
		indicator := "🟢"
		if f.DaysSinceContact > f.CadenceDays+7 {
			indicator = "🔴"
		} else if f.DaysSinceContact >= f.CadenceDays-3 {
			indicator = "🟡"
		}

		fmt.Fprintf(w, "%s %s\t%d\t%.1f\t%s\t%s\n",
			indicator, f.Name, f.DaysSinceContact, f.PriorityScore,
			f.RelationshipStrength, f.Email)
	}

	w.Flush()
	return nil
}

// FollowupStatsCommand shows follow-up statistics
func FollowupStatsCommand(database *sql.DB, args []string) error {
	query := `
		SELECT
			relationship_strength,
			COUNT(*) as count,
			AVG(CAST((julianday('now') - julianday(last_interaction_date)) AS INTEGER)) as avg_days
		FROM contact_cadence
		WHERE last_interaction_date IS NOT NULL
		GROUP BY relationship_strength
	`

	rows, err := database.Query(query)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}
	defer rows.Close()

	fmt.Println("NETWORK HEALTH")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for rows.Next() {
		var strength string
		var count int
		var avgDays sql.NullFloat64

		err := rows.Scan(&strength, &count, &avgDays)
		if err != nil {
			return err
		}

		icon := "🟢"
		if strength == models.StrengthWeak {
			icon = "🔴"
		} else if strength == models.StrengthMedium {
			icon = "🟡"
		}

		fmt.Printf("  %s %s relationships: %d (avg contact: %.0f days)\n",
			icon, strength, count, avgDays.Float64)
	}

	return rows.Err()
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./cli -v -run TestFollowupListCommand
```

Expected: PASS

**Step 5: Manual test**

```bash
go run . followups list
```

**Step 6: Commit**

```bash
git add cli/followups.go cli/followups_test.go
git commit -m "feat: add followup list and stats CLI commands

Add commands to view follow-up list and network health statistics."
```

---

## Task 6: CLI Commands - Log Interaction

**Files:**
- Modify: `cli/followups.go`

**Step 1: Write test**

Add to `cli/followups_test.go`:

```go
func TestLogInteractionCommand(t *testing.T) {
	database := setupTestCLI(t)
	defer database.Close()

	// Create a contact first
	contact := &models.Contact{Name: "Alice", Email: "alice@example.com"}
	db.CreateContact(database, contact)

	args := []string{
		"--contact", contact.ID.String(),
		"--type", "meeting",
		"--notes", "Coffee chat",
	}

	err := LogInteractionCommand(database, args)
	if err != nil {
		t.Errorf("LogInteractionCommand failed: %v", err)
	}

	// Verify interaction was logged
	history, err := db.GetInteractionHistory(database, contact.ID, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 interaction, got %d", len(history))
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./cli -v -run TestLogInteractionCommand
```

Expected: FAIL

**Step 3: Implement log interaction command**

Add to `cli/followups.go`:

```go
// LogInteractionCommand logs an interaction with a contact
func LogInteractionCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("log", flag.ExitOnError)
	contactIDStr := fs.String("contact", "", "Contact ID or name (required)")
	interactionType := fs.String("type", "meeting", "Interaction type (meeting/call/email/message/event)")
	notes := fs.String("notes", "", "Notes about the interaction")
	sentiment := fs.String("sentiment", "", "Sentiment (positive/neutral/negative)")
	_ = fs.Parse(args)

	if *contactIDStr == "" {
		return fmt.Errorf("--contact is required")
	}

	// Try to parse as UUID, otherwise search by name
	var contactID uuid.UUID
	parsedID, err := uuid.Parse(*contactIDStr)
	if err == nil {
		contactID = parsedID
	} else {
		// Search by name
		contacts, err := db.FindContactsByQuery(database, *contactIDStr)
		if err != nil {
			return fmt.Errorf("failed to find contact: %w", err)
		}
		if len(contacts) == 0 {
			return fmt.Errorf("no contact found matching: %s", *contactIDStr)
		}
		if len(contacts) > 1 {
			return fmt.Errorf("multiple contacts found, please use ID")
		}
		contactID = contacts[0].ID
	}

	interaction := &models.InteractionLog{
		ContactID:       contactID,
		InteractionType: *interactionType,
		Timestamp:       time.Now(),
		Notes:           *notes,
	}

	if *sentiment != "" {
		interaction.Sentiment = sentiment
	}

	if err := db.LogInteraction(database, interaction); err != nil {
		return fmt.Errorf("failed to log interaction: %w", err)
	}

	fmt.Printf("✓ Logged %s interaction with contact\n", *interactionType)
	return nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./cli -v -run TestLogInteractionCommand
```

Expected: PASS

**Step 5: Commit**

```bash
git add cli/followups.go cli/followups_test.go
git commit -m "feat: add log interaction CLI command

Allow logging interactions with contacts from command line."
```

---

## Task 7: CLI Commands - Set Cadence and Digest

**Files:**
- Modify: `cli/followups.go`

**Step 1: Add implementations directly (small functions)**

Add to `cli/followups.go`:

```go
// SetCadenceCommand sets the follow-up cadence for a contact
func SetCadenceCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("set-cadence", flag.ExitOnError)
	contactIDStr := fs.String("contact", "", "Contact ID or name (required)")
	days := fs.Int("days", 30, "Cadence in days")
	strength := fs.String("strength", "medium", "Relationship strength (weak/medium/strong)")
	_ = fs.Parse(args)

	if *contactIDStr == "" {
		return fmt.Errorf("--contact is required")
	}

	// Resolve contact ID
	var contactID uuid.UUID
	parsedID, err := uuid.Parse(*contactIDStr)
	if err == nil {
		contactID = parsedID
	} else {
		contacts, err := db.FindContactsByQuery(database, *contactIDStr)
		if err != nil {
			return fmt.Errorf("failed to find contact: %w", err)
		}
		if len(contacts) == 0 {
			return fmt.Errorf("no contact found matching: %s", *contactIDStr)
		}
		contactID = contacts[0].ID
	}

	err = db.SetContactCadence(database, contactID, *days, *strength)
	if err != nil {
		return fmt.Errorf("failed to set cadence: %w", err)
	}

	fmt.Printf("✓ Set cadence to %d days (%s strength)\n", *days, *strength)
	return nil
}

// DigestCommand generates a daily follow-up digest
func DigestCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("digest", flag.ExitOnError)
	format := fs.String("format", "text", "Output format (text/json/html)")
	_ = fs.Parse(args)

	followups, err := db.GetFollowupList(database, 50)
	if err != nil {
		return fmt.Errorf("failed to get followup list: %w", err)
	}

	if *format == "text" {
		return printTextDigest(followups)
	} else if *format == "json" {
		return printJSONDigest(followups)
	} else if *format == "html" {
		return printHTMLDigest(followups)
	}

	return fmt.Errorf("unsupported format: %s", *format)
}

func printTextDigest(followups []models.FollowupContact) error {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  FOLLOW-UPS FOR %s\n", time.Now().Format("2006-01-02"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Split into categories
	var overdue, dueSoon []models.FollowupContact
	for _, f := range followups {
		if f.DaysSinceContact > f.CadenceDays+7 {
			overdue = append(overdue, f)
		} else if f.DaysSinceContact >= f.CadenceDays-3 {
			dueSoon = append(dueSoon, f)
		}
	}

	if len(overdue) > 0 {
		fmt.Printf("🔴 OVERDUE (%d contacts)\n", len(overdue))
		for _, f := range overdue {
			fmt.Printf("  %-20s  %3d days  (priority: %.0f)\n", f.Name, f.DaysSinceContact, f.PriorityScore)
		}
		fmt.Println()
	}

	if len(dueSoon) > 0 {
		fmt.Printf("🟡 DUE SOON (%d contacts)\n", len(dueSoon))
		for _, f := range dueSoon {
			fmt.Printf("  %-20s  %3d days  (priority: %.0f)\n", f.Name, f.DaysSinceContact, f.PriorityScore)
		}
		fmt.Println()
	}

	return nil
}

func printJSONDigest(followups []models.FollowupContact) error {
	// Simple JSON output for webhook integration
	fmt.Printf("{\"date\":\"%s\",\"followups\":[", time.Now().Format("2006-01-02"))
	for i, f := range followups {
		if i > 0 {
			fmt.Print(",")
		}
		fmt.Printf("{\"name\":\"%s\",\"days\":%d,\"priority\":%.1f}",
			f.Name, f.DaysSinceContact, f.PriorityScore)
	}
	fmt.Println("]}")
	return nil
}

func printHTMLDigest(followups []models.FollowupContact) error {
	fmt.Println("<html><body>")
	fmt.Printf("<h1>Follow-Ups for %s</h1>\n", time.Now().Format("2006-01-02"))
	fmt.Println("<table border='1'>")
	fmt.Println("<tr><th>Name</th><th>Days Since</th><th>Priority</th></tr>")
	for _, f := range followups {
		fmt.Printf("<tr><td>%s</td><td>%d</td><td>%.1f</td></tr>\n",
			f.Name, f.DaysSinceContact, f.PriorityScore)
	}
	fmt.Println("</table>")
	fmt.Println("</body></html>")
	return nil
}
```

**Step 2: Test manually**

```bash
go run . followups digest
go run . followups digest --format json
```

**Step 3: Commit**

```bash
git add cli/followups.go
git commit -m "feat: add set-cadence and digest CLI commands

Add ability to configure cadence and generate daily digests
in multiple formats (text/json/html)."
```

---

## Task 8: Wire Up CLI Commands

**Files:**
- Modify: `main.go`

**Step 1: Find where commands are registered**

```bash
grep -n "AddContactCommand" main.go
```

**Step 2: Add followup subcommands**

In `main.go`, add after other CRM commands:

```go
// In the command routing section
} else if len(os.Args) > 1 && os.Args[1] == "followups" {
	if len(os.Args) < 3 {
		fmt.Println("Usage: pagen followups <command>")
		fmt.Println("Commands: list, log, set-cadence, stats, digest")
		os.Exit(1)
	}

	db, err := db.OpenDatabase(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	subCmd := os.Args[2]
	args := os.Args[3:]

	switch subCmd {
	case "list":
		err = cli.FollowupListCommand(db, args)
	case "log":
		err = cli.LogInteractionCommand(db, args)
	case "set-cadence":
		err = cli.SetCadenceCommand(db, args)
	case "stats":
		err = cli.FollowupStatsCommand(db, args)
	case "digest":
		err = cli.DigestCommand(db, args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown followups command: %s\n", subCmd)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
```

**Step 3: Test commands**

```bash
go run . followups list
go run . followups stats
go run . followups digest
```

**Step 4: Commit**

```bash
git add main.go
git commit -m "feat: wire up followup CLI commands in main

Register followup subcommands in main command router."
```

---

## Task 9: MCP Tools for Follow-Ups

**Files:**
- Create: `mcp/handlers/followup_handlers.go`
- Modify: `cli/mcp.go` (register tools)

**Step 1: Write MCP handler for get_followup_list**

```go
// mcp/handlers/followup_handlers.go
// ABOUTME: MCP handlers for follow-up operations
// ABOUTME: Provides follow-up list, interaction logging, and cadence management to Claude
package handlers

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
)

type FollowupHandlers struct {
	db *sql.DB
}

func NewFollowupHandlers(database *sql.DB) *FollowupHandlers {
	return &FollowupHandlers{db: database}
}

type GetFollowupListInput struct {
	Limit        *int    `json:"limit,omitempty" jsonschema:"description=Maximum number of contacts to return (default 10)"`
	OverdueOnly  *bool   `json:"overdue_only,omitempty" jsonschema:"description=Only show overdue contacts"`
	MinPriority  *float64 `json:"min_priority,omitempty" jsonschema:"description=Minimum priority score"`
}

type GetFollowupListOutput struct {
	Followups []models.FollowupContact `json:"followups"`
	Count     int                      `json:"count"`
}

func (h *FollowupHandlers) GetFollowupList(ctx context.Context, input GetFollowupListInput) (*GetFollowupListOutput, error) {
	limit := 10
	if input.Limit != nil {
		limit = *input.Limit
	}

	followups, err := db.GetFollowupList(h.db, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get followup list: %w", err)
	}

	// Apply filters
	var filtered []models.FollowupContact
	for _, f := range followups {
		if input.OverdueOnly != nil && *input.OverdueOnly && f.PriorityScore <= 0 {
			continue
		}
		if input.MinPriority != nil && f.PriorityScore < *input.MinPriority {
			continue
		}
		filtered = append(filtered, f)
	}

	return &GetFollowupListOutput{
		Followups: filtered,
		Count:     len(filtered),
	}, nil
}

type LogInteractionInput struct {
	ContactID       string  `json:"contact_id" jsonschema:"required,description=Contact ID or name"`
	InteractionType string  `json:"interaction_type" jsonschema:"required,description=Type of interaction (meeting/call/email/message/event)"`
	Notes           *string `json:"notes,omitempty" jsonschema:"description=Notes about the interaction"`
	Sentiment       *string `json:"sentiment,omitempty" jsonschema:"description=Sentiment (positive/neutral/negative)"`
}

type LogInteractionOutput struct {
	Success          bool   `json:"success"`
	Message          string `json:"message"`
	InteractionID    string `json:"interaction_id"`
	UpdatedPriority  float64 `json:"updated_priority"`
}

func (h *FollowupHandlers) LogInteraction(ctx context.Context, input LogInteractionInput) (*LogInteractionOutput, error) {
	// Resolve contact ID
	var contactID uuid.UUID
	parsedID, err := uuid.Parse(input.ContactID)
	if err == nil {
		contactID = parsedID
	} else {
		contacts, err := db.FindContactsByQuery(h.db, input.ContactID)
		if err != nil {
			return nil, fmt.Errorf("failed to find contact: %w", err)
		}
		if len(contacts) == 0 {
			return nil, fmt.Errorf("no contact found matching: %s", input.ContactID)
		}
		contactID = contacts[0].ID
	}

	interaction := &models.InteractionLog{
		ContactID:       contactID,
		InteractionType: input.InteractionType,
		Notes:           *input.Notes,
		Sentiment:       input.Sentiment,
	}

	err = db.LogInteraction(h.db, interaction)
	if err != nil {
		return nil, fmt.Errorf("failed to log interaction: %w", err)
	}

	// Get updated priority
	cadence, _ := db.GetContactCadence(h.db, contactID)
	priority := 0.0
	if cadence != nil {
		priority = cadence.PriorityScore
	}

	return &LogInteractionOutput{
		Success:         true,
		Message:         fmt.Sprintf("Logged %s interaction", input.InteractionType),
		InteractionID:   interaction.ID.String(),
		UpdatedPriority: priority,
	}, nil
}

type SetCadenceInput struct {
	ContactID string `json:"contact_id" jsonschema:"required,description=Contact ID or name"`
	Days      int    `json:"days" jsonschema:"required,description=Cadence in days"`
	Strength  string `json:"strength" jsonschema:"required,description=Relationship strength (weak/medium/strong)"`
}

type SetCadenceOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (h *FollowupHandlers) SetCadence(ctx context.Context, input SetCadenceInput) (*SetCadenceOutput, error) {
	// Resolve contact ID
	var contactID uuid.UUID
	parsedID, err := uuid.Parse(input.ContactID)
	if err == nil {
		contactID = parsedID
	} else {
		contacts, err := db.FindContactsByQuery(h.db, input.ContactID)
		if err != nil {
			return nil, fmt.Errorf("failed to find contact: %w", err)
		}
		if len(contacts) == 0 {
			return nil, fmt.Errorf("no contact found matching: %s", input.ContactID)
		}
		contactID = contacts[0].ID
	}

	err = db.SetContactCadence(h.db, contactID, input.Days, input.Strength)
	if err != nil {
		return nil, fmt.Errorf("failed to set cadence: %w", err)
	}

	return &SetCadenceOutput{
		Success: true,
		Message: fmt.Sprintf("Set cadence to %d days (%s strength)", input.Days, input.Strength),
	}, nil
}
```

**Step 2: Register MCP tools**

In `cli/mcp.go`, add after existing handler creation:

```go
followupHandlers := handlers.NewFollowupHandlers(db)

// After other tool registrations
mcp.AddTool(server, &mcp.Tool{
	Name:        "get_followup_list",
	Description: "Get list of contacts needing follow-up, sorted by priority",
}, followupHandlers.GetFollowupList)

mcp.AddTool(server, &mcp.Tool{
	Name:        "log_interaction",
	Description: "Log an interaction with a contact and update follow-up tracking",
}, followupHandlers.LogInteraction)

mcp.AddTool(server, &mcp.Tool{
	Name:        "set_cadence",
	Description: "Set the follow-up cadence and relationship strength for a contact",
}, followupHandlers.SetCadence)
```

**Step 3: Test with MCP server**

```bash
go run . mcp
# Test in Claude Desktop
```

**Step 4: Commit**

```bash
git add mcp/handlers/followup_handlers.go cli/mcp.go
git commit -m "feat: add MCP tools for follow-up tracking

Add get_followup_list, log_interaction, and set_cadence tools
for Claude integration."
```

---

## Task 10: TUI - Follow-Ups Tab

**Files:**
- Create: `tui/followup_view.go`
- Modify: `tui/tui.go` (add tab)

**Step 1: Create followup view**

```go
// tui/followup_view.go
// ABOUTME: TUI view for follow-up tracking
// ABOUTME: Displays prioritized list of contacts needing follow-up
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
)

type followupView struct {
	table     table.Model
	followups []models.FollowupContact
	width     int
	height    int
}

func newFollowupView() followupView {
	columns := []table.Column{
		{Title: "Status", Width: 3},
		{Title: "Name", Width: 20},
		{Title: "Days", Width: 6},
		{Title: "Priority", Width: 8},
		{Title: "Strength", Width: 8},
		{Title: "Email", Width: 25},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return followupView{
		table: t,
	}
}

func (v *followupView) Init() tea.Cmd {
	return nil
}

func (v *followupView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.table.SetHeight(msg.Height - 10)
	}

	v.table, cmd = v.table.Update(msg)
	return v, cmd
}

func (v *followupView) View() string {
	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(v.table.View())
}

func (v *followupView) loadData(database interface{}) error {
	// Type assert to *sql.DB
	sqlDB, ok := database.(*sql.DB)
	if !ok {
		return fmt.Errorf("invalid database type")
	}

	followups, err := db.GetFollowupList(sqlDB, 100)
	if err != nil {
		return err
	}

	v.followups = followups

	rows := make([]table.Row, len(followups))
	for i, f := range followups {
		indicator := "🟢"
		if f.DaysSinceContact > f.CadenceDays+7 {
			indicator = "🔴"
		} else if f.DaysSinceContact >= f.CadenceDays-3 {
			indicator = "🟡"
		}

		rows[i] = table.Row{
			indicator,
			f.Name,
			fmt.Sprintf("%d", f.DaysSinceContact),
			fmt.Sprintf("%.1f", f.PriorityScore),
			f.RelationshipStrength,
			f.Email,
		}
	}

	v.table.SetRows(rows)
	return nil
}

func (v *followupView) getSelectedContact() *models.FollowupContact {
	if len(v.followups) == 0 {
		return nil
	}

	cursor := v.table.Cursor()
	if cursor >= 0 && cursor < len(v.followups) {
		return &v.followups[cursor]
	}
	return nil
}
```

**Step 2: Wire into main TUI**

In `tui/tui.go`, add followup tab. Find the tab definitions and add:

```go
// In the model struct
type model struct {
	// ... existing fields
	followupView followupView
}

// In the Init or NewModel function
m.followupView = newFollowupView()

// In the tab switching logic
const (
	tabContacts = iota
	tabCompanies
	tabDeals
	tabFollowups  // Add this
)

// In the key handling
case "f":
	m.activeTab = tabFollowups
	m.followupView.loadData(m.db)

// In the View function
case tabFollowups:
	content = m.followupView.View()
```

**Step 3: Test TUI**

```bash
go run .
# Press 'f' to switch to followups tab
```

**Step 4: Commit**

```bash
git add tui/followup_view.go tui/tui.go
git commit -m "feat: add follow-ups tab to TUI

Add interactive follow-up view with priority-based sorting
and visual indicators for overdue contacts."
```

---

## Task 11: Web UI - Follow-Ups Page

**Files:**
- Create: `web/templates/followups.html`
- Modify: `web/server.go` (add route)

**Step 1: Create followups template**

```html
<!-- web/templates/followups.html -->
{{define "followups"}}
<!DOCTYPE html>
<html>
<head>
    <title>Follow-Ups - Pagen CRM</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100">
    <div class="container mx-auto px-4 py-8">
        <h1 class="text-3xl font-bold mb-6">Follow-Ups</h1>

        <div class="bg-white rounded-lg shadow p-6">
            <div class="mb-4 flex gap-4">
                <select id="filter-strength" class="border rounded px-3 py-2">
                    <option value="">All Strengths</option>
                    <option value="strong">Strong</option>
                    <option value="medium">Medium</option>
                    <option value="weak">Weak</option>
                </select>

                <label class="flex items-center gap-2">
                    <input type="checkbox" id="overdue-only" class="rounded">
                    <span>Overdue Only</span>
                </label>
            </div>

            <table class="w-full">
                <thead class="bg-gray-50">
                    <tr>
                        <th class="px-4 py-2 text-left">Name</th>
                        <th class="px-4 py-2 text-left">Days Since</th>
                        <th class="px-4 py-2 text-left">Priority</th>
                        <th class="px-4 py-2 text-left">Strength</th>
                        <th class="px-4 py-2 text-left">Actions</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Followups}}
                    <tr class="border-t hover:bg-gray-50">
                        <td class="px-4 py-3">
                            {{if gt .DaysSinceContact (add .CadenceDays 7)}}🔴{{else if ge .DaysSinceContact (sub .CadenceDays 3)}}🟡{{else}}🟢{{end}}
                            <a href="/contacts/{{.ID}}" class="text-blue-600 hover:underline">{{.Name}}</a>
                        </td>
                        <td class="px-4 py-3">{{.DaysSinceContact}} days</td>
                        <td class="px-4 py-3">{{printf "%.1f" .PriorityScore}}</td>
                        <td class="px-4 py-3">
                            <span class="px-2 py-1 rounded text-sm
                                {{if eq .RelationshipStrength "strong"}}bg-green-100 text-green-800
                                {{else if eq .RelationshipStrength "medium"}}bg-yellow-100 text-yellow-800
                                {{else}}bg-gray-100 text-gray-800{{end}}">
                                {{.RelationshipStrength}}
                            </span>
                        </td>
                        <td class="px-4 py-3">
                            <button
                                hx-post="/followups/log/{{.ID}}"
                                hx-swap="outerHTML"
                                class="bg-blue-500 text-white px-3 py-1 rounded hover:bg-blue-600">
                                Log Contact
                            </button>
                        </td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
    </div>
</body>
</html>
{{end}}
```

**Step 2: Add route handler**

In `web/server.go`:

```go
// Add to server routes
http.HandleFunc("/followups", func(w http.ResponseWriter, r *http.Request) {
	followups, err := db.GetFollowupList(database, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Followups []models.FollowupContact
	}{
		Followups: followups,
	}

	tmpl.ExecuteTemplate(w, "followups", data)
})

// HTMX endpoint for logging interaction
http.HandleFunc("/followups/log/", func(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contactID := strings.TrimPrefix(r.URL.Path, "/followups/log/")
	id, err := uuid.Parse(contactID)
	if err != nil {
		http.Error(w, "Invalid contact ID", http.StatusBadRequest)
		return
	}

	interaction := &models.InteractionLog{
		ContactID:       id,
		InteractionType: models.InteractionMessage,
		Timestamp:       time.Now(),
		Notes:           "Quick contact via web UI",
	}

	err = db.LogInteraction(database, interaction)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(`<td colspan="5" class="px-4 py-3 text-green-600">✓ Interaction logged</td>`))
})
```

**Step 3: Test web UI**

```bash
go run . web
# Visit http://localhost:10666/followups
```

**Step 4: Commit**

```bash
git add web/templates/followups.html web/server.go
git commit -m "feat: add follow-ups page to web UI

Add /followups page with HTMX-powered quick logging
and filter controls."
```

---

## Task 12: Update README and Documentation

**Files:**
- Modify: `README.md`

**Step 1: Add follow-up features to README**

Update features section and add new commands:

```markdown
## Features

- **Contact Management** - Track people with full interaction history
- **Company Management** - Organize contacts by company with industry tracking
- **Deal Pipeline** - Manage sales from prospecting to closed
- **Relationship Tracking** - Map connections between contacts (colleagues, friends, etc.)
- **Follow-Up Tracking** - Never lose touch with your network through smart cadence tracking
- **Universal Query** - Flexible searching across all entity types

...

### Follow-Up Commands

```bash
# List contacts needing follow-up
pagen followups list [--overdue-only] [--strength weak|medium|strong] [--limit 10]

# Log an interaction
pagen followups log --contact "Alice" --type meeting --notes "Coffee chat"

# Set follow-up cadence
pagen followups set-cadence --contact "Bob" --days 14 --strength strong

# View network health stats
pagen followups stats

# Generate daily digest
pagen followups digest [--format text|json|html]
```

### Follow-Up in TUI

Press `f` to view the Follow-Ups tab showing:
- Prioritized list of contacts needing attention
- Visual indicators (🔴 overdue, 🟡 due soon, 🟢 on track)
- Quick interaction logging with `l`
- Cadence adjustment with `c`

### Follow-Up in Web UI

Visit `/followups` for:
- Filterable table of contacts needing follow-up
- One-click interaction logging via HTMX
- Priority-based sorting

### MCP Tools

New tools for Claude Desktop:
- `get_followup_list` - Get prioritized follow-up suggestions
- `log_interaction` - Log interactions and update tracking
- `set_cadence` - Configure follow-up frequency per contact
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: update README with follow-up features

Document new follow-up tracking commands, TUI tab, web page,
and MCP tools."
```

---

## Task 13: Integration Testing

**Files:**
- Create: `.scratch/test_followups.sh`

**Step 1: Write integration test script**

```bash
#!/bin/bash
# .scratch/test_followups.sh
# Integration test for follow-up system

set -e

# Setup
TEST_DB=$(mktemp -t test-followups-XXXXXX.db)
echo "Using test database: $TEST_DB"

# Build
go build -o pagen-test

# Create test data
./pagen-test --db-path "$TEST_DB" crm add-company --name "TestCorp" --industry "Software"
./pagen-test --db-path "$TEST_DB" crm add-contact --name "Alice Test" --email "alice@test.com" --company "TestCorp"
./pagen-test --db-path "$TEST_DB" crm add-contact --name "Bob Test" --email "bob@test.com" --company "TestCorp"

# Test followup commands
echo "Testing followup list..."
./pagen-test --db-path "$TEST_DB" followups list

echo "Testing followup stats..."
./pagen-test --db-path "$TEST_DB" followups stats

echo "Testing set cadence..."
ALICE_ID=$(./pagen-test --db-path "$TEST_DB" crm list-contacts | grep "Alice Test" | awk '{print $1}')
./pagen-test --db-path "$TEST_DB" followups set-cadence --contact "$ALICE_ID" --days 14 --strength strong

echo "Testing log interaction..."
./pagen-test --db-path "$TEST_DB" followups log --contact "$ALICE_ID" --type meeting --notes "Test meeting"

echo "Testing digest..."
./pagen-test --db-path "$TEST_DB" followups digest
./pagen-test --db-path "$TEST_DB" followups digest --format json

# Cleanup
rm -f pagen-test "$TEST_DB"

echo "✓ All integration tests passed!"
```

**Step 2: Make executable and run**

```bash
chmod +x .scratch/test_followups.sh
./.scratch/test_followups.sh
```

**Step 3: Commit**

```bash
git add .scratch/test_followups.sh
git commit -m "test: add follow-up integration tests

Add comprehensive integration test script covering all
follow-up commands and workflows."
```

---

## Execution Options

Plan complete and saved to `docs/plans/2025-11-27-followup-system.md`.

**Two execution options:**

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration with quality gates

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
