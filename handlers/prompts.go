// ABOUTME: MCP prompt handlers for reusable CRM workflow templates
// ABOUTME: Provides standardized prompts for common CRM operations
package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/db"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PromptHandlers struct {
	db *sql.DB
}

func NewPromptHandlers(database *sql.DB) *PromptHandlers {
	return &PromptHandlers{db: database}
}

// GetPrompt generates the prompt message based on the template
func (h *PromptHandlers) GetPrompt(ctx context.Context, request *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	name := request.Params.Name
	arguments := request.Params.Arguments
	switch name {
	case "contact-summary":
		return h.getContactSummaryPrompt(arguments)
	case "deal-analysis":
		return h.getDealAnalysisPrompt(arguments)
	case "relationship-map":
		return h.getRelationshipMapPrompt(arguments)
	case "follow-up-suggestions":
		return h.getFollowUpSuggestionsPrompt(arguments)
	case "company-overview":
		return h.getCompanyOverviewPrompt(arguments)
	default:
		return nil, fmt.Errorf("unknown prompt: %s", name)
	}
}

func (h *PromptHandlers) getContactSummaryPrompt(args map[string]string) (*mcp.GetPromptResult, error) {
	contactIDStr, ok := args["contact_id"]
	if !ok {
		return nil, fmt.Errorf("contact_id is required")
	}

	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid contact_id: %w", err)
	}

	contact, err := db.GetContact(h.db, contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contact: %w", err)
	}

	// Get company info if available
	var companyName string
	if contact.CompanyID != nil {
		company, err := db.GetCompany(h.db, *contact.CompanyID)
		if err == nil {
			companyName = company.Name
		}
	}

	// Get relationships
	relationships, err := db.FindContactRelationships(h.db, contactID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch relationships: %w", err)
	}

	// Build the prompt
	var promptText strings.Builder
	promptText.WriteString("Please provide a comprehensive summary of this contact:\n\n")
	promptText.WriteString(fmt.Sprintf("Name: %s\n", contact.Name))
	if contact.Email != "" {
		promptText.WriteString(fmt.Sprintf("Email: %s\n", contact.Email))
	}
	if contact.Phone != "" {
		promptText.WriteString(fmt.Sprintf("Phone: %s\n", contact.Phone))
	}
	if companyName != "" {
		promptText.WriteString(fmt.Sprintf("Company: %s\n", companyName))
	}
	if contact.LastContactedAt != nil {
		promptText.WriteString(fmt.Sprintf("Last Contacted: %s\n", contact.LastContactedAt.Format("2006-01-02")))
	}
	if len(relationships) > 0 {
		promptText.WriteString(fmt.Sprintf("\nRelationships: %d connections\n", len(relationships)))
	}
	if contact.Notes != "" {
		promptText.WriteString(fmt.Sprintf("\nNotes: %s\n", contact.Notes))
	}

	promptText.WriteString("\nPlease analyze this contact and provide:")
	promptText.WriteString("\n1. A brief summary of their role and background")
	promptText.WriteString("\n2. Recommendations for next steps or follow-up actions")
	promptText.WriteString("\n3. Any patterns or insights from their interaction history")

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Summary for contact: %s", contact.Name),
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{

					Text: promptText.String(),
				},
			},
		},
	}, nil
}

