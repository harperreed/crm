// ABOUTME: Canonicalizes optional CRM collection fields at storage boundaries.
// ABOUTME: Keeps contact and company Fields and Tags non-nil across backends.
package storage

import "github.com/harperreed/crm/v2/internal/models"

func canonicalContact(contact *models.Contact) *models.Contact {
	candidate := *contact
	canonicalizeContactCollections(&candidate)
	return &candidate
}

func canonicalizeContactCollections(contact *models.Contact) {
	if contact.Fields == nil {
		contact.Fields = make(map[string]any)
	}
	if contact.Tags == nil {
		contact.Tags = []string{}
	}
}

func canonicalCompany(company *models.Company) *models.Company {
	candidate := *company
	canonicalizeCompanyCollections(&candidate)
	return &candidate
}

func canonicalizeCompanyCollections(company *models.Company) {
	if company.Fields == nil {
		company.Fields = make(map[string]any)
	}
	if company.Tags == nil {
		company.Tags = []string{}
	}
}
