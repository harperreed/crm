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
	fmt.Fprintln(w, "NAME\tEMAIL\tPHONE\tCOMPANY\tID")
	fmt.Fprintln(w, "----\t-----\t-----\t-------\t--")

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

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			contact.Name, email, phone, companyName, contact.ID.String()[:8])
	}
	w.Flush()

	fmt.Printf("\nTotal: %d contact(s)\n", len(contacts))
	return nil
}
