// ABOUTME: Entry point for CRM MCP server and CLI
// ABOUTME: Routes to MCP server or CLI commands based on arguments
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/harperreed/pagen/cli"
	"github.com/harperreed/pagen/repository"
	"github.com/harperreed/pagen/web"
	"github.com/joho/godotenv"
)

const version = "0.2.0"

func main() {
	// Load .env file if it exists (ignore errors if not found)
	_ = godotenv.Load()

	// Global flags
	showVersion := flag.Bool("version", false, "Show version and exit")
	showHelp := flag.Bool("help", false, "Show help and exit")
	dbPath := flag.String("db", "", "Database path (default: XDG data path)")

	// Parse global flags but don't fail on unknown (for subcommands)
	_ = flag.CommandLine.Parse(os.Args[1:])

	// Handle version flag
	if *showVersion {
		fmt.Printf("pagen version %s\n", version)
		os.Exit(0)
	}

	// Handle help flag
	if *showHelp {
		printUsage()
		os.Exit(0)
	}

	// Get remaining args after flags
	args := flag.Args()

	// If no command specified, show help
	if len(args) == 0 {
		// Display ASCII art welcome banner
		fmt.Print(`
  ██████╗  █████╗  ██████╗ ███████╗███╗   ██╗
  ██╔══██╗██╔══██╗██╔════╝ ██╔════╝████╗  ██║
  ██████╔╝███████║██║  ███╗█████╗  ██╔██╗ ██║
  ██╔═══╝ ██╔══██║██║   ██║██╔══╝  ██║╚██╗██║
  ██║     ██║  ██║╚██████╔╝███████╗██║ ╚████║
  ╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝

           Your Personal CRM Agent

`)
		printUsage()
		return
	}

	// Route to top-level command
	command := args[0]
	commandArgs := args[1:]

	// Open database (used by most commands)
	openDB := func() *repository.DB {
		db, err := repository.Open(*dbPath)
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		return db
	}

	switch command {
	case "mcp":
		// MCP server
		db := openDB()
		defer func() { _ = db.Close() }()

		if err := cli.MCPCommand(db); err != nil {
			log.Fatalf("MCP server failed: %v", err)
		}

	case "crm":
		// CRM subcommands
		db := openDB()
		defer func() { _ = db.Close() }()

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
			if err := cli.AddContactCommand(db, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "list-contacts":
			if err := cli.ListContactsCommand(db, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "update-contact":
			if err := cli.UpdateContactCommand(db, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "delete-contact":
			if err := cli.DeleteContactCommand(db, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}

		// Company commands
		case "add-company":
			if err := cli.AddCompanyCommand(db, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "list-companies":
			if err := cli.ListCompaniesCommand(db, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "update-company":
			if err := cli.UpdateCompanyCommand(db, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "delete-company":
			if err := cli.DeleteCompanyCommand(db, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}

		// Deal commands
		case "add-deal":
			if err := cli.AddDealCommand(db, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "list-deals":
			if err := cli.ListDealsCommand(db, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "delete-deal":
			if err := cli.DeleteDealCommand(db, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}

		// Relationship commands
		case "update-relationship":
			if err := cli.UpdateRelationshipCommand(db, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "delete-relationship":
			if err := cli.DeleteRelationshipCommand(db, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}

		// Export commands
		case "export":
			if len(crmArgs) == 0 {
				if err := cli.ExportAllCommand(db, crmArgs); err != nil {
					log.Fatalf("Error: %v", err)
				}
			} else {
				exportType := crmArgs[0]
				exportArgs := crmArgs[1:]
				switch exportType {
				case "contacts":
					if err := cli.ExportContactsCommand(db, exportArgs); err != nil {
						log.Fatalf("Error: %v", err)
					}
				case "companies":
					if err := cli.ExportCompaniesCommand(db, exportArgs); err != nil {
						log.Fatalf("Error: %v", err)
					}
				case "deals":
					if err := cli.ExportDealsCommand(db, exportArgs); err != nil {
						log.Fatalf("Error: %v", err)
					}
				case "all":
					if err := cli.ExportAllCommand(db, exportArgs); err != nil {
						log.Fatalf("Error: %v", err)
					}
				default:
					fmt.Printf("Unknown export type: %s\n", exportType)
					fmt.Println("Available: contacts, companies, deals, all")
					os.Exit(1)
				}
			}

		default:
			fmt.Printf("Unknown crm command: %s\n\n", crmCommand)
			printUsage()
			os.Exit(1)
		}

	case "viz":
		// Visualization subcommands
		db := openDB()
		defer func() { _ = db.Close() }()

		if len(commandArgs) == 0 {
			// No subcommand = dashboard
			if err := cli.VizDashboardCommand(db, commandArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
			return
		}

		vizCommand := commandArgs[0]
		vizArgs := commandArgs[1:]

		switch vizCommand {
		case "graph":
			if len(vizArgs) == 0 {
				fmt.Println("Error: viz graph requires a type (contacts, company, or pipeline)")
				printUsage()
				os.Exit(1)
			}

			graphType := vizArgs[0]
			graphArgs := vizArgs[1:]

			switch graphType {
			case "all":
				if err := cli.VizGraphAllCommand(db, graphArgs); err != nil {
					log.Fatalf("Error: %v", err)
				}
			case "contacts":
				if err := cli.VizGraphContactsCommand(db, graphArgs); err != nil {
					log.Fatalf("Error: %v", err)
				}
			case "company":
				if err := cli.VizGraphCompanyCommand(db, graphArgs); err != nil {
					log.Fatalf("Error: %v", err)
				}
			case "pipeline":
				if err := cli.VizGraphPipelineCommand(db, graphArgs); err != nil {
					log.Fatalf("Error: %v", err)
				}
			default:
				fmt.Printf("Unknown graph type: %s\n\n", graphType)
				printUsage()
				os.Exit(1)
			}

		default:
			fmt.Printf("Unknown viz command: %s\n\n", vizCommand)
			printUsage()
			os.Exit(1)
		}

	case "web":
		port := 10666
		if len(commandArgs) > 0 && commandArgs[0] == "--port" && len(commandArgs) > 1 {
			_, _ = fmt.Sscanf(commandArgs[1], "%d", &port)
		}

		db := openDB()
		defer func() { _ = db.Close() }()

		server, err := web.NewServer(db)
		if err != nil {
			log.Fatalf("Failed to create web server: %v", err)
		}

		if err := server.Start(port); err != nil {
			log.Fatalf("Web server error: %v", err)
		}

	case "followups":
		// Follow-up tracking subcommands
		db := openDB()
		defer func() { _ = db.Close() }()

		if len(commandArgs) == 0 {
			fmt.Println("Usage: pagen followups <command>")
			fmt.Println("Commands: list, log, set-cadence, stats, digest")
			os.Exit(1)
		}

		followupCommand := commandArgs[0]
		followupArgs := commandArgs[1:]

		switch followupCommand {
		case "list":
			if err := cli.FollowupListCommand(db, followupArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "log":
			if err := cli.LogInteractionCommand(db, followupArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "set-cadence":
			if err := cli.SetCadenceCommand(db, followupArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "stats":
			if err := cli.FollowupStatsCommand(db, followupArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "digest":
			if err := cli.DigestCommand(db, followupArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		default:
			fmt.Printf("Unknown followups command: %s\n", followupCommand)
			fmt.Println("Commands: list, log, set-cadence, stats, digest")
			os.Exit(1)
		}

	case "export":
		// Export command at top level
		db := openDB()
		defer func() { _ = db.Close() }()

		if len(commandArgs) == 0 {
			if err := cli.ExportAllCommand(db, commandArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		} else {
			exportType := commandArgs[0]
			exportArgs := commandArgs[1:]
			switch exportType {
			case "contacts":
				if err := cli.ExportContactsCommand(db, exportArgs); err != nil {
					log.Fatalf("Error: %v", err)
				}
			case "companies":
				if err := cli.ExportCompaniesCommand(db, exportArgs); err != nil {
					log.Fatalf("Error: %v", err)
				}
			case "deals":
				if err := cli.ExportDealsCommand(db, exportArgs); err != nil {
					log.Fatalf("Error: %v", err)
				}
			case "all":
				if err := cli.ExportAllCommand(db, exportArgs); err != nil {
					log.Fatalf("Error: %v", err)
				}
			default:
				fmt.Printf("Unknown export type: %s\n", exportType)
				fmt.Println("Available: contacts, companies, deals, all")
				os.Exit(1)
			}
		}

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`pagen v%s - Personal Agent toolkit

USAGE:
  pagen [global flags] <command> [subcommand] [flags]

GLOBAL FLAGS:
  --version              Show version and exit
  --db <path>            Database path (default: XDG data path)

COMMANDS:
  mcp                    Start MCP server for Claude Desktop
  crm                    CRM management commands
  viz                    Visualization commands
  web                    Start web UI server
  followups              Follow-up tracking commands
  export                 Export data to YAML/Markdown

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

  pagen crm export [type]      Export data
    contacts                   Export contacts
    companies                  Export companies
    deals                      Export deals
    all                        Export all data (default)
    --format <yaml|markdown>   Output format (default: yaml)
    --output <file>            Output file (default: stdout)

VIZ COMMANDS:
  pagen viz                      Show terminal dashboard

  pagen viz graph all            Generate complete graph (all contacts, companies, deals)
    --output <file>               Output file (default: stdout)

  pagen viz graph contacts [id]  Generate contact relationship network
    --output <file>               Output file (default: stdout)
    [id]                          Optional contact ID to center graph on

  pagen viz graph company <id>   Generate company org chart
    --output <file>               Output file (default: stdout)

  pagen viz graph pipeline       Generate deal pipeline graph
    --output <file>               Output file (default: stdout)

WEB UI:
  pagen web                      Start web UI server at http://localhost:10666
    --port <port>                 Port to listen on (default: 10666)

FOLLOWUPS COMMANDS:
  pagen followups list           List contacts needing follow-up
    --overdue-only               Show only overdue contacts
    --strength <weak|medium|strong>  Filter by relationship strength
    --limit <n>                  Maximum contacts (default: 10)

  pagen followups log            Log an interaction
    --contact <id|name>          Contact ID or name (required)
    --type <type>                Interaction type (meeting/call/email/message/event)
    --notes <notes>              Notes about interaction
    --sentiment <sentiment>      Sentiment (positive/neutral/negative)

  pagen followups set-cadence    Set follow-up cadence
    --contact <id|name>          Contact ID or name (required)
    --days <n>                   Cadence in days (default: 30)
    --strength <weak|medium|strong>  Relationship strength

  pagen followups stats          Show follow-up statistics
  pagen followups digest         Generate daily follow-up digest

EXPORT COMMANDS:
  pagen export [type]            Export data
    contacts                     Export contacts
    companies                    Export companies
    deals                        Export deals
    all                          Export all data (default)
    --format <yaml|markdown>     Output format (default: yaml)
    --output <file>              Output file (default: stdout)

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

  # Export all data to YAML
  pagen export all --output backup.yaml

  # Export contacts as Markdown
  pagen export contacts --format markdown

`, version)
}
