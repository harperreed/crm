// ABOUTME: MCP prompt templates for guided CRM workflows.
// ABOUTME: Provides add-contact, relationship-mapping, and cross-entity search prompts.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerPrompts adds workflow prompts to the MCP server.
func (s *Server) registerPrompts() {
	s.server.AddPrompt(
		&mcp.Prompt{
			Name:        "add-contact-workflow",
			Description: "Guided workflow for adding a new contact",
			Arguments: []*mcp.PromptArgument{{
				Name:        "name",
				Description: "Name of the contact to add",
				Required:    true,
			}},
		},
		s.handleAddContactPrompt,
	)

	s.server.AddPrompt(
		&mcp.Prompt{
			Name:        "relationship-mapping",
			Description: "Explore an entity's connections and relationships",
			Arguments: []*mcp.PromptArgument{{
				Name:        "entity_id",
				Description: "UUID or prefix of the entity to explore",
				Required:    true,
			}},
		},
		s.handleRelationshipMappingPrompt,
	)

	s.server.AddPrompt(
		&mcp.Prompt{
			Name:        "crm-search",
			Description: "Search across all CRM entities",
			Arguments: []*mcp.PromptArgument{{
				Name:        "query",
				Description: "Search query string",
				Required:    true,
			}},
		},
		s.handleCRMSearchPrompt,
	)
}

func (s *Server) handleAddContactPrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	name := req.Params.Arguments["name"]
	if name == "" {
		name = "unknown"
	}

	return &mcp.GetPromptResult{
		Description: "Add a new contact to the CRM",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(
						"I want to add a new contact named %q to the CRM.\n\n"+
							"Please use the add_contact tool. Ask me for any additional details like:\n"+
							"- Email address\n"+
							"- Phone number\n"+
							"- Company affiliation\n"+
							"- Tags for categorization\n"+
							"- Any custom fields\n\n"+
							"After adding the contact, suggest creating relationships with existing contacts or companies if relevant.",
						name,
					),
				},
			},
		},
	}, nil
}

func (s *Server) handleRelationshipMappingPrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	entityID := req.Params.Arguments["entity_id"]
	if entityID == "" {
		return &mcp.GetPromptResult{
			Description: "Explore entity relationships",
			Messages: []*mcp.PromptMessage{{
				Role:    "user",
				Content: &mcp.TextContent{Text: "Please provide an entity ID to explore relationships for."},
			}},
		}, nil
	}

	return &mcp.GetPromptResult{
		Description: "Explore entity relationships",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(
						"I want to explore the relationships for entity %q.\n\n"+
							"Please:\n"+
							"1. Look up the entity using get_contact or get_company\n"+
							"2. List all its relationships\n"+
							"3. For each related entity, fetch its details\n"+
							"4. Present a summary of the relationship network\n"+
							"5. Suggest any missing or potential relationships",
						entityID,
					),
				},
			},
		},
	}, nil
}

func (s *Server) handleCRMSearchPrompt(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	query := req.Params.Arguments["query"]
	if query == "" {
		return &mcp.GetPromptResult{
			Description: "Search across CRM entities",
			Messages: []*mcp.PromptMessage{{
				Role:    "user",
				Content: &mcp.TextContent{Text: "Please provide a search query."},
			}},
		}, nil
	}

	// Run the search and include results in the prompt context.
	results, err := s.store.Search(query)
	if err != nil {
		return nil, fmt.Errorf("search CRM: %w", err)
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		data = []byte("{}")
	}

	return &mcp.GetPromptResult{
		Description: "Search across CRM entities",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(
						"I searched the CRM for %q. Here are the results:\n\n```json\n%s\n```\n\n"+
							"Please analyze these results and:\n"+
							"1. Summarize what was found\n"+
							"2. Highlight any relationships between the results\n"+
							"3. Suggest follow-up actions",
						query, string(data),
					),
				},
			},
		},
	}, nil
}
