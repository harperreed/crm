// ABOUTME: Export commands for contacts, companies, and deals
// ABOUTME: Supports YAML and Markdown output formats

package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harperreed/pagen/repository"
	"gopkg.in/yaml.v3"
)

// ExportContactsCommand exports contacts to YAML or Markdown.
func ExportContactsCommand(db *repository.DB, args []string) error {
	fs := flag.NewFlagSet("export-contacts", flag.ExitOnError)
	format := fs.String("format", "yaml", "Output format (yaml|markdown)")
	output := fs.String("output", "", "Output file (default: stdout)")
	_ = fs.Parse(args)

	contacts, err := db.GetAllContacts()
	if err != nil {
		return fmt.Errorf("failed to get contacts: %w", err)
	}

	var out string
	switch *format {
	case "yaml":
		out, err = exportContactsYAML(contacts)
	case "markdown":
		out, err = exportContactsMarkdown(contacts)
	default:
		return fmt.Errorf("unknown format: %s", *format)
	}
	if err != nil {
		return err
	}

	return writeOutput(out, *output)
}

// ExportCompaniesCommand exports companies to YAML or Markdown.
func ExportCompaniesCommand(db *repository.DB, args []string) error {
	fs := flag.NewFlagSet("export-companies", flag.ExitOnError)
	format := fs.String("format", "yaml", "Output format (yaml|markdown)")
	output := fs.String("output", "", "Output file (default: stdout)")
	_ = fs.Parse(args)

	companies, err := db.GetAllCompanies()
	if err != nil {
		return fmt.Errorf("failed to get companies: %w", err)
	}

	var out string
	switch *format {
	case "yaml":
		out, err = exportCompaniesYAML(companies)
	case "markdown":
		out, err = exportCompaniesMarkdown(companies)
	default:
		return fmt.Errorf("unknown format: %s", *format)
	}
	if err != nil {
		return err
	}

	return writeOutput(out, *output)
}

// ExportDealsCommand exports deals to YAML or Markdown.
func ExportDealsCommand(db *repository.DB, args []string) error {
	fs := flag.NewFlagSet("export-deals", flag.ExitOnError)
	format := fs.String("format", "yaml", "Output format (yaml|markdown)")
	output := fs.String("output", "", "Output file (default: stdout)")
	_ = fs.Parse(args)

	deals, err := db.GetAllDeals()
	if err != nil {
		return fmt.Errorf("failed to get deals: %w", err)
	}

	var out string
	switch *format {
	case "yaml":
		out, err = exportDealsYAML(deals)
	case "markdown":
		out, err = exportDealsMarkdown(deals)
	default:
		return fmt.Errorf("unknown format: %s", *format)
	}
	if err != nil {
		return err
	}

	return writeOutput(out, *output)
}

