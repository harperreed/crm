// ABOUTME: Company database operations
// ABOUTME: Handles CRUD operations and company lookups
package db

import (
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/crm-mcp/models"
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
	defer rows.Close()

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
