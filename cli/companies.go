// ABOUTME: Company CLI commands
// ABOUTME: Human-friendly commands for managing companies
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

// AddCompanyCommand adds a new company
func AddCompanyCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("add-company", flag.ExitOnError)
	name := fs.String("name", "", "Company name (required)")
	domain := fs.String("domain", "", "Company domain (e.g., acme.com)")
	industry := fs.String("industry", "", "Industry")
	notes := fs.String("notes", "", "Notes about the company")
	_ = fs.Parse(args)

	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	company := &models.Company{
		Name:     *name,
		Domain:   *domain,
		Industry: *industry,
		Notes:    *notes,
	}

	if err := db.CreateCompany(database, company); err != nil {
		return fmt.Errorf("failed to create company: %w", err)
	}

	fmt.Printf("✓ Company created: %s (ID: %s)\n", company.Name, company.ID)
	if company.Domain != "" {
		fmt.Printf("  Domain: %s\n", company.Domain)
	}
	if company.Industry != "" {
		fmt.Printf("  Industry: %s\n", company.Industry)
	}

	return nil
}

// ListCompaniesCommand lists all companies
func ListCompaniesCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("list-companies", flag.ExitOnError)
	query := fs.String("query", "", "Search by name or domain")
	limit := fs.Int("limit", 50, "Maximum results")
	_ = fs.Parse(args)

	companies, err := db.FindCompanies(database, *query, *limit)
	if err != nil {
		return fmt.Errorf("failed to find companies: %w", err)
	}

	if len(companies) == 0 {
		fmt.Println("No companies found")
		return nil
	}

	// Pretty print results
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tDOMAIN\tINDUSTRY\tID")
	_, _ = fmt.Fprintln(w, "----\t------\t--------\t--")

	for _, company := range companies {
		domain := company.Domain
		if domain == "" {
			domain = "-"
		}
		industry := company.Industry
		if industry == "" {
			industry = "-"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			company.Name, domain, industry, company.ID.String()[:8])
	}
	_ = w.Flush()

	fmt.Printf("\nTotal: %d company(ies)\n", len(companies))
	return nil
}

// UpdateCompanyCommand updates an existing company
func UpdateCompanyCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("update-company", flag.ExitOnError)
	name := fs.String("name", "", "Company name")
	domain := fs.String("domain", "", "Domain")
	industry := fs.String("industry", "", "Industry")
	notes := fs.String("notes", "", "Notes")
	_ = fs.Parse(args)

	// First positional arg is the company ID
	if len(fs.Args()) < 1 {
		return fmt.Errorf("company ID is required")
	}

	companyID, err := uuid.Parse(fs.Args()[0])
	if err != nil {
		return fmt.Errorf("invalid company ID: %w", err)
	}

	// Get existing company
	existing, err := db.GetCompany(database, companyID)
	if err != nil {
		return fmt.Errorf("company not found: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("company not found: %s", companyID)
	}

	// Apply updates from flags
	if *name != "" {
		existing.Name = *name
	}
	if *domain != "" {
		existing.Domain = *domain
	}
	if *industry != "" {
		existing.Industry = *industry
	}
	if *notes != "" {
		existing.Notes = *notes
	}

	err = db.UpdateCompany(database, companyID, existing)
	if err != nil {
		return fmt.Errorf("failed to update company: %w", err)
	}

	fmt.Printf("✓ Company updated: %s (ID: %s)\n", existing.Name, companyID)
	return nil
}

// DeleteCompanyCommand deletes a company
func DeleteCompanyCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("delete-company", flag.ExitOnError)
	_ = fs.Parse(args)

	// First positional arg is the company ID
	if len(fs.Args()) < 1 {
		return fmt.Errorf("company ID is required")
	}

	companyID, err := uuid.Parse(fs.Args()[0])
	if err != nil {
		return fmt.Errorf("invalid company ID: %w", err)
	}

	err = db.DeleteCompany(database, companyID)
	if err != nil {
		return fmt.Errorf("failed to delete company: %w", err)
	}

	fmt.Printf("✓ Company deleted: %s\n", companyID)
	return nil
}
