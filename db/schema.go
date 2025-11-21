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
`

func InitSchema(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}
