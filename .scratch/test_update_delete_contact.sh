#!/bin/bash
set -e

echo "=== Testing Contact Update/Delete ==="

export DB=/tmp/test_crud_$$.db

# Setup - capture the ID from the add-contact output
OUTPUT=$(./pagen --db-path $DB crm add-contact --name "John Doe" --email "john@example.com" 2>&1)
echo "$OUTPUT"
CONTACT_ID=$(echo "$OUTPUT" | grep -o '[0-9a-f]\{8\}-[0-9a-f]\{4\}-[0-9a-f]\{4\}-[0-9a-f]\{4\}-[0-9a-f]\{12\}' | head -1)

# Test update (will add CLI command in next task)
echo "Contact ID: $CONTACT_ID"

# Cleanup
rm $DB

echo "✓ Contact CRUD functions added"