// ExportAllCommand exports all data to a directory.
func ExportAllCommand(db *repository.DB, args []string) error {
	fs := flag.NewFlagSet("export-all", flag.ExitOnError)
	format := fs.String("format", "yaml", "Output format (yaml)")
	outputDir := fs.String("output-dir", ".", "Output directory")
	_ = fs.Parse(args)

	if *format != "yaml" {
		return fmt.Errorf("export all only supports yaml format")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Export contacts
	contacts, err := db.GetAllContacts()
	if err != nil {
		return fmt.Errorf("failed to get contacts: %w", err)
	}
	contactsYAML, err := exportContactsYAML(contacts)
	if err != nil {
		return err
	}
	if err := writeOutput(contactsYAML, filepath.Join(*outputDir, "contacts.yaml")); err != nil {
		return err
	}
	fmt.Printf("Exported %d contacts to %s/contacts.yaml\n", len(contacts), *outputDir)

	// Export companies
	companies, err := db.GetAllCompanies()
	if err != nil {
		return fmt.Errorf("failed to get companies: %w", err)
	}
	companiesYAML, err := exportCompaniesYAML(companies)
	if err != nil {
		return err
	}
	if err := writeOutput(companiesYAML, filepath.Join(*outputDir, "companies.yaml")); err != nil {
		return err
	}
	fmt.Printf("Exported %d companies to %s/companies.yaml\n", len(companies), *outputDir)

	// Export deals
	deals, err := db.GetAllDeals()
	if err != nil {
		return fmt.Errorf("failed to get deals: %w", err)
	}
	dealsYAML, err := exportDealsYAML(deals)
	if err != nil {
		return err
	}
	if err := writeOutput(dealsYAML, filepath.Join(*outputDir, "deals.yaml")); err != nil {
		return err
	}
	fmt.Printf("Exported %d deals to %s/deals.yaml\n", len(deals), *outputDir)

	return nil
}

// YAML Export Helpers

type yamlExport struct {
	Version    string      `yaml:"version"`
	ExportedAt string      `yaml:"exported_at"`
	Tool       string      `yaml:"tool"`
	Data       interface{} `yaml:"data,omitempty"`
}

func exportContactsYAML(contacts []*repository.Contact) (string, error) {
	export := yamlExport{
		Version:    "1.0",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Tool:       "pagen",
		Data:       contacts,
	}

	data, err := yaml.Marshal(export)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return string(data), nil
}

func exportCompaniesYAML(companies []*repository.Company) (string, error) {
	export := yamlExport{
		Version:    "1.0",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Tool:       "pagen",
		Data:       companies,
	}

	data, err := yaml.Marshal(export)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return string(data), nil
}

func exportDealsYAML(deals []*repository.Deal) (string, error) {
	export := yamlExport{
		Version:    "1.0",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Tool:       "pagen",
		Data:       deals,
	}

	data, err := yaml.Marshal(export)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return string(data), nil
}

// Markdown Export Helpers

func exportContactsMarkdown(contacts []*repository.Contact) (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Pagen Export - Contacts - %s\n\n", time.Now().Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Total: %d contacts\n\n", len(contacts)))
	sb.WriteString("---\n\n")

	for _, contact := range contacts {
		sb.WriteString(fmt.Sprintf("## %s\n\n", contact.Name))

		if contact.Email != "" {
			sb.WriteString(fmt.Sprintf("- **Email**: %s\n", contact.Email))
		}
		if contact.Phone != "" {
			sb.WriteString(fmt.Sprintf("- **Phone**: %s\n", contact.Phone))
		}
		if contact.CompanyName != "" {
			sb.WriteString(fmt.Sprintf("- **Company**: %s\n", contact.CompanyName))
		}
		if contact.LastContactedAt != nil {
			sb.WriteString(fmt.Sprintf("- **Last Contacted**: %s\n", contact.LastContactedAt.Format("2006-01-02")))
		}
		if contact.Notes != "" {
			sb.WriteString(fmt.Sprintf("\n### Notes\n\n%s\n", contact.Notes))
		}
		sb.WriteString("\n---\n\n")
	}

	return sb.String(), nil
}

func exportCompaniesMarkdown(companies []*repository.Company) (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Pagen Export - Companies - %s\n\n", time.Now().Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Total: %d companies\n\n", len(companies)))
	sb.WriteString("---\n\n")

	for _, company := range companies {
		sb.WriteString(fmt.Sprintf("## %s\n\n", company.Name))

		if company.Domain != "" {
			sb.WriteString(fmt.Sprintf("- **Domain**: %s\n", company.Domain))
		}
		if company.Industry != "" {
			sb.WriteString(fmt.Sprintf("- **Industry**: %s\n", company.Industry))
		}
		if company.Notes != "" {
			sb.WriteString(fmt.Sprintf("\n### Notes\n\n%s\n", company.Notes))
		}
		sb.WriteString("\n---\n\n")
	}

	return sb.String(), nil
}

func exportDealsMarkdown(deals []*repository.Deal) (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Pagen Export - Deals - %s\n\n", time.Now().Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Total: %d deals\n\n", len(deals)))
	sb.WriteString("---\n\n")

	for _, deal := range deals {
		sb.WriteString(fmt.Sprintf("## %s\n\n", deal.Title))

		sb.WriteString(fmt.Sprintf("- **Stage**: %s\n", deal.Stage))
		if deal.CompanyName != "" {
			sb.WriteString(fmt.Sprintf("- **Company**: %s\n", deal.CompanyName))
		}
		if deal.ContactName != "" {
			sb.WriteString(fmt.Sprintf("- **Contact**: %s\n", deal.ContactName))
		}
		if deal.Amount > 0 {
			sb.WriteString(fmt.Sprintf("- **Amount**: %s %.2f\n", deal.Currency, float64(deal.Amount)/100))
		}
		if deal.ExpectedCloseDate != nil {
			sb.WriteString(fmt.Sprintf("- **Expected Close**: %s\n", deal.ExpectedCloseDate.Format("2006-01-02")))
		}
		sb.WriteString("\n---\n\n")
	}

	return sb.String(), nil
}

// Output Helper

func writeOutput(content, outputPath string) error {
	if outputPath == "" {
		fmt.Print(content)
		return nil
	}

	return os.WriteFile(outputPath, []byte(content), 0644)
}
