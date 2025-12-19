// ABOUTME: Deal CLI commands
// ABOUTME: Human-friendly commands for managing deals
package cli

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/charm"
)

// AddDealCommand adds a new deal.
func AddDealCommand(client *charm.Client, args []string) error {
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
	existingCompany, err := client.FindCompanyByName(*company)
	if err != nil {
		return fmt.Errorf("failed to lookup company: %w", err)
	}

	var companyUUID uuid.UUID
	var companyName string
	if existingCompany == nil {
		newCompany := &charm.Company{Name: *company}
		if err := client.CreateCompany(newCompany); err != nil {
			return fmt.Errorf("failed to create company: %w", err)
		}
		companyUUID = newCompany.ID
		companyName = newCompany.Name
	} else {
		companyUUID = existingCompany.ID
		companyName = existingCompany.Name
	}

	// Handle optional contact association
	var contactUUID *uuid.UUID
	var contactName string
	if *contact != "" {
		contacts, err := client.ListContacts(&charm.ContactFilter{Query: *contact, Limit: 1})
		if err != nil {
			return fmt.Errorf("failed to lookup contact: %w", err)
		}

		if len(contacts) == 0 {
			// Create contact
			newContact := &charm.Contact{Name: *contact, CompanyID: &companyUUID, CompanyName: companyName}
			if err := client.CreateContact(newContact); err != nil {
				return fmt.Errorf("failed to create contact: %w", err)
			}
			contactUUID = &newContact.ID
			contactName = newContact.Name
		} else {
			contactUUID = &contacts[0].ID
			contactName = contacts[0].Name
		}
	}

	deal := &charm.Deal{
		Title:       *title,
		Amount:      *amount,
		Currency:    *currency,
		Stage:       *stage,
		CompanyID:   companyUUID,
		CompanyName: companyName,
		ContactID:   contactUUID,
		ContactName: contactName,
	}

	if err := client.CreateDeal(deal); err != nil {
		return fmt.Errorf("failed to create deal: %w", err)
	}

	// TODO: Queue to vault sync once sync package is migrated

	fmt.Printf("✓ Deal created: %s (ID: %s)\n", deal.Title, deal.ID)
	fmt.Printf("  Company: %s\n", companyName)
	fmt.Printf("  Amount: $%.2f %s\n", float64(deal.Amount)/100.0, deal.Currency)
	fmt.Printf("  Stage: %s\n", deal.Stage)
	if contactName != "" {
		fmt.Printf("  Contact: %s\n", contactName)
	}

	// Add initial note if provided
	if *notes != "" {
		note := &charm.DealNote{
			DealID:          deal.ID,
			DealTitle:       deal.Title,
			DealCompanyName: companyName,
			Content:         *notes,
		}
		if err := client.CreateDealNote(note); err != nil {
			fmt.Printf("  Warning: Failed to add note: %v\n", err)
		} else {
			fmt.Printf("  Note added\n")
		}
	}

	return nil
}

// ListDealsCommand lists all deals.
func ListDealsCommand(client *charm.Client, args []string) error {
	fs := flag.NewFlagSet("list-deals", flag.ExitOnError)
	stage := fs.String("stage", "", "Filter by stage")
	company := fs.String("company", "", "Filter by company name")
	limit := fs.Int("limit", 50, "Maximum results")
	_ = fs.Parse(args)

	filter := &charm.DealFilter{
		Stage: *stage,
		Limit: *limit,
	}

	if *company != "" {
		existingCompany, err := client.FindCompanyByName(*company)
		if err != nil {
			return fmt.Errorf("failed to lookup company: %w", err)
		}
		if existingCompany != nil {
			filter.CompanyID = &existingCompany.ID
		}
	}

	deals, err := client.ListDeals(filter)
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
		companyName := deal.CompanyName
		if companyName == "" {
			companyName = "-"
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
func DeleteDealCommand(client *charm.Client, args []string) error {
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
	deal, err := client.GetDeal(dealID)
	if err != nil {
		return fmt.Errorf("deal not found: %w", err)
	}
	if deal == nil {
		return fmt.Errorf("deal not found: %s", dealID)
	}

	err = client.DeleteDeal(dealID)
	if err != nil {
		return fmt.Errorf("failed to delete deal: %w", err)
	}

	// TODO: Queue to vault sync once sync package is migrated

	fmt.Printf("✓ Deleted deal: %s\n", dealID)
	return nil
}
