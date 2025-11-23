#!/bin/bash
set -e

echo "=== Web UI Manual Test Instructions ==="
echo ""
echo "This script creates test data. Then launch the web server manually."
echo ""

export DB=/tmp/test_web_$$.db

# Create test data
./pagen --db-path $DB crm add-company --name "Acme Corp" --domain "acme.com" --industry "Software"
./pagen --db-path $DB crm add-company --name "TechStart Inc" --domain "techstart.io" --industry "SaaS"
./pagen --db-path $DB crm add-contact --name "Alice Johnson" --email "alice@acme.com" --company "Acme Corp"
./pagen --db-path $DB crm add-contact --name "Bob Smith" --email "bob@techstart.io" --company "TechStart Inc"
./pagen --db-path $DB crm add-contact --name "Carol White" --email "carol@acme.com" --company "Acme Corp"
./pagen --db-path $DB crm add-deal --title "Enterprise Deal" --company "Acme Corp" --amount 500000 --stage "negotiation"
./pagen --db-path $DB crm add-deal --title "Startup Package" --company "TechStart Inc" --amount 50000 --stage "prospecting"
./pagen --db-path $DB crm add-deal --title "Premium License" --company "Acme Corp" --amount 250000 --stage "proposal"

echo ""
echo "Test data created in: $DB"
echo ""
echo "To launch web UI, run:"
echo "  ./pagen --db-path $DB web"
echo ""
echo "Then visit: http://localhost:8080"
echo ""
echo "Test checklist:"
echo "  [ ] Dashboard shows correct stats (3 contacts, 2 companies, 3 deals)"
echo "  [ ] Dashboard shows pipeline overview with bars"
echo "  [ ] Contacts page shows all 3 contacts"
echo "  [ ] Clicking 'View' on contact shows detail panel"
echo "  [ ] Search on contacts page filters results"
echo "  [ ] Companies page shows all 2 companies"
echo "  [ ] Company detail shows associated contacts"
echo "  [ ] Deals page shows all 3 deals"
echo "  [ ] Deal detail shows notes (if any)"
echo "  [ ] Graphs page allows selecting graph type"
echo "  [ ] Generating contact graph shows DOT output"
echo "  [ ] Generating pipeline graph shows DOT output"
echo "  [ ] All navigation links work"
echo ""
echo "Cleanup: rm $DB"
