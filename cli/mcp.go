// ABOUTME: MCP server subcommand
// ABOUTME: Starts the MCP server for Claude Desktop integration
package cli

import (
	"context"
	"log"

	"github.com/harperreed/pagen/charm"
	"github.com/harperreed/pagen/handlers"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPCommand starts the MCP server on stdio.
func MCPCommand(client *charm.Client) error {
	log.Println("Starting CRM MCP Server...")

	// Create handlers
	companyHandlers := handlers.NewCompanyHandlers(client)
	contactHandlers := handlers.NewContactHandlers(client)
	dealHandlers := handlers.NewDealHandlers(client)
	relationshipHandlers := handlers.NewRelationshipHandlers(client)
	queryHandlers := handlers.NewQueryHandlers(client)
	resourceHandlers := handlers.NewResourceHandlers(client)
	promptHandlers := handlers.NewPromptHandlers(client)
	vizHandlers := handlers.NewVizHandlers(client)
	followupHandlers := handlers.NewFollowupHandlers(client)

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
		Name:        "update_company",
		Description: "Update an existing company's information",
	}, companyHandlers.UpdateCompany)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_company",
		Description: "Delete a company (must have no active deals)",
	}, companyHandlers.DeleteCompany)

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
		Name:        "delete_contact",
		Description: "Delete a contact and all associated relationships",
	}, contactHandlers.DeleteContact)

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
		Name:        "delete_deal",
		Description: "Delete a deal and all associated notes",
	}, dealHandlers.DeleteDeal)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "link_contacts",
		Description: "Create a relationship between two contacts with optional type and context",
	}, relationshipHandlers.LinkContacts)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "find_contact_relationships",
		Description: "Find all relationships for a contact, with optional filtering by type",
	}, relationshipHandlers.FindContactRelationships)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_relationship",
		Description: "Update a relationship's type and context",
	}, relationshipHandlers.UpdateRelationship)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "remove_relationship",
		Description: "Delete a relationship between contacts",
	}, relationshipHandlers.RemoveRelationship)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_crm",
		Description: "Universal query tool for flexible filtering across all CRM entity types (contact, company, deal, relationship)",
	}, queryHandlers.QueryCRM)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "generate_graph",
		Description: "Generate GraphViz relationship/org/pipeline graphs",
	}, vizHandlers.GenerateGraph)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_followup_list",
		Description: "Get list of contacts needing follow-up, sorted by priority",
	}, followupHandlers.GetFollowupList)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "log_interaction",
		Description: "Log an interaction with a contact and update follow-up tracking",
	}, followupHandlers.LogInteraction)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_cadence",
		Description: "Set the follow-up cadence and relationship strength for a contact",
	}, followupHandlers.SetCadence)

	// Register resources
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "crm://contacts/{id}",
		Name:        "Contact",
		Description: "Individual contact by ID",
		MIMEType:    "application/json",
	}, resourceHandlers.ReadResource)

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "crm://companies/{id}",
		Name:        "Company",
		Description: "Individual company by ID",
		MIMEType:    "application/json",
	}, resourceHandlers.ReadResource)

	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "crm://deals/{id}",
		Name:        "Deal",
		Description: "Individual deal by ID",
		MIMEType:    "application/json",
	}, resourceHandlers.ReadResource)

	server.AddResource(&mcp.Resource{
		URI:         "crm://contacts",
		Name:        "All Contacts",
		Description: "List of all contacts",
		MIMEType:    "application/json",
	}, resourceHandlers.ReadResource)

	server.AddResource(&mcp.Resource{
		URI:         "crm://companies",
		Name:        "All Companies",
		Description: "List of all companies",
		MIMEType:    "application/json",
	}, resourceHandlers.ReadResource)

	server.AddResource(&mcp.Resource{
		URI:         "crm://deals",
		Name:        "All Deals",
		Description: "List of all deals",
		MIMEType:    "application/json",
	}, resourceHandlers.ReadResource)

	server.AddResource(&mcp.Resource{
		URI:         "crm://pipeline",
		Name:        "Deal Pipeline",
		Description: "Deal pipeline summary",
		MIMEType:    "application/json",
	}, resourceHandlers.ReadResource)

	// Register prompts
	server.AddPrompt(&mcp.Prompt{
		Name:        "contact-summary",
		Description: "Generate a comprehensive summary of a contact",
		Arguments: []*mcp.PromptArgument{
			{Name: "contact_id", Description: "UUID of the contact", Required: true},
		},
	}, promptHandlers.GetPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "deal-analysis",
		Description: "Analyze the current deal pipeline",
	}, promptHandlers.GetPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "relationship-map",
		Description: "Map out relationships for a contact or company",
		Arguments: []*mcp.PromptArgument{
			{Name: "entity_type", Description: "Type: 'contact' or 'company'", Required: true},
			{Name: "entity_id", Description: "UUID of the entity", Required: true},
		},
	}, promptHandlers.GetPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "follow-up-suggestions",
		Description: "Suggest contacts to follow up with",
		Arguments: []*mcp.PromptArgument{
			{Name: "days_since_contact", Description: "Days since last contact (default: 30)", Required: false},
		},
	}, promptHandlers.GetPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "company-overview",
		Description: "Complete overview of a company",
		Arguments: []*mcp.PromptArgument{
			{Name: "company_id", Description: "UUID of the company", Required: true},
		},
	}, promptHandlers.GetPrompt)

	// Run server on stdio transport
	ctx := context.Background()
	return server.Run(ctx, &mcp.StdioTransport{})
}
