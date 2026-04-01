// ABOUTME: MCP resource templates for reading CRM contacts and companies by ID.
// ABOUTME: Returns entity data plus associated relationships as JSON.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerResources adds resource templates for contacts and companies.
func (s *Server) registerResources() {
	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "crm://contacts/{id}",
			Name:        "CRM Contact",
			Description: "A contact with its relationships",
			MIMEType:    "application/json",
		},
		s.handleContactResource,
	)

	s.server.AddResourceTemplate(
		&mcp.ResourceTemplate{
			URITemplate: "crm://companies/{id}",
			Name:        "CRM Company",
			Description: "A company with its relationships",
			MIMEType:    "application/json",
		},
		s.handleCompanyResource,
	)
}

// extractID pulls the trailing path segment from a crm:// URI.
func extractID(uri, prefix string) (string, error) {
	if !strings.HasPrefix(uri, prefix) {
		return "", fmt.Errorf("unexpected URI prefix: %s", uri)
	}
	id := strings.TrimPrefix(uri, prefix)
	if id == "" {
		return "", fmt.Errorf("missing id in URI: %s", uri)
	}
	return id, nil
}

func (s *Server) handleContactResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	idStr, err := extractID(req.Params.URI, "crm://contacts/")
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	contact, err := s.resolveContact(idStr)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	rels, err := s.store.ListRelationships(contact.ID)
	if err != nil {
		return nil, fmt.Errorf("list relationships: %w", err)
	}

	result := map[string]any{
		"contact":       contact,
		"relationships": rels,
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal contact resource: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func (s *Server) handleCompanyResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	idStr, err := extractID(req.Params.URI, "crm://companies/")
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	company, err := s.store.GetCompany(id)
	if err != nil {
		return nil, mcp.ResourceNotFoundError(req.Params.URI)
	}

	rels, err := s.store.ListRelationships(company.ID)
	if err != nil {
		return nil, fmt.Errorf("list relationships: %w", err)
	}

	result := map[string]any{
		"company":       company,
		"relationships": rels,
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal company resource: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}
