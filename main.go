// ABOUTME: Entry point for CRM MCP server and CLI
// ABOUTME: Routes to MCP server or CLI commands based on arguments
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/harperreed/pagen/charm"
	"github.com/harperreed/pagen/cli"
	"github.com/harperreed/pagen/tui"
	"github.com/harperreed/pagen/web"
	"github.com/joho/godotenv"
)

const version = "0.1.10"

func main() {
	// Load .env file if it exists (ignore errors if not found)
	_ = godotenv.Load()

	// Global flags
	showVersion := flag.Bool("version", false, "Show version and exit")
	showHelp := flag.Bool("help", false, "Show help and exit")
	initOnly := flag.Bool("init", false, "Initialize Charm KV and exit")

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

	// If no command specified, show welcome banner and launch TUI
	if len(args) == 0 {
		// Display ASCII art welcome banner
		fmt.Print(`
  ██████╗  █████╗  ██████╗ ███████╗███╗   ██╗
  ██╔══██╗██╔══██╗██╔════╝ ██╔════╝████╗  ██║
  ██████╔╝███████║██║  ███╗█████╗  ██╔██╗ ██║
  ██╔═══╝ ██╔══██║██║   ██║██╔══╝  ██║╚██╗██║
  ██║     ██║  ██║╚██████╔╝███████╗██║ ╚████║
  ╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═══╝

           🚀 Your Personal CRM Agent ⚡

`)

		client, err := charm.GetClient()
		if err != nil {
			log.Fatalf("Failed to initialize Charm KV: %v", err)
		}

		fmt.Println("  🔐 Loading interactive interface...")
		fmt.Println()

		tuiModel := tui.NewModel(client)
		p := tea.NewProgram(tuiModel, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			log.Fatalf("TUI error: %v", err)
		}
		return
	}

	// Route to top-level command
	command := args[0]
	commandArgs := args[1:]

	switch command {
	case "mcp":
		// MCP server uses Charm KV
		client, err := charm.GetClient()
		if err != nil {
			log.Fatalf("Failed to initialize Charm KV: %v", err)
		}

		if err := cli.MCPCommand(client); err != nil {
			log.Fatalf("MCP server failed: %v", err)
		}

	case "crm":
		// CRM subcommands - use Charm KV
		client, err := charm.GetClient()
		if err != nil {
			log.Fatalf("Failed to initialize Charm KV: %v", err)
		}

		log.Printf("CRM using Charm KV (server: %s)", client.Config().Host)

		// Handle init-only flag
		if *initOnly {
			log.Println("Charm KV initialized successfully")
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
			if err := cli.AddContactCommand(client, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "list-contacts":
			if err := cli.ListContactsCommand(client, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "update-contact":
			if err := cli.UpdateContactCommand(client, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "delete-contact":
			if err := cli.DeleteContactCommand(client, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}

		// Company commands
		case "add-company":
			if err := cli.AddCompanyCommand(client, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "list-companies":
			if err := cli.ListCompaniesCommand(client, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "update-company":
			if err := cli.UpdateCompanyCommand(client, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "delete-company":
			if err := cli.DeleteCompanyCommand(client, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}

		// Deal commands
		case "add-deal":
			if err := cli.AddDealCommand(client, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "list-deals":
			if err := cli.ListDealsCommand(client, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "delete-deal":
			if err := cli.DeleteDealCommand(client, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}

		// Relationship commands
		case "update-relationship":
			if err := cli.UpdateRelationshipCommand(client, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "delete-relationship":
			if err := cli.DeleteRelationshipCommand(client, crmArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}

		default:
			fmt.Printf("Unknown crm command: %s\n\n", crmCommand)
			printUsage()
			os.Exit(1)
		}

	case "viz":
		// Visualization subcommands - use Charm KV
		client, err := charm.GetClient()
		if err != nil {
			log.Fatalf("Failed to initialize Charm KV: %v", err)
		}

		log.Printf("Viz using Charm KV (server: %s)", client.Config().Host)

		if len(commandArgs) == 0 {
			// No subcommand = dashboard
			if err := cli.VizDashboardCommand(client, commandArgs); err != nil {
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
				if err := cli.VizGraphAllCommand(client, graphArgs); err != nil {
					log.Fatalf("Error: %v", err)
				}
			case "contacts":
				if err := cli.VizGraphContactsCommand(client, graphArgs); err != nil {
					log.Fatalf("Error: %v", err)
				}
			case "company":
				if err := cli.VizGraphCompanyCommand(client, graphArgs); err != nil {
					log.Fatalf("Error: %v", err)
				}
			case "pipeline":
				if err := cli.VizGraphPipelineCommand(client, graphArgs); err != nil {
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

		client, err := charm.GetClient()
		if err != nil {
			log.Fatalf("Failed to initialize Charm KV: %v", err)
		}

		server, err := web.NewServer(client)
		if err != nil {
			log.Fatalf("Failed to create web server: %v", err)
		}

		if err := server.Start(port); err != nil {
			log.Fatalf("Web server error: %v", err)
		}

	case "followups":
		// Follow-up tracking subcommands - use Charm KV
		client, err := charm.GetClient()
		if err != nil {
			log.Fatalf("Failed to initialize Charm KV: %v", err)
		}

		log.Printf("Followups using Charm KV (server: %s)", client.Config().Host)

		if len(commandArgs) == 0 {
			fmt.Println("Usage: pagen followups <command>")
			fmt.Println("Commands: list, log, set-cadence, stats, digest")
			os.Exit(1)
		}

		followupCommand := commandArgs[0]
		followupArgs := commandArgs[1:]

		switch followupCommand {
		case "list":
			if err := cli.FollowupListCommand(client, followupArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "log":
			if err := cli.LogInteractionCommand(client, followupArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "set-cadence":
			if err := cli.SetCadenceCommand(client, followupArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "stats":
			if err := cli.FollowupStatsCommand(client, followupArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "digest":
			if err := cli.DigestCommand(client, followupArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		default:
			fmt.Printf("Unknown followups command: %s\n", followupCommand)
			fmt.Println("Commands: list, log, set-cadence, stats, digest")
			os.Exit(1)
		}

	case "sync":
		// Charm KV sync commands
		if len(commandArgs) == 0 {
			fmt.Println("Usage: pagen sync <command>")
			fmt.Println("Commands: link, status, unlink, wipe, now, auto")
			os.Exit(1)
		}

		syncCommand := commandArgs[0]
		syncArgs := commandArgs[1:]

		switch syncCommand {
		// Charm sync commands
		case "link":
			if err := charm.SyncLinkCommand(syncArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "status":
			if err := charm.SyncStatusCommand(syncArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "unlink":
			if err := charm.SyncUnlinkCommand(syncArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "wipe":
			if err := charm.SyncWipeCommand(syncArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "now":
			if err := charm.SyncNowCommand(syncArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}
		case "auto":
			if err := charm.SetAutoSyncCommand(syncArgs); err != nil {
				log.Fatalf("Error: %v", err)
			}

		// Legacy Google sync commands (deprecated - now using Charm KV)
		case "init", "contacts", "calendar", "gmail", "reset", "daemon":
			fmt.Printf("Command 'sync %s' is deprecated.\n", syncCommand)
			fmt.Println("Google sync has been replaced by Charm KV sync.")
			fmt.Println("Available sync commands:")
			fmt.Println("  sync link   - Link this device to Charm cloud")
			fmt.Println("  sync status - Show sync status")
			fmt.Println("  sync now    - Sync immediately")
			fmt.Println("  sync auto   - Configure auto-sync")
			fmt.Println("  sync unlink - Unlink device")
			fmt.Println("  sync wipe   - Wipe all synced data")
			os.Exit(1)

		// Legacy vault commands (deprecated)
		case "vault-init", "vault-login", "vault-status", "vault-now", "vault-pending", "vault-logout", "vault-wipe", "charm-status":
			fmt.Printf("Command '%s' is deprecated. Use the new sync commands instead:\n", syncCommand)
			fmt.Println("  sync link   - Link this device")
			fmt.Println("  sync status - Show sync status")
			fmt.Println("  sync unlink - Unlink device")
			fmt.Println("  sync wipe   - Wipe all data")
			fmt.Println("  sync now    - Sync immediately")
			os.Exit(1)

		default:
			fmt.Printf("Unknown sync command: %s\n", syncCommand)
			fmt.Println("Commands: link, status, unlink, wipe, now, auto")
			os.Exit(1)
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
  --init                 Initialize Charm KV and exit (use with 'crm')

COMMANDS:
  (none)                 Launch interactive TUI (default)
  mcp                    Start MCP server for Claude Desktop
  crm                    CRM management commands
  viz                    Visualization commands
  web                    Start web UI server
  sync                   Google sync commands (contacts, calendar, gmail)

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
  pagen web                      Start web UI server at http://localhost:8080
    --port <port>                 Port to listen on (default: 8080)

SYNC COMMANDS (Charm KV Cloud Sync):
  pagen sync link                Link this device to Charm cloud
                                 Uses SSH key authentication
                                 Creates encrypted cloud backup

  pagen sync status              Show sync status and configuration

  pagen sync now                 Sync immediately
                                 Pushes local changes and pulls remote updates

  pagen sync auto <on|off>       Enable or disable auto-sync on write

  pagen sync unlink              Unlink this device from cloud sync
                                 Clears sync configuration but preserves local data

  pagen sync wipe                Remove all synced data
                                 WARNING: Deletes cloud data for this user

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

  # Link device to Charm cloud sync
  pagen sync link

  # Check sync status
  pagen sync status

  # Sync now
  pagen sync now

`, version)
}
