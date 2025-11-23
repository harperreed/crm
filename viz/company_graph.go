package viz

import (
	"bytes"
	"context"
	"fmt"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/google/uuid"
	"github.com/harperreed/pagen/db"
)

func (g *GraphGenerator) GenerateCompanyGraph(companyID uuid.UUID) (string, error) {
	ctx := context.Background()
	gv, err := graphviz.New(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create graphviz instance: %w", err)
	}
	defer gv.Close()

	graph, err := gv.Graph()
	if err != nil {
		return "", fmt.Errorf("failed to create graph: %w", err)
	}
	defer graph.Close()

	graph.SetLayout("dot")

	// Get company
	company, err := db.GetCompany(g.db, companyID)
	if err != nil {
		return "", fmt.Errorf("company not found: %w", err)
	}

	// Create root node
	rootNode, err := graph.CreateNodeByName(company.Name)
	if err != nil {
		return "", fmt.Errorf("failed to create root node: %w", err)
	}
	rootNode.SetShape(cgraph.BoxShape)
	rootNode.SetStyle(cgraph.FilledNodeStyle)
	rootNode.SetFillColor("lightblue")

	// Get all contacts at company
	contacts, err := db.FindContacts(g.db, "", &companyID, 1000)
	if err != nil {
		return "", fmt.Errorf("failed to fetch contacts: %w", err)
	}

	// Create nodes for contacts
	contactNodes := make(map[string]*cgraph.Node)
	for _, contact := range contacts {
		node, err := graph.CreateNodeByName(contact.Name)
		if err != nil {
			continue
		}
		contactNodes[contact.ID.String()] = node
		// Link to company
		_, _ = graph.CreateEdgeByName("", rootNode, node)
	}

	// Add relationships between contacts
	for _, contact := range contacts {
		relationships, _ := db.FindContactRelationships(g.db, contact.ID, "")
		for _, rel := range relationships {
			otherID := rel.ContactID2
			if rel.ContactID1 != contact.ID {
				otherID = rel.ContactID1
			}

			if otherNode, exists := contactNodes[otherID.String()]; exists {
				edge, err := graph.CreateEdgeByName("", contactNodes[contact.ID.String()], otherNode)
				if err != nil {
					continue
				}
				edge.SetStyle(cgraph.DashedEdgeStyle)
				if rel.RelationshipType != "" {
					edge.SetLabel(rel.RelationshipType)
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := gv.Render(ctx, graph, graphviz.XDOT, &buf); err != nil {
		return "", fmt.Errorf("failed to render graph: %w", err)
	}

	return buf.String(), nil
}
