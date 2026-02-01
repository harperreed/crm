// ABOUTME: Complete graph generation combining all entities
// ABOUTME: Generates comprehensive visualization of entire CRM database
package viz

import (
	"bytes"
	"context"
	"fmt"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/harperreed/pagen/repository"
)

// GenerateCompleteGraph creates a comprehensive graph with all contacts, companies, and deals.
func (g *GraphGenerator) GenerateCompleteGraph() (string, error) {
	ctx := context.Background()
	gv, err := graphviz.New(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create graphviz: %w", err)
	}
	defer func() {
		if err := gv.Close(); err != nil {
			fmt.Printf("Error closing graphviz: %v\n", err)
		}
	}()

	graph, err := gv.Graph()
	if err != nil {
		return "", fmt.Errorf("failed to create graph: %w", err)
	}
	defer func() {
		if err := graph.Close(); err != nil {
			fmt.Printf("Error closing graph: %v\n", err)
		}
	}()

	graph.SetLabel("Complete CRM Graph")

	// Get all entities
	contacts, err := g.db.ListContacts(&repository.ContactFilter{Limit: 10000})
	if err != nil {
		return "", fmt.Errorf("failed to fetch contacts: %w", err)
	}

	companies, err := g.db.ListCompanies(&repository.CompanyFilter{Limit: 10000})
	if err != nil {
		return "", fmt.Errorf("failed to fetch companies: %w", err)
	}

	deals, err := g.db.ListDeals(&repository.DealFilter{Limit: 10000})
	if err != nil {
		return "", fmt.Errorf("failed to fetch deals: %w", err)
	}

	// Create nodes for companies
	companyNodes := make(map[string]*cgraph.Node)
	for _, company := range companies {
		node, err := graph.CreateNodeByName(fmt.Sprintf("company_%s", company.ID.String()[:8]))
		if err != nil {
			return "", fmt.Errorf("failed to create company node: %w", err)
		}
		node.SetLabel(fmt.Sprintf("%s\n(Company)", company.Name))
		node.SetShape("box")
		node.SetStyle("filled")
		node.SetFillColor("lightblue")
		companyNodes[company.ID.String()] = node
	}

	// Create nodes for contacts
	contactNodes := make(map[string]*cgraph.Node)
	for _, contact := range contacts {
		node, err := graph.CreateNodeByName(fmt.Sprintf("contact_%s", contact.ID.String()[:8]))
		if err != nil {
			return "", fmt.Errorf("failed to create contact node: %w", err)
		}
		node.SetLabel(fmt.Sprintf("%s\n%s", contact.Name, contact.Email))
		node.SetShape("ellipse")
		node.SetStyle("filled")
		node.SetFillColor("lightgreen")
		contactNodes[contact.ID.String()] = node

		// Link contact to company
		if contact.CompanyID != nil {
			if companyNode, ok := companyNodes[contact.CompanyID.String()]; ok {
				edge, err := graph.CreateEdgeByName("works_at", node, companyNode)
				if err != nil {
					return "", fmt.Errorf("failed to create edge: %w", err)
				}
				edge.SetLabel("works at")
				edge.SetStyle("dashed")
			}
		}
	}

	// Create nodes for deals
	for _, deal := range deals {
		node, err := graph.CreateNodeByName(fmt.Sprintf("deal_%s", deal.ID.String()[:8]))
		if err != nil {
			return "", fmt.Errorf("failed to create deal node: %w", err)
		}
		amountK := deal.Amount / 100000
		node.SetLabel(fmt.Sprintf("%s\n$%dK\n(%s)", deal.Title, amountK, deal.Stage))
		node.SetShape("diamond")
		node.SetStyle("filled")
		node.SetFillColor("lightyellow")

		// Link deal to company
		if companyNode, ok := companyNodes[deal.CompanyID.String()]; ok {
			edge, err := graph.CreateEdgeByName("deal_with", companyNode, node)
			if err != nil {
				return "", fmt.Errorf("failed to create edge: %w", err)
			}
			edge.SetLabel("deal")
		}

		// Link deal to contact
		if deal.ContactID != nil {
			if contactNode, ok := contactNodes[deal.ContactID.String()]; ok {
				edge, err := graph.CreateEdgeByName("contact_for", contactNode, node)
				if err != nil {
					return "", fmt.Errorf("failed to create edge: %w", err)
				}
				edge.SetLabel("contact")
				edge.SetStyle("dotted")
			}
		}
	}

	// Get relationships between contacts
	relationships, err := g.db.ListRelationships(nil)
	if err != nil {
		return "", fmt.Errorf("failed to fetch relationships: %w", err)
	}

	for _, rel := range relationships {
		node1, ok1 := contactNodes[rel.ContactID1.String()]
		node2, ok2 := contactNodes[rel.ContactID2.String()]
		if ok1 && ok2 {
			edge, err := graph.CreateEdgeByName(rel.RelationshipType, node1, node2)
			if err != nil {
				return "", fmt.Errorf("failed to create relationship edge: %w", err)
			}
			edge.SetLabel(rel.RelationshipType)
			edge.SetDir("none") // Undirected edge for relationships
		}
	}

	var buf bytes.Buffer
	if err := gv.Render(ctx, graph, graphviz.XDOT, &buf); err != nil {
		return "", fmt.Errorf("failed to render graph: %w", err)
	}

	return buf.String(), nil
}
