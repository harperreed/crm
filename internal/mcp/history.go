// ABOUTME: Defines MCP tools for listing CRM history and reading full events.
// ABOUTME: Validates tool input and delegates history queries to storage.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func listHistoryTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_history",
		Description: "List change history for a CRM entity",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"entity_id": {"type": "string", "description": "Entity UUID or prefix"},
				"limit":     {"type": "integer", "description": "Maximum events (default 20, maximum 100)"}
			},
			"required": ["entity_id"]
		}`),
	}
}

func getHistoryEventTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_history_event",
		Description: "Get a full history event by ID",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"event_id": {"type": "string", "description": "History event UUID or prefix"}
			},
			"required": ["event_id"]
		}`),
	}
}

func (s *Server) handleListHistory(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		EntityID string `json:"entity_id"`
		Limit    int    `json:"limit"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}
	if strings.TrimSpace(params.EntityID) == "" {
		return errResult("entity_id is required")
	}

	summaries, err := s.store.ListHistory(params.EntityID, params.Limit)
	if err != nil {
		return errResult(fmt.Sprintf("list history: %v", err))
	}
	return jsonResult(summaries)
}

func (s *Server) handleGetHistoryEvent(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var params struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &params); err != nil {
		return errResult(fmt.Sprintf("invalid arguments: %v", err))
	}
	if strings.TrimSpace(params.EventID) == "" {
		return errResult("event_id is required")
	}

	event, err := s.store.GetHistoryEvent(params.EventID)
	if err != nil {
		return errResult(fmt.Sprintf("get history event: %v", err))
	}
	return jsonResult(event)
}
