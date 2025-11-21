// ABOUTME: Company CLI commands
// ABOUTME: Human-friendly commands for managing companies
package cli

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/harperreed/crm-mcp/db"
	"github.com/harperreed/crm-mcp/models"
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
	fmt.Fprintln(w, "NAME\tDOMAIN\tINDUSTRY\tID")
	fmt.Fprintln(w, "----\t------\t--------\t--")

	for _, company := range companies {
		domain := company.Domain
		if domain == "" {
			domain = "-"
		}
		industry := company.Industry
		if industry == "" {
			industry = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			company.Name, domain, industry, company.ID.String()[:8])
	}
	w.Flush()

	fmt.Printf("\nTotal: %d company(ies)\n", len(companies))
	return nil
}
