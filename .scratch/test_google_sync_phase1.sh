#!/bin/bash
# ABOUTME: Integration test script for Google Sync Phase 1
# ABOUTME: Verifies database schema and provides OAuth testing instructions

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Database location (XDG standard)
DB_PATH="${XDG_DATA_HOME:-$HOME/.local/share}/pagen/pagen.db"

echo "=================================="
echo "Google Sync Phase 1 - Integration Test"
echo "=================================="
echo

# Check if database exists
if [ ! -f "$DB_PATH" ]; then
    echo -e "${RED}✗ FAIL${NC}: Database not found at $DB_PATH"
    echo "  Please run pagen to initialize the database first"
    exit 1
fi

echo -e "${GREEN}✓${NC} Database found at: $DB_PATH"
echo

# Function to check if a table exists
check_table() {
    local table_name=$1
    local result=$(sqlite3 "$DB_PATH" "SELECT name FROM sqlite_master WHERE type='table' AND name='$table_name';")

    if [ -z "$result" ]; then
        echo -e "${RED}✗ FAIL${NC}: Table '$table_name' does not exist"
        return 1
    else
        echo -e "${GREEN}✓ PASS${NC}: Table '$table_name' exists"
        return 0
    fi
}

# Function to show table schema
show_schema() {
    local table_name=$1
    echo -e "${BLUE}  Schema:${NC}"
    sqlite3 "$DB_PATH" ".schema $table_name" | sed 's/^/    /'
    echo
}

# Track overall success
ALL_PASSED=true

echo "Checking required tables..."
echo "----------------------------"

# Check sync_state table
if check_table "sync_state"; then
    show_schema "sync_state"
else
    ALL_PASSED=false
fi

# Check sync_log table
if check_table "sync_log"; then
    show_schema "sync_log"
else
    ALL_PASSED=false
fi

# Check suggestions table
if check_table "suggestions"; then
    show_schema "suggestions"
else
    ALL_PASSED=false
fi

# Print overall result
echo "=================================="
if [ "$ALL_PASSED" = true ]; then
    echo -e "${GREEN}✓ ALL SCHEMA CHECKS PASSED${NC}"
    echo

    # Show table row counts
    echo "Table Statistics:"
    echo "-----------------"
    sync_state_count=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM sync_state;")
    sync_log_count=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM sync_log;")
    suggestions_count=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM suggestions;")

    echo "  sync_state:   $sync_state_count rows"
    echo "  sync_log:     $sync_log_count rows"
    echo "  suggestions:  $suggestions_count rows"
    echo
else
    echo -e "${RED}✗ SOME SCHEMA CHECKS FAILED${NC}"
    echo
    exit 1
fi

# Manual OAuth Testing Instructions
echo "=================================="
echo "Manual OAuth Testing Instructions"
echo "=================================="
echo
echo -e "${YELLOW}Step 1: Configure Google OAuth${NC}"
echo "  1. Go to https://console.cloud.google.com/"
echo "  2. Create a new project or select existing"
echo "  3. Enable 'People API' and 'Contacts API'"
echo "  4. Create OAuth 2.0 credentials (Desktop app)"
echo "  5. Download credentials JSON"
echo
echo -e "${YELLOW}Step 2: Set Environment Variables${NC}"
echo "  export GOOGLE_CLIENT_ID='your-client-id'"
echo "  export GOOGLE_CLIENT_SECRET='your-client-secret'"
echo "  export GOOGLE_REDIRECT_URI='http://localhost:8080/callback'"
echo
echo -e "${YELLOW}Step 3: Test OAuth Flow${NC}"
echo "  # Start the OAuth authorization"
echo "  pagen sync google auth"
echo
echo "  # This should:"
echo "  - Open a browser to Google's authorization page"
echo "  - Start a local callback server on port 8080"
echo "  - Handle the OAuth callback"
echo "  - Store tokens in the sync_state table"
echo
echo -e "${YELLOW}Step 4: Verify Token Storage${NC}"
echo "  sqlite3 $DB_PATH \"SELECT provider, last_sync, error FROM sync_state WHERE provider='google';\""
echo
echo -e "${YELLOW}Step 5: Test Token Refresh${NC}"
echo "  # Tokens should auto-refresh when expired"
echo "  # Force a refresh test by:"
echo "  pagen sync google status"
echo
echo "=================================="
echo -e "${GREEN}Phase 1 Schema Tests: COMPLETE${NC}"
echo "=================================="
