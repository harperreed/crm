// ABOUTME: Entry point for CRM MCP server
// ABOUTME: Initializes database and starts MCP server on stdio
package main

import (
	"context"
	"log"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/harperreed/crm-mcp/db"
	"github.com/harperreed/crm-mcp/handlers"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Initialize database at XDG data directory
	dbPath := filepath.Join(xdg.DataHome, "crm", "crm.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	log.Printf("CRM MCP Server started. Database: %s", dbPath)

	// Create handlers
	companyHandlers := handlers.NewCompanyHandlers(database)

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

	// Run server on stdio transport
	ctx := context.Background()
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
