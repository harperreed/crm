// ABOUTME: MCP server subcommand
// ABOUTME: Starts the MCP server for Claude Desktop integration
package cli

import (
	"context"
	"database/sql"
	"log"

	"github.com/harperreed/pagen/handlers"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPCommand starts the MCP server on stdio
func MCPCommand(db *sql.DB) error {
	log.Println("Starting CRM MCP Server...")

	// Create handlers
	companyHandlers := handlers.NewCompanyHandlers(db)
	contactHandlers := handlers.NewContactHandlers(db)
	dealHandlers := handlers.NewDealHandlers(db)
	relationshipHandlers := handlers.NewRelationshipHandlers(db)
	queryHandlers := handlers.NewQueryHandlers(db)

	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "crm",
		Version: "0.1.0",
	}, nil)

	// Register tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_company",
		Description: "Add a new company to the CRM",
	}, companyHandlers.AddCompany)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_companies",
		Description: "Search for companies by name or domain",
	}, companyHandlers.FindCompanies)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_contact",
		Description: "Add a new contact to the CRM",
	}, contactHandlers.AddContact)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_contacts",
		Description: "Search for contacts by name, email, or company",
	}, contactHandlers.FindContacts)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_contact",
		Description: "Update an existing contact's information",
	}, contactHandlers.UpdateContact)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "log_contact_interaction",
		Description: "Log an interaction with a contact and update last contacted timestamp",
	}, contactHandlers.LogContactInteraction)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_deal",
		Description: "Create a new deal in the CRM with company and optional contact",
	}, dealHandlers.CreateDeal)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_deal",
		Description: "Update an existing deal's information including stage and amount",
	}, dealHandlers.UpdateDeal)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_deal_note",
		Description: "Add a note to a deal and update activity timestamps",
	}, dealHandlers.AddDealNote)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "link_contacts",
		Description: "Create a relationship between two contacts with optional type and context",
	}, relationshipHandlers.LinkContacts)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_contact_relationships",
		Description: "Find all relationships for a contact, with optional filtering by type",
	}, relationshipHandlers.FindContactRelationships)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "remove_relationship",
		Description: "Delete a relationship between contacts",
	}, relationshipHandlers.RemoveRelationship)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_crm",
		Description: "Universal query tool for flexible filtering across all CRM entity types (contact, company, deal, relationship)",
	}, queryHandlers.QueryCRM)

	// Run server on stdio transport
	ctx := context.Background()
	return server.Run(ctx, &mcp.StdioTransport{})
}
