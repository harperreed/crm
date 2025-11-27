#!/bin/bash
# .scratch/test_followups.sh
# Integration test for follow-up system

set -e

# Setup
TEST_DB=$(mktemp -t test-followups-XXXXXX.db)
echo "Using test database: $TEST_DB"

# Build
go build -o pagen-test

# Create test data
./pagen-test --db-path "$TEST_DB" crm add-company --name "TestCorp" --industry "Software"
./pagen-test --db-path "$TEST_DB" crm add-contact --name "Alice Test" --email "alice@test.com" --company "TestCorp"
./pagen-test --db-path "$TEST_DB" crm add-contact --name "Bob Test" --email "bob@test.com" --company "TestCorp"

# Test followup commands
echo "Testing followup list..."
./pagen-test --db-path "$TEST_DB" followups list

echo "Testing followup stats..."
./pagen-test --db-path "$TEST_DB" followups stats

echo "Testing set cadence..."
ALICE_ID=$(./pagen-test --db-path "$TEST_DB" crm list-contacts | grep "Alice Test" | awk '{print $1}')
./pagen-test --db-path "$TEST_DB" followups set-cadence --contact "$ALICE_ID" --days 14 --strength strong

echo "Testing log interaction..."
./pagen-test --db-path "$TEST_DB" followups log --contact "$ALICE_ID" --type meeting --notes "Test meeting"

echo "Testing digest..."
./pagen-test --db-path "$TEST_DB" followups digest
./pagen-test --db-path "$TEST_DB" followups digest --format json

# Cleanup
rm -f pagen-test "$TEST_DB"

echo "✓ All integration tests passed!"
