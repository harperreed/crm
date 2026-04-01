// ABOUTME: MCP server wrapper that exposes CRM operations as tools, resources, and prompts.
// ABOUTME: Bridges the storage layer to the Model Context Protocol for AI-driven CRM access.
package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/harperreed/crm/internal/storage"
)

// Server wraps an MCP server with a CRM storage backend.
type Server struct {
	server *mcp.Server
	store  storage.Storage
}

// NewServer creates an MCP server wired to the given storage backend,
// registering all CRM tools, resource templates, and prompts.
func NewServer(store storage.Storage) *Server {
	s := &Server{
		server: mcp.NewServer(
			&mcp.Implementation{Name: "crm", Version: "1.0.0"},
			nil,
		),
		store: store,
	}
	s.registerTools()
	s.registerResources()
	s.registerPrompts()
	return s
}

// Serve runs the MCP server on stdio until ctx is cancelled or the connection closes.
func (s *Server) Serve(ctx context.Context) error {
	return s.server.Run(ctx, &mcp.StdioTransport{})
}
