#!/bin/bash
set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  PAGEN CRM - MCP Tool Verification"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Count tool registrations in cli/mcp.go
echo "Counting MCP tool registrations..."
TOOL_COUNT=$(grep -c 'mcp.AddTool(server, &mcp.Tool{' cli/mcp.go || echo 0)

echo "Found $TOOL_COUNT tool registrations in cli/mcp.go"
echo ""

# List all tools
echo "Registered MCP tools:"
grep -A 1 'Name:' cli/mcp.go | grep 'Name:' | sed 's/.*Name: "\([^"]*\)".*/  - \1/'
echo ""

# Expected tools
EXPECTED_TOOLS=(
    "add_contact"
    "find_contacts"
    "update_contact"
    "delete_contact"
    "log_contact_interaction"
    "add_company"
    "find_companies"
    "update_company"
    "delete_company"
    "create_deal"
    "update_deal"
    "delete_deal"
    "add_deal_note"
    "link_contacts"
    "find_contact_relationships"
    "update_relationship"
    "remove_relationship"
    "query_crm"
    "generate_graph"
)

EXPECTED_COUNT=${#EXPECTED_TOOLS[@]}

echo "Expected tool count: $EXPECTED_COUNT"
echo "Actual tool count: $TOOL_COUNT"
echo ""

if [ "$TOOL_COUNT" -eq "$EXPECTED_COUNT" ]; then
    echo "✓ Tool count matches expected"
else
    echo "✗ Tool count mismatch!"
    echo ""
    echo "Expected tools:"
    for tool in "${EXPECTED_TOOLS[@]}"; do
        echo "  - $tool"
    done
    exit 1
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✓ MCP TOOL VERIFICATION PASSED"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
