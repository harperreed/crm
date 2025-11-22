// ABOUTME: Deal CLI commands
// ABOUTME: Human-friendly commands for managing deals
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

// AddDealCommand adds a new deal
func AddDealCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("add-deal", flag.ExitOnError)
	title := fs.String("title", "", "Deal title (required)")
	company := fs.String("company", "", "Company name (required)")
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

	deal := &models.Deal{
		Title:     *title,
		Amount:    *amount,
		Currency:  *currency,
		Stage:     *stage,
		CompanyID: companyUUID,
	}

	if err := db.CreateDeal(database, deal); err != nil {
		return fmt.Errorf("failed to create deal: %w", err)
	}

	fmt.Printf("✓ Deal created: %s (ID: %s)\n", deal.Title, deal.ID)
	fmt.Printf("  Company: %s\n", *company)
	fmt.Printf("  Amount: $%.2f %s\n", float64(deal.Amount)/100.0, deal.Currency)
	fmt.Printf("  Stage: %s\n", deal.Stage)

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

// ListDealsCommand lists all deals
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
	fmt.Fprintln(w, "TITLE\tCOMPANY\tAMOUNT\tSTAGE\tID")
	fmt.Fprintln(w, "-----\t-------\t------\t-----\t--")

	for _, deal := range deals {
		companyName := "-"
		// Get company name
		if company, err := db.GetCompany(database, deal.CompanyID); err == nil && company != nil {
			companyName = company.Name
		}

		amountStr := fmt.Sprintf("$%.2f", float64(deal.Amount)/100.0)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			deal.Title, companyName, amountStr, deal.Stage, deal.ID.String()[:8])
	}
	w.Flush()

	// Calculate total
	var total int64
	for _, deal := range deals {
		total += deal.Amount
	}

	fmt.Printf("\nTotal: %d deal(s) - $%.2f\n", len(deals), float64(total)/100.0)
	return nil
}
