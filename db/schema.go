// ABOUTME: Database schema definitions and migrations
// ABOUTME: Handles SQLite table creation and initialization
package db

import (
	"database/sql"
)

const schema = `
CREATE TABLE IF NOT EXISTS companies (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	domain TEXT,
	industry TEXT,
	notes TEXT,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_companies_name ON companies(name);

CREATE TABLE IF NOT EXISTS contacts (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	email TEXT,
	phone TEXT,
	company_id TEXT,
	notes TEXT,
	last_contacted_at DATETIME,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY (company_id) REFERENCES companies(id)
);

CREATE INDEX IF NOT EXISTS idx_contacts_email ON contacts(email);
CREATE INDEX IF NOT EXISTS idx_contacts_company_id ON contacts(company_id);

CREATE TABLE IF NOT EXISTS deals (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	amount INTEGER,
	currency TEXT NOT NULL DEFAULT 'USD',
	stage TEXT NOT NULL,
	company_id TEXT NOT NULL,
	contact_id TEXT,
	expected_close_date DATE,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	last_activity_at DATETIME NOT NULL,
	FOREIGN KEY (company_id) REFERENCES companies(id),
	FOREIGN KEY (contact_id) REFERENCES contacts(id)
);

CREATE INDEX IF NOT EXISTS idx_deals_stage ON deals(stage);
CREATE INDEX IF NOT EXISTS idx_deals_company_id ON deals(company_id);

CREATE TABLE IF NOT EXISTS deal_notes (
	id TEXT PRIMARY KEY,
	deal_id TEXT NOT NULL,
	content TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	FOREIGN KEY (deal_id) REFERENCES deals(id)
);

CREATE INDEX IF NOT EXISTS idx_deal_notes_deal_id ON deal_notes(deal_id);

CREATE TABLE IF NOT EXISTS relationships (
	id TEXT PRIMARY KEY,
	contact_id_1 TEXT NOT NULL,
	contact_id_2 TEXT NOT NULL,
	relationship_type TEXT,
	context TEXT,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY (contact_id_1) REFERENCES contacts(id),
	FOREIGN KEY (contact_id_2) REFERENCES contacts(id)
);

CREATE INDEX IF NOT EXISTS idx_relationships_contact_1 ON relationships(contact_id_1);
CREATE INDEX IF NOT EXISTS idx_relationships_contact_2 ON relationships(contact_id_2);

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
	metadata TEXT,
	FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_interaction_log_contact ON interaction_log(contact_id);
CREATE INDEX IF NOT EXISTS idx_interaction_log_timestamp ON interaction_log(timestamp DESC);

CREATE TABLE IF NOT EXISTS sync_state (
	service TEXT PRIMARY KEY,
	last_sync_time DATETIME,
	last_sync_token TEXT,
	status TEXT CHECK(status IN ('idle', 'syncing', 'error')),
	error_message TEXT,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sync_log (
	id TEXT PRIMARY KEY,
	source_service TEXT NOT NULL,
	source_id TEXT NOT NULL,
	entity_type TEXT NOT NULL,
	entity_id TEXT NOT NULL,
	imported_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	metadata TEXT,
	UNIQUE(source_service, source_id)
);

CREATE INDEX IF NOT EXISTS idx_sync_log_source ON sync_log(source_service, source_id);
CREATE INDEX IF NOT EXISTS idx_sync_log_entity ON sync_log(entity_type, entity_id);

CREATE TABLE IF NOT EXISTS suggestions (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL CHECK(type IN ('deal', 'relationship', 'company')),
	confidence REAL NOT NULL,
	source_service TEXT NOT NULL,
	source_id TEXT,
	source_data TEXT,
	status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'accepted', 'rejected')),
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	reviewed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_suggestions_status ON suggestions(status);
CREATE INDEX IF NOT EXISTS idx_suggestions_type ON suggestions(type);
`

func InitSchema(db *sql.DB) error {
	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	// Run migrations
	if err := migrateInteractionLogMetadata(db); err != nil {
		return err
	}

	return nil
}

// migrateInteractionLogMetadata adds the metadata column if it doesn't exist
// This ensures users upgrading from Phase 1 can continue using the database
func migrateInteractionLogMetadata(db *sql.DB) error {
	// Check if column exists
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('interaction_log')
		WHERE name='metadata'
	`).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err = db.Exec(`ALTER TABLE interaction_log ADD COLUMN metadata TEXT`)
		return err
	}
	return nil
}