func (h *PromptHandlers) getDealAnalysisPrompt(args map[string]string) (*mcp.GetPromptResult, error) {
	// Get all deals
	deals, err := db.FindDeals(h.db, "", nil, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deals: %w", err)
	}

	// Calculate pipeline metrics
	stageCount := make(map[string]int)
	stageValue := make(map[string]int64)
	totalValue := int64(0)

	for _, deal := range deals {
		stage := deal.Stage
		if stage == "" {
			stage = "unknown"
		}
		stageCount[stage]++
		stageValue[stage] += deal.Amount
		totalValue += deal.Amount
	}

	// Build the prompt
	var promptText strings.Builder
	promptText.WriteString("Please analyze the current deal pipeline:\n\n")
	promptText.WriteString(fmt.Sprintf("Total Deals: %d\n", len(deals)))
	promptText.WriteString(fmt.Sprintf("Total Value: $%d\n\n", totalValue/100))
	promptText.WriteString("Pipeline by Stage:\n")
	for stage, count := range stageCount {
		promptText.WriteString(fmt.Sprintf("  - %s: %d deals, $%d\n", stage, count, stageValue[stage]/100))
	}

	promptText.WriteString("\nPlease provide:")
	promptText.WriteString("\n1. Analysis of pipeline health and distribution")
	promptText.WriteString("\n2. Recommendations for deals that may need attention")
	promptText.WriteString("\n3. Suggestions for improving conversion rates")

	return &mcp.GetPromptResult{
		Description: "Deal pipeline analysis",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{

					Text: promptText.String(),
				},
			},
		},
	}, nil
}

func (h *PromptHandlers) getRelationshipMapPrompt(args map[string]string) (*mcp.GetPromptResult, error) {
	entityType, ok := args["entity_type"]
	if !ok {
		return nil, fmt.Errorf("entity_type is required")
	}

	entityIDStr, ok := args["entity_id"]
	if !ok {
		return nil, fmt.Errorf("entity_id is required")
	}

	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid entity_id: %w", err)
	}

	var promptText strings.Builder

	switch entityType {
	case "contact":
		contact, err := db.GetContact(h.db, entityID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch contact: %w", err)
		}

		relationships, err := db.FindContactRelationships(h.db, entityID, "")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch relationships: %w", err)
		}

		promptText.WriteString(fmt.Sprintf("Map out the relationship network for: %s\n\n", contact.Name))
		promptText.WriteString(fmt.Sprintf("Direct Connections: %d\n\n", len(relationships)))

		if len(relationships) > 0 {
			promptText.WriteString("Relationships:\n")
			for _, rel := range relationships {
				// Get the other contact's name
				otherContactID := rel.ContactID2
				if rel.ContactID1 != entityID {
					otherContactID = rel.ContactID1
				}
				otherContact, err := db.GetContact(h.db, otherContactID)
				contactName := "Unknown"
				if err == nil && otherContact != nil {
					contactName = otherContact.Name
				}

				if rel.RelationshipType != "" {
					promptText.WriteString(fmt.Sprintf("  - %s (%s)\n", contactName, rel.RelationshipType))
				} else {
					promptText.WriteString(fmt.Sprintf("  - %s\n", contactName))
				}
			}
		}
	case "company":
		company, err := db.GetCompany(h.db, entityID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch company: %w", err)
		}

		contacts, err := db.FindContacts(h.db, "", &entityID, 1000)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch company contacts: %w", err)
		}

		promptText.WriteString(fmt.Sprintf("Map out the relationship network for company: %s\n\n", company.Name))
		promptText.WriteString(fmt.Sprintf("Contacts at company: %d\n\n", len(contacts)))

		if len(contacts) > 0 {
			promptText.WriteString("People:\n")
			for _, contact := range contacts {
				promptText.WriteString(fmt.Sprintf("  - %s", contact.Name))
				if contact.Email != "" {
					promptText.WriteString(fmt.Sprintf(" (%s)", contact.Email))
				}
				promptText.WriteString("\n")
			}
		}
	default:
		return nil, fmt.Errorf("invalid entity_type: must be 'contact' or 'company'")
	}

	promptText.WriteString("\nPlease visualize and analyze this relationship network.")

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Relationship map for %s", entityType),
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{

					Text: promptText.String(),
				},
			},
		},
	}, nil
}

