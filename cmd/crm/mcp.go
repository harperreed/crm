// ABOUTME: CLI command to start the MCP server for AI-driven CRM access.
// ABOUTME: Runs the server on stdio transport, bridging storage to the MCP protocol.
package main

import (
	mcpserver "github.com/harperreed/crm/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server (stdio transport)",
	Long:  "Start an MCP server that exposes CRM tools, resources, and prompts over stdio.",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mcpserver.NewServer(store)
		return server.Serve(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
