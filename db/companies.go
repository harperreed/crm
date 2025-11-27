// ABOUTME: Company database operations
// ABOUTME: Handles CRUD operations and company lookups
package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/models"
)

func CreateCompany(db *sql.DB, company *models.Company) error {
	company.ID = uuid.New()
	now := time.Now()
	company.CreatedAt = now
	company.UpdatedAt = now

	_, err := db.Exec(`
		INSERT INTO companies (id, name, domain, industry, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, company.ID.String(), company.Name, company.Domain, company.Industry, company.Notes, company.CreatedAt, company.UpdatedAt)

	return err
}

func GetCompany(db *sql.DB, id uuid.UUID) (*models.Company, error) {
	company := &models.Company{}
	err := db.QueryRow(`
		SELECT id, name, domain, industry, notes, created_at, updated_at
		FROM companies WHERE id = ?
	`, id.String()).Scan(
		&company.ID,
		&company.Name,
		&company.Domain,
		&company.Industry,
		&company.Notes,
		&company.CreatedAt,
		&company.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return company, err
}

func FindCompanies(db *sql.DB, query string, limit int) ([]models.Company, error) {
	if limit <= 0 {
		limit = 10
	}

	searchPattern := "%" + strings.ToLower(query) + "%"
	rows, err := db.Query(`
		SELECT id, name, domain, industry, notes, created_at, updated_at
		FROM companies
		WHERE LOWER(name) LIKE ? OR LOWER(domain) LIKE ?
		ORDER BY created_at DESC
		LIMIT ?
	`, searchPattern, searchPattern, limit)

	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var companies []models.Company
	for rows.Next() {
		var c models.Company
		if err := rows.Scan(&c.ID, &c.Name, &c.Domain, &c.Industry, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		companies = append(companies, c)
	}

	return companies, rows.Err()
}

func FindCompanyByName(db *sql.DB, name string) (*models.Company, error) {
	company := &models.Company{}
	err := db.QueryRow(`
		SELECT id, name, domain, industry, notes, created_at, updated_at
		FROM companies WHERE LOWER(name) = LOWER(?)
	`, name).Scan(
		&company.ID,
		&company.Name,
		&company.Domain,
		&company.Industry,
		&company.Notes,
		&company.CreatedAt,
		&company.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return company, err
}

func UpdateCompany(db *sql.DB, id uuid.UUID, updates *models.Company) error {
	updates.UpdatedAt = time.Now()

	_, err := db.Exec(`
		UPDATE companies
		SET name = ?, domain = ?, industry = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`, updates.Name, updates.Domain, updates.Industry, updates.Notes, updates.UpdatedAt, id.String())

	return err
}

func DeleteCompany(db *sql.DB, id uuid.UUID) error {
	// Check if company has deals
	var dealCount int
	err := db.QueryRow(`SELECT COUNT(*) FROM deals WHERE company_id = ?`, id.String()).Scan(&dealCount)
	if err != nil {
		return fmt.Errorf("failed to check deals: %w", err)
	}
	if dealCount > 0 {
		return fmt.Errorf("cannot delete company with %d active deals", dealCount)
	}

	// Set contact.company_id to NULL for affected contacts
	_, err = db.Exec(`UPDATE contacts SET company_id = NULL WHERE company_id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("failed to update contacts: %w", err)
	}

	// Delete the company
	_, err = db.Exec(`DELETE FROM companies WHERE id = ?`, id.String())
	return err
}