func (h *PromptHandlers) getFollowUpSuggestionsPrompt(args map[string]string) (*mcp.GetPromptResult, error) {
	// Get all contacts
	contacts, err := db.FindContacts(h.db, "", nil, 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contacts: %w", err)
	}

	// Default to 30 days
	daysThreshold := 30
	if d, ok := args["days_since_contact"]; ok {
		if parsedDays, err := fmt.Sscanf(d, "%d", &daysThreshold); err != nil || parsedDays != 1 {
			daysThreshold = 30
		}
	}

	var promptText strings.Builder
	promptText.WriteString(fmt.Sprintf("Contacts that may need follow-up (no contact in %d+ days):\n\n", daysThreshold))

	now := time.Now()
	threshold := time.Duration(daysThreshold) * 24 * time.Hour

	count := 0
	for _, contact := range contacts {
		// Show contacts with no recent interaction or old last_contacted_at
		if contact.LastContactedAt == nil {
			promptText.WriteString(fmt.Sprintf("- %s (never contacted)\n", contact.Name))
			count++
		} else if now.Sub(*contact.LastContactedAt) > threshold {
			daysSince := int(now.Sub(*contact.LastContactedAt).Hours() / 24)
			promptText.WriteString(fmt.Sprintf("- %s (last contacted %d days ago)\n", contact.Name, daysSince))
			count++
		}
	}

	if count == 0 {
		promptText.WriteString("All contacts have been contacted recently.\n")
	}

	promptText.WriteString("\nPlease:")
	promptText.WriteString("\n1. Prioritize which contacts to reach out to first")
	promptText.WriteString("\n2. Suggest personalized outreach approaches for each")
	promptText.WriteString("\n3. Identify any patterns in follow-up gaps")

	return &mcp.GetPromptResult{
		Description: "Follow-up suggestions for contacts",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{

					Text: promptText.String(),
				},
			},
		},
	}, nil
}

func (h *PromptHandlers) getCompanyOverviewPrompt(args map[string]string) (*mcp.GetPromptResult, error) {
	companyIDStr, ok := args["company_id"]
	if !ok {
		return nil, fmt.Errorf("company_id is required")
	}

	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid company_id: %w", err)
	}

	company, err := db.GetCompany(h.db, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch company: %w", err)
	}

	contacts, err := db.FindContacts(h.db, "", &companyID, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contacts: %w", err)
	}

	deals, err := db.FindDeals(h.db, "", &companyID, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deals: %w", err)
	}

	var promptText strings.Builder
	promptText.WriteString(fmt.Sprintf("Complete overview of: %s\n\n", company.Name))

	if company.Industry != "" {
		promptText.WriteString(fmt.Sprintf("Industry: %s\n", company.Industry))
	}
	if company.Domain != "" {
		promptText.WriteString(fmt.Sprintf("Domain: %s\n", company.Domain))
	}

	promptText.WriteString(fmt.Sprintf("\nContacts: %d people\n", len(contacts)))
	for _, contact := range contacts {
		promptText.WriteString(fmt.Sprintf("  - %s", contact.Name))
		if contact.Email != "" {
			promptText.WriteString(fmt.Sprintf(" <%s>", contact.Email))
		}
		promptText.WriteString("\n")
	}

	promptText.WriteString(fmt.Sprintf("\nDeals: %d active\n", len(deals)))
	totalValue := int64(0)
	for _, deal := range deals {
		promptText.WriteString(fmt.Sprintf("  - %s: $%d (%s)\n", deal.Title, deal.Amount/100, deal.Stage))
		totalValue += deal.Amount
	}
	if len(deals) > 0 {
		promptText.WriteString(fmt.Sprintf("\nTotal Deal Value: $%d\n", totalValue/100))
	}

	if company.Notes != "" {
		promptText.WriteString(fmt.Sprintf("\nNotes: %s\n", company.Notes))
	}

	promptText.WriteString("\nPlease provide:")
	promptText.WriteString("\n1. A summary of the relationship with this company")
	promptText.WriteString("\n2. Key opportunities or risks")
	promptText.WriteString("\n3. Recommended next actions")

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Overview of %s", company.Name),
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{

					Text: promptText.String(),
				},
			},
		},
	}, nil
}
