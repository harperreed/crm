// ABOUTME: Entry point for CRM MCP server and CLI
// ABOUTME: Routes to MCP server or CLI commands based on arguments
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/harperreed/pagen/cli"
	"github.com/harperreed/pagen/db"
)

const version = "0.1.3"

func main() {
	// Global flags
	showVersion := flag.Bool("version", false, "Show version and exit")
	dbPath := flag.String("db-path", "", "Database path (default: ~/.local/share/crm/crm.db)")
	initOnly := flag.Bool("init", false, "Initialize database and exit")

	// Parse global flags but don't fail on unknown (for subcommands)
	_ = flag.CommandLine.Parse(os.Args[1:])

	// Handle version flag
	if *showVersion {
		fmt.Printf("pagen version %s\n", version)
		os.Exit(0)
	}

	// Get remaining args after flags
	args := flag.Args()

	// If no command specified, show usage
	if len(args) == 0 {
		printUsage()
		os.Exit(0)
	}

	// Route to top-level command
	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "mcp":
		// MCP server doesn't need database init message
		finalDBPath := getDatabasePath(*dbPath)
		database, err := db.OpenDatabase(finalDBPath)
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer database.Close()

		if err := cli.MCPCommand(database); err != nil {
			log.Fatalf("MCP server failed: %v", err)
		}

	case "crm":
		// CRM subcommands - initialize database with message
		finalDBPath := getDatabasePath(*dbPath)
		database, err := db.OpenDatabase(finalDBPath)
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer database.Close()

		log.Printf("CRM database: %s", finalDBPath)

		// Handle init-only flag
		if *initOnly {
			log.Println("Database initialized successfully")
			os.Exit(0)
		}

		if len(commandArgs) == 0 {
			fmt.Println("Error: crm requires a subcommand")
			printUsage()
			os.Exit(1)
		}

		crmCommand := commandArgs[0]
		crmArgs := commandArgs[1:]

		switch crmCommand {
		// Contact commands
		case "add-contact":
			if err := cli.AddContactCommand(database, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "list-contacts":
			if err := cli.ListContactsCommand(database, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "update-contact":
			if err := cli.UpdateContactCommand(database, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "delete-contact":
			if err := cli.DeleteContactCommand(database, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}

		// Company commands
		case "add-company":
			if err := cli.AddCompanyCommand(database, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "list-companies":
			if err := cli.ListCompaniesCommand(database, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "update-company":
			if err := cli.UpdateCompanyCommand(database, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "delete-company":
			if err := cli.DeleteCompanyCommand(database, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}

		// Deal commands
		case "add-deal":
			if err := cli.AddDealCommand(database, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "list-deals":
			if err := cli.ListDealsCommand(database, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "delete-deal":
			if err := cli.DeleteDealCommand(database, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}

		// Relationship commands
		case "update-relationship":
			if err := cli.UpdateRelationshipCommand(database, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "delete-relationship":
			if err := cli.DeleteRelationshipCommand(database, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}

		default:
			fmt.Printf("Unknown crm command: %s\n\n", crmCommand)
			printUsage()
			os.Exit(1)
		}

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func getDatabasePath(dbPath string) string {
	if dbPath != "" {
		return dbPath
	}
	return filepath.Join(xdg.DataHome, "crm", "crm.db")
}

func printUsage() {
	fmt.Printf(`pagen v%s - Personal Agent toolkit

USAGE:
  pagen [global flags] <command> [subcommand] [flags]

GLOBAL FLAGS:
  --version              Show version and exit
  --db-path <path>       Database path (default: ~/.local/share/crm/crm.db)
  --init                 Initialize database and exit (use with 'crm')

COMMANDS:
  mcp                    Start MCP server for Claude Desktop
  crm                    CRM management commands

MCP SERVER:
  pagen mcp              Start MCP server (for Claude Desktop integration)

CRM COMMANDS:
  pagen crm add-contact     Add a new contact
    --name <name>             Contact name (required)
    --email <email>           Email address
    --phone <phone>           Phone number
    --company <company>       Company name
    --notes <notes>           Notes about contact

  pagen crm list-contacts   List contacts
    --query <text>            Search by name or email
    --company <company>       Filter by company name
    --limit <n>               Max results (default: 50)

  pagen crm update-contact [flags] <id>  Update an existing contact
    --name <name>             Contact name
    --email <email>           Email address
    --phone <phone>           Phone number
    --company <company>       Company name
    --notes <notes>           Notes about contact
    Note: flags must come before the contact ID

  pagen crm delete-contact <id>  Delete a contact

  pagen crm add-company     Add a new company
    --name <name>             Company name (required)
    --domain <domain>         Company domain (e.g., acme.com)
    --industry <industry>     Industry
    --notes <notes>           Notes about company

  pagen crm list-companies  List companies
    --query <text>            Search by name or domain
    --limit <n>               Max results (default: 50)

  pagen crm add-deal        Add a new deal
    --title <title>           Deal title (required)
    --company <company>       Company name (required)
    --amount <cents>          Deal amount in cents
    --currency <code>         Currency code (default: USD)
    --stage <stage>           Stage (default: prospecting)
    --notes <notes>           Initial notes

  pagen crm list-deals      List deals
    --stage <stage>           Filter by stage
    --company <company>       Filter by company name
    --limit <n>               Max results (default: 50)

  pagen crm delete-deal <id>   Delete a deal

  pagen crm update-relationship [flags] <id>  Update a relationship
    --type <type>             Relationship type
    --context <context>       Relationship context
    Note: flags must come before the relationship ID

  pagen crm delete-relationship <id>  Delete a relationship

EXAMPLES:
  # Start MCP server for Claude Desktop
  pagen mcp

  # Add a contact
  pagen crm add-contact --name "John Smith" --email "john@acme.com" --company "Acme Corp"

  # List all contacts at Acme Corp
  pagen crm list-contacts --company "Acme Corp"

  # Add a deal
  pagen crm add-deal --title "Enterprise License" --company "Acme Corp" --amount 5000000

  # List deals in negotiation stage
  pagen crm list-deals --stage negotiation

`, version)
}
