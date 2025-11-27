// ABOUTME: Contact CLI commands
// ABOUTME: Human-friendly commands for managing contacts
package cli

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
)

// AddContactCommand adds a new contact
func AddContactCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("add-contact", flag.ExitOnError)
	name := fs.String("name", "", "Contact name (required)")
	email := fs.String("email", "", "Email address")
	phone := fs.String("phone", "", "Phone number")
	company := fs.String("company", "", "Company name")
	notes := fs.String("notes", "", "Notes about the contact")
	_ = fs.Parse(args)

	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	contact := &models.Contact{
		Name:  *name,
		Email: *email,
		Phone: *phone,
		Notes: *notes,
	}

	// Handle company association
	if *company != "" {
		existingCompany, err := db.FindCompanyByName(database, *company)
		if err != nil {
			return fmt.Errorf("failed to lookup company: %w", err)
		}

		if existingCompany == nil {
			// Create company
			newCompany := &models.Company{Name: *company}
			if err := db.CreateCompany(database, newCompany); err != nil {
				return fmt.Errorf("failed to create company: %w", err)
			}
			contact.CompanyID = &newCompany.ID
		} else {
			contact.CompanyID = &existingCompany.ID
		}
	}

	if err := db.CreateContact(database, contact); err != nil {
		return fmt.Errorf("failed to create contact: %w", err)
	}

	fmt.Printf("✓ Contact created: %s (ID: %s)\n", contact.Name, contact.ID)
	if contact.Email != "" {
		fmt.Printf("  Email: %s\n", contact.Email)
	}
	if contact.Phone != "" {
		fmt.Printf("  Phone: %s\n", contact.Phone)
	}
	if *company != "" {
		fmt.Printf("  Company: %s\n", *company)
	}

	return nil
}

// ListContactsCommand lists all contacts
func ListContactsCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("list-contacts", flag.ExitOnError)
	query := fs.String("query", "", "Search by name or email")
	company := fs.String("company", "", "Filter by company name")
	limit := fs.Int("limit", 50, "Maximum results")
	_ = fs.Parse(args)

	var companyIDPtr *uuid.UUID
	if *company != "" {
		existingCompany, err := db.FindCompanyByName(database, *company)
		if err != nil {
			return fmt.Errorf("failed to lookup company: %w", err)
		}
		if existingCompany != nil {
			companyIDPtr = &existingCompany.ID
		}
	}

	contacts, err := db.FindContacts(database, *query, companyIDPtr, *limit)
	if err != nil {
		return fmt.Errorf("failed to find contacts: %w", err)
	}

	if len(contacts) == 0 {
		fmt.Println("No contacts found")
		return nil
	}

	// Pretty print results
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tEMAIL\tPHONE\tCOMPANY\tID")
	_, _ = fmt.Fprintln(w, "----\t-----\t-----\t-------\t--")

	for _, contact := range contacts {
		email := contact.Email
		if email == "" {
			email = "-"
		}
		phone := contact.Phone
		if phone == "" {
			phone = "-"
		}

		companyName := "-"
		if contact.CompanyID != nil {
			company, err := db.GetCompany(database, *contact.CompanyID)
			if err == nil && company != nil {
				companyName = company.Name
			}
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			contact.Name, email, phone, companyName, contact.ID.String()[:8])
	}
	_ = w.Flush()

	fmt.Printf("\nTotal: %d contact(s)\n", len(contacts))
	return nil
}

// UpdateContactCommand updates an existing contact
func UpdateContactCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("update-contact", flag.ExitOnError)
	name := fs.String("name", "", "Contact name")
	email := fs.String("email", "", "Email address")
	phone := fs.String("phone", "", "Phone number")
	company := fs.String("company", "", "Company name")
	notes := fs.String("notes", "", "Notes about the contact")
	_ = fs.Parse(args)

	// First positional arg is the contact ID
	if len(fs.Args()) < 1 {
		return fmt.Errorf("contact ID is required")
	}

	contactID, err := uuid.Parse(fs.Args()[0])
	if err != nil {
		return fmt.Errorf("invalid contact ID: %w", err)
	}

	// Get existing contact
	existing, err := db.GetContact(database, contactID)
	if err != nil {
		return fmt.Errorf("contact not found: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("contact not found: %s", contactID)
	}

	// Apply updates from flags
	if *name != "" {
		existing.Name = *name
	}
	if *email != "" {
		existing.Email = *email
	}
	if *phone != "" {
		existing.Phone = *phone
	}
	if *notes != "" {
		existing.Notes = *notes
	}

	if *company != "" {
		existingCompany, err := db.FindCompanyByName(database, *company)
		if err != nil {
			return fmt.Errorf("failed to lookup company: %w", err)
		}
		if existingCompany == nil {
			return fmt.Errorf("company not found: %s", *company)
		}
		existing.CompanyID = &existingCompany.ID
	}

	err = db.UpdateContact(database, contactID, existing)
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	fmt.Printf("✓ Contact updated: %s (ID: %s)\n", existing.Name, contactID)
	return nil
}

// DeleteContactCommand deletes a contact
func DeleteContactCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("delete-contact", flag.ExitOnError)
	_ = fs.Parse(args)

	// First positional arg is the contact ID
	if len(fs.Args()) < 1 {
		return fmt.Errorf("contact ID is required")
	}

	contactID, err := uuid.Parse(fs.Args()[0])
	if err != nil {
		return fmt.Errorf("invalid contact ID: %w", err)
	}

	err = db.DeleteContact(database, contactID)
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	fmt.Printf("✓ Contact deleted: %s\n", contactID)
	return nil
}
