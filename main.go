// ABOUTME: Entry point for CRM MCP server
// ABOUTME: Initializes database and starts MCP server on stdio
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/harperreed/crm-mcp/db"
)

func main() {
	// Initialize database at XDG data directory
	dbPath := filepath.Join(xdg.DataHome, "crm", "crm.db")
	database, err := db.OpenDatabase(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	fmt.Fprintf(os.Stderr, "CRM MCP Server started. Database: %s\n", dbPath)

	// TODO: Initialize MCP server
	// TODO: Register tools
	// TODO: Start stdio transport
}
