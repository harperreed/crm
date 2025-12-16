// ABOUTME: Deal CLI commands
// ABOUTME: Human-friendly commands for managing deals
package cli

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/harperreed/sweet/vault"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
	"github.com/harperreed/pagen/sync"
)

// queueDealToVault queues a deal change to vault sync.
// Sync failures are non-fatal - the local operation already succeeded.
func queueDealToVault(database *sql.DB, deal *models.Deal, companyName, contactName string, op vault.Op) {
	cfg, err := sync.LoadVaultConfig()
	if err != nil || !cfg.IsConfigured() {
		return
	}

	syncer, err := sync.NewVaultSyncer(cfg, database)
	if err != nil {
		log.Printf("warning: vault sync init failed: %v", err)
		return
	}
	defer func() { _ = syncer.Close() }()

	if err := syncer.QueueDealChange(context.Background(), deal, companyName, contactName, op); err != nil {
		log.Printf("warning: vault sync queue failed: %v", err)
	}
}

// AddDealCommand adds a new deal.
func AddDealCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("add-deal", flag.ExitOnError)
	title := fs.String("title", "", "Deal title (required)")
	company := fs.String("company", "", "Company name (required)")
	contact := fs.String("contact", "", "Contact name (optional)")
	amount := fs.Int64("amount", 0, "Deal amount in cents")
	currency := fs.String("currency", "USD", "Currency code")
	stage := fs.String("stage", "prospecting", "Stage (prospecting, qualification, proposal, negotiation, closed_won, closed_lost)")
	notes := fs.String("notes", "", "Initial notes")
	_ = fs.Parse(args)

	if *title == "" {
		return fmt.Errorf("--title is required")
	}
	if *company == "" {
		return fmt.Errorf("--company is required")
	}

	// Find or create company
	existingCompany, err := db.FindCompanyByName(database, *company)
	if err != nil {
		return fmt.Errorf("failed to lookup company: %w", err)
	}

	var companyUUID uuid.UUID
	if existingCompany == nil {
		newCompany := &models.Company{Name: *company}
		if err := db.CreateCompany(database, newCompany); err != nil {
			return fmt.Errorf("failed to create company: %w", err)
		}
		companyUUID = newCompany.ID
	} else {
		companyUUID = existingCompany.ID
	}

	// Handle optional contact association
	var contactUUID *uuid.UUID
	if *contact != "" {
		contacts, err := db.FindContacts(database, *contact, nil, 1)
		if err != nil {
			return fmt.Errorf("failed to lookup contact: %w", err)
		}

		if len(contacts) == 0 {
			// Create contact
			newContact := &models.Contact{Name: *contact, CompanyID: &companyUUID}
			if err := db.CreateContact(database, newContact); err != nil {
				return fmt.Errorf("failed to create contact: %w", err)
			}
			contactUUID = &newContact.ID
		} else {
			contactUUID = &contacts[0].ID
		}
	}

	deal := &models.Deal{
		Title:     *title,
		Amount:    *amount,
		Currency:  *currency,
		Stage:     *stage,
		CompanyID: companyUUID,
		ContactID: contactUUID,
	}

	if err := db.CreateDeal(database, deal); err != nil {
		return fmt.Errorf("failed to create deal: %w", err)
	}

	// Queue to vault sync (non-fatal)
	queueDealToVault(database, deal, *company, *contact, vault.OpUpsert)

	fmt.Printf("✓ Deal created: %s (ID: %s)\n", deal.Title, deal.ID)
	fmt.Printf("  Company: %s\n", *company)
	fmt.Printf("  Amount: $%.2f %s\n", float64(deal.Amount)/100.0, deal.Currency)
	fmt.Printf("  Stage: %s\n", deal.Stage)
	if *contact != "" {
		fmt.Printf("  Contact: %s\n", *contact)
	}

	// Add initial note if provided
	if *notes != "" {
		note := &models.DealNote{
			DealID:  deal.ID,
			Content: *notes,
		}
		if err := db.AddDealNote(database, note); err != nil {
			fmt.Printf("  Warning: Failed to add note: %v\n", err)
		} else {
			fmt.Printf("  Note added\n")
		}
	}

	return nil
}

// ListDealsCommand lists all deals.
func ListDealsCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("list-deals", flag.ExitOnError)
	stage := fs.String("stage", "", "Filter by stage")
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

	deals, err := db.FindDeals(database, *stage, companyIDPtr, *limit)
	if err != nil {
		return fmt.Errorf("failed to find deals: %w", err)
	}

	if len(deals) == 0 {
		fmt.Println("No deals found")
		return nil
	}

	// Pretty print results
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TITLE\tCOMPANY\tAMOUNT\tSTAGE\tID")
	_, _ = fmt.Fprintln(w, "-----\t-------\t------\t-----\t--")

	for _, deal := range deals {
		companyName := "-"
		// Get company name
		if company, err := db.GetCompany(database, deal.CompanyID); err == nil && company != nil {
			companyName = company.Name
		}

		amountStr := fmt.Sprintf("$%.2f", float64(deal.Amount)/100.0)

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			deal.Title, companyName, amountStr, deal.Stage, deal.ID.String()[:8])
	}
	_ = w.Flush()

	// Calculate total
	var total int64
	for _, deal := range deals {
		total += deal.Amount
	}

	fmt.Printf("\nTotal: %d deal(s) - $%.2f\n", len(deals), float64(total)/100.0)
	return nil
}

// DeleteDealCommand deletes a deal.
func DeleteDealCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("delete-deal", flag.ExitOnError)
	_ = fs.Parse(args)

	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: delete-deal <id>")
	}

	dealID, err := uuid.Parse(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("invalid deal ID: %w", err)
	}

	// Get deal before deletion for vault sync
	deal, err := db.GetDeal(database, dealID)
	if err != nil {
		return fmt.Errorf("deal not found: %w", err)
	}
	if deal == nil {
		return fmt.Errorf("deal not found: %s", dealID)
	}

	// Look up company name for vault sync
	var companyName string
	companyRecord, err := db.GetCompany(database, deal.CompanyID)
	if err == nil && companyRecord != nil {
		companyName = companyRecord.Name
	}

	// Look up contact name for vault sync
	var contactName string
	if deal.ContactID != nil {
		contactRecord, err := db.GetContact(database, *deal.ContactID)
		if err == nil && contactRecord != nil {
			contactName = contactRecord.Name
		}
	}

	err = db.DeleteDeal(database, dealID)
	if err != nil {
		return fmt.Errorf("failed to delete deal: %w", err)
	}

	// Queue to vault sync (non-fatal)
	queueDealToVault(database, deal, companyName, contactName, vault.OpDelete)

	fmt.Printf("✓ Deleted deal: %s\n", dealID)
	return nil
}
