// ABOUTME: Storage interface defining the contract for CRM data backends.
// ABOUTME: Includes filter types, search results, and sentinel errors for common failure cases.
package storage

import (
	"errors"

	"github.com/google/uuid"
	"github.com/harperreed/crm/internal/models"
)

var (
	ErrContactNotFound      = errors.New("contact not found")
	ErrCompanyNotFound      = errors.New("company not found")
	ErrRelationshipNotFound = errors.New("relationship not found")
	ErrPrefixTooShort       = errors.New("prefix must be at least 6 characters")
	ErrAmbiguousPrefix      = errors.New("prefix matches multiple records")
)

// Storage defines the contract that all CRM data backends must satisfy.
type Storage interface {
	CreateContact(contact *models.Contact) error
	GetContact(id uuid.UUID) (*models.Contact, error)
	GetContactByPrefix(prefix string) (*models.Contact, error)
	ListContacts(filter *ContactFilter) ([]*models.Contact, error)
	UpdateContact(contact *models.Contact) error
	DeleteContact(id uuid.UUID) error

	CreateCompany(company *models.Company) error
	GetCompany(id uuid.UUID) (*models.Company, error)
	GetCompanyByPrefix(prefix string) (*models.Company, error)
	ListCompanies(filter *CompanyFilter) ([]*models.Company, error)
	UpdateCompany(company *models.Company) error
	DeleteCompany(id uuid.UUID) error

	CreateRelationship(rel *models.Relationship) error
	ListRelationships(entityID uuid.UUID) ([]*models.Relationship, error)
	DeleteRelationship(id uuid.UUID) error

	Search(query string) (*SearchResults, error)

	Close() error
}

// ContactFilter controls which contacts are returned by ListContacts.
type ContactFilter struct {
	Tag    *string
	Search string
	Limit  int
}

// CompanyFilter controls which companies are returned by ListCompanies.
type CompanyFilter struct {
	Tag    *string
	Search string
	Limit  int
}

// SearchResults holds the combined results of a cross-entity search.
type SearchResults struct {
	Contacts  []*models.Contact
	Companies []*models.Company
}
