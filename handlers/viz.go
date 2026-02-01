// ABOUTME: GraphViz visualization MCP handlers
// ABOUTME: Provides generate_graph tool for agents
package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/repository"
	"github.com/harperreed/pagen/viz"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type VizHandlers struct {
	db *repository.DB
}

func NewVizHandlers(db *repository.DB) *VizHandlers {
	return &VizHandlers{db: db}
}

type GenerateGraphInput struct {
	Type     string `json:"type" jsonschema:"Graph type: contacts, company, or pipeline"`
	EntityID string `json:"entity_id,omitempty" jsonschema:"UUID of entity (required for company, optional for contacts)"`
}

type GenerateGraphOutput struct {
	GraphType string `json:"graph_type"`
	DOTSource string `json:"dot_source"`
	NodeCount int    `json:"node_count"`
	EdgeCount int    `json:"edge_count"`
}

func (h *VizHandlers) GenerateGraph(_ context.Context, request *mcp.CallToolRequest, input GenerateGraphInput) (*mcp.CallToolResult, GenerateGraphOutput, error) {
	if input.Type == "" {
		return nil, GenerateGraphOutput{}, fmt.Errorf("type is required")
	}

	generator := viz.NewGraphGenerator(h.db)
	var dot string
	var err error

	switch input.Type {
	case "contacts":
		var contactID *uuid.UUID
		if input.EntityID != "" {
			var id uuid.UUID
			id, err = uuid.Parse(input.EntityID)
			if err != nil {
				return nil, GenerateGraphOutput{}, fmt.Errorf("invalid entity_id: %w", err)
			}
			contactID = &id
		}
		dot, err = generator.GenerateContactGraph(contactID)

	case "company":
		if input.EntityID == "" {
			return nil, GenerateGraphOutput{}, fmt.Errorf("entity_id required for company graph")
		}
		var companyID uuid.UUID
		companyID, err = uuid.Parse(input.EntityID)
		if err != nil {
			return nil, GenerateGraphOutput{}, fmt.Errorf("invalid entity_id: %w", err)
		}
		dot, err = generator.GenerateCompanyGraph(companyID)

	case "pipeline":
		dot, err = generator.GeneratePipelineGraph()

	default:
		return nil, GenerateGraphOutput{}, fmt.Errorf("unknown graph type: %s (valid types: contacts, company, pipeline)", input.Type)
	}

	if err != nil {
		return nil, GenerateGraphOutput{}, fmt.Errorf("failed to generate graph: %w", err)
	}

	// Count nodes and edges for stats
	nodeCount := strings.Count(dot, "[label=")
	edgeCount := strings.Count(dot, "->")

	return nil, GenerateGraphOutput{
		GraphType: input.Type,
		DOTSource: dot,
		NodeCount: nodeCount,
		EdgeCount: edgeCount,
	}, nil
}
