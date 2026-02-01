package viz

import (
	"bytes"
	"context"
	"fmt"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/google/uuid"
	"github.com/harperreed/pagen/repository"
)

func (g *GraphGenerator) GenerateContactGraph(contactID *uuid.UUID) (string, error) {
	ctx := context.Background()
	gv, err := graphviz.New(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create graphviz instance: %w", err)
	}
	defer func() { _ = gv.Close() }()

	graph, err := gv.Graph()
	if err != nil {
		return "", fmt.Errorf("failed to create graph: %w", err)
	}
	defer func() { _ = graph.Close() }()

	graph.SetLayout("neato")
	graph.SetRankDir(cgraph.LRRank)

	// If contactID provided, show that contact's network
	// Otherwise show all contacts and relationships
	var relationships []*repository.Relationship
	if contactID != nil {
		relationships, err = g.db.ListRelationshipsForContact(*contactID)
	} else {
		// Get all relationships
		relationships, err = g.db.ListRelationships(nil)
	}

	if err != nil {
		return "", fmt.Errorf("failed to fetch relationships: %w", err)
	}

	// Create nodes for all unique contacts
	// Use denormalized names from relationships where possible
	nodes := make(map[string]*cgraph.Node)
	for _, rel := range relationships {
		id1 := rel.ContactID1.String()
		id2 := rel.ContactID2.String()

		if _, exists := nodes[id1]; !exists {
			name := rel.Contact1Name
			if name == "" {
				name = "Unknown"
			}
			nodes[id1], _ = graph.CreateNodeByName(name)
		}

		if _, exists := nodes[id2]; !exists {
			name := rel.Contact2Name
			if name == "" {
				name = "Unknown"
			}
			nodes[id2], _ = graph.CreateNodeByName(name)
		}

		// Create edge
		edge, _ := graph.CreateEdgeByName("", nodes[id1], nodes[id2])
		if rel.RelationshipType != "" {
			edge.SetLabel(rel.RelationshipType)
		}
	}

	// Generate DOT source
	var buf bytes.Buffer
	if err := gv.Render(ctx, graph, graphviz.XDOT, &buf); err != nil {
		return "", fmt.Errorf("failed to render graph: %w", err)
	}

	return buf.String(), nil
}
