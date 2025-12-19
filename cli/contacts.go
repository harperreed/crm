// ABOUTME: Contact CLI commands
// ABOUTME: Human-friendly commands for managing contacts
package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/harperreed/sweet/vault"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/charm"
	"github.com/harperreed/pagen/sync"
)

// queueContactToVault queues a contact change to vault sync.
// Sync failures are non-fatal - the local operation already succeeded.
// NOTE: Currently disabled during migration to charm KV - will be re-enabled when vault sync is migrated.
func queueContactToVault(client *charm.Client, contact *charm.Contact, op vault.Op) {
	cfg, err := sync.LoadVaultConfig()
	if err != nil || !cfg.IsConfigured() {
		return // Vault sync not configured, silently skip
	}

	// TODO: Re-enable when vault sync is migrated to charm KV
	_ = cfg
	_ = contact
	_ = op
	log.Printf("warning: vault sync temporarily disabled during charm KV migration")
}

// AddContactCommand adds a new contact.
func AddContactCommand(client *charm.Client, args []string) error {
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

	contact := &charm.Contact{
		Name:  *name,
		Email: *email,
		Phone: *phone,
		Notes: *notes,
	}

	// Handle company association
	if *company != "" {
		existingCompany, err := client.FindCompanyByName(*company)
		if err != nil {
			return fmt.Errorf("failed to lookup company: %w", err)
		}

		if existingCompany == nil {
			// Create company
			newCompany := &charm.Company{Name: *company}
			if err := client.CreateCompany(newCompany); err != nil {
				return fmt.Errorf("failed to create company: %w", err)
			}
			contact.CompanyID = &newCompany.ID
			contact.CompanyName = newCompany.Name
		} else {
			contact.CompanyID = &existingCompany.ID
			contact.CompanyName = existingCompany.Name
		}
	}

	if err := client.CreateContact(contact); err != nil {
		return fmt.Errorf("failed to create contact: %w", err)
	}

	// Queue to vault sync (non-fatal)
	queueContactToVault(client, contact, vault.OpUpsert)

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

// ListContactsCommand lists all contacts.
func ListContactsCommand(client *charm.Client, args []string) error {
	fs := flag.NewFlagSet("list-contacts", flag.ExitOnError)
	query := fs.String("query", "", "Search by name or email")
	company := fs.String("company", "", "Filter by company name")
	limit := fs.Int("limit", 50, "Maximum results")
	_ = fs.Parse(args)

	var companyIDPtr *uuid.UUID
	if *company != "" {
		existingCompany, err := client.FindCompanyByName(*company)
		if err != nil {
			return fmt.Errorf("failed to lookup company: %w", err)
		}
		if existingCompany != nil {
			companyIDPtr = &existingCompany.ID
		}
	}

	contacts, err := client.ListContacts(&charm.ContactFilter{
		Query:     *query,
		CompanyID: companyIDPtr,
		Limit:     *limit,
	})
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
		if contact.CompanyName != "" {
			companyName = contact.CompanyName
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			contact.Name, email, phone, companyName, contact.ID.String()[:8])
	}
	_ = w.Flush()

	fmt.Printf("\nTotal: %d contact(s)\n", len(contacts))
	return nil
}

// UpdateContactCommand updates an existing contact.
func UpdateContactCommand(client *charm.Client, args []string) error {
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
	existing, err := client.GetContact(contactID)
	if err != nil {
		return fmt.Errorf("contact not found: %w", err)
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
		existingCompany, err := client.FindCompanyByName(*company)
		if err != nil {
			return fmt.Errorf("failed to lookup company: %w", err)
		}
		if existingCompany == nil {
			return fmt.Errorf("company not found: %s", *company)
		}
		existing.CompanyID = &existingCompany.ID
		existing.CompanyName = existingCompany.Name
	}

	err = client.UpdateContact(existing)
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	// Queue to vault sync (non-fatal)
	queueContactToVault(client, existing, vault.OpUpsert)

	fmt.Printf("✓ Contact updated: %s (ID: %s)\n", existing.Name, contactID)
	return nil
}

// DeleteContactCommand deletes a contact.
func DeleteContactCommand(client *charm.Client, args []string) error {
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

	// Get contact before deletion for vault sync
	contact, err := client.GetContact(contactID)
	if err != nil {
		return fmt.Errorf("contact not found: %w", err)
	}

	err = client.DeleteContact(contactID)
	if err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	// Queue to vault sync (non-fatal)
	queueContactToVault(client, contact, vault.OpDelete)

	fmt.Printf("✓ Contact deleted: %s\n", contactID)
	return nil
}
