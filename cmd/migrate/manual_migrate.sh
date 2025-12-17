#!/bin/bash
# Manual migration script to preserve data when migrating from legacy to Office OS schema

set -e

BACKUP_DB="$1"
TARGET_DB="$2"

if [ -z "$BACKUP_DB" ] || [ -z "$TARGET_DB" ]; then
    echo "Usage: $0 <backup_db> <target_db>"
    echo "Example: $0 ~/.local/share/crm/crm.db.backup.20251202-141026 ~/.local/share/crm/crm.db"
    exit 1
fi

if [ ! -f "$BACKUP_DB" ]; then
    echo "Error: Backup database not found: $BACKUP_DB"
    exit 1
fi

echo "=== Migrating data from legacy schema to Office OS schema ==="
echo "Source: $BACKUP_DB"
echo "Target: $TARGET_DB"
echo

# Migrate Companies
echo "Migrating companies..."
sqlite3 "$BACKUP_DB" <<EOF | sqlite3 "$TARGET_DB"
.mode insert objects
SELECT
    'INSERT INTO objects (id, kind, created_at, updated_at, created_by, acl, tags, fields) VALUES (' ||
    quote(id) || ', ' ||
    quote('Company') || ', ' ||
    quote(created_at) || ', ' ||
    quote(updated_at) || ', ' ||
    quote('system') || ', ' ||
    quote('[]') || ', ' ||
    quote('[]') || ', ' ||
    quote(json_object(
        'name', name,
        'domain', COALESCE(domain, ''),
        'industry', COALESCE(industry, ''),
        'notes', COALESCE(notes, '')
    )) ||
    ');'
FROM companies;
EOF

# Migrate Contacts
echo "Migrating contacts..."
sqlite3 "$BACKUP_DB" <<EOF | sqlite3 "$TARGET_DB"
.mode insert objects
SELECT
    'INSERT INTO objects (id, kind, created_at, updated_at, created_by, acl, tags, fields) VALUES (' ||
    quote(id) || ', ' ||
    quote('Contact') || ', ' ||
    quote(created_at) || ', ' ||
    quote(updated_at) || ', ' ||
    quote('system') || ', ' ||
    quote('[]') || ', ' ||
    quote('[]') || ', ' ||
    quote(json_object(
        'name', name,
        'email', COALESCE(email, ''),
        'phone', COALESCE(phone, ''),
        'company_id', COALESCE(company_id, ''),
        'notes', COALESCE(notes, ''),
        'last_contacted_at', COALESCE(last_contacted_at, '')
    )) ||
    ');'
FROM contacts;
EOF

# Migrate Deals
echo "Migrating deals..."
sqlite3 "$BACKUP_DB" <<EOF | sqlite3 "$TARGET_DB"
.mode insert objects
SELECT
    'INSERT INTO objects (id, kind, created_at, updated_at, created_by, acl, tags, fields) VALUES (' ||
    quote(id) || ', ' ||
    quote('Deal') || ', ' ||
    quote(created_at) || ', ' ||
    quote(updated_at) || ', ' ||
    quote('system') || ', ' ||
    quote('[]') || ', ' ||
    quote('[]') || ', ' ||
    quote(json_object(
        'title', title,
        'amount', COALESCE(amount, 0),
        'currency', COALESCE(currency, 'USD'),
        'stage', COALESCE(stage, ''),
        'company_id', company_id,
        'contact_id', COALESCE(contact_id, ''),
        'expected_close_date', COALESCE(expected_close_date, ''),
        'last_activity_at', last_activity_at
    )) ||
    ');'
FROM deals;
EOF

# Migrate Relationships (if they exist in old schema)
echo "Migrating relationships..."
sqlite3 "$BACKUP_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='relationships';" | while read count; do
    if [ "$count" = "1" ]; then
        sqlite3 "$BACKUP_DB" <<EOF | sqlite3 "$TARGET_DB"
.mode insert relationships
SELECT
    'INSERT INTO relationships (id, source_id, target_id, type, metadata, created_at, updated_at) VALUES (' ||
    quote(COALESCE(id, hex(randomblob(16)))) || ', ' ||
    quote(contact_id_1) || ', ' ||
    quote(contact_id_2) || ', ' ||
    quote(COALESCE(relationship_type, 'knows')) || ', ' ||
    quote(json_object('context', COALESCE(context, ''))) || ', ' ||
    quote(created_at) || ', ' ||
    quote(updated_at) ||
    ');'
FROM relationships;
EOF
    else
        echo "No legacy relationships table found, skipping..."
    fi
done

# Migrate interaction_log (if exists)
echo "Migrating interaction log..."
sqlite3 "$BACKUP_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='interaction_log';" | while read count; do
    if [ "$count" = "1" ]; then
        sqlite3 "$TARGET_DB" "DELETE FROM interaction_log;"
        sqlite3 "$BACKUP_DB" ".dump interaction_log" | sqlite3 "$TARGET_DB"
    fi
done

# Migrate contact_cadence (if exists)
echo "Migrating contact cadence..."
sqlite3 "$BACKUP_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='contact_cadence';" | while read count; do
    if [ "$count" = "1" ]; then
        sqlite3 "$TARGET_DB" "DELETE FROM contact_cadence;"
        sqlite3 "$BACKUP_DB" ".dump contact_cadence" | sqlite3 "$TARGET_DB"
    fi
done

# Migrate sync_state (if exists)
echo "Migrating sync state..."
sqlite3 "$BACKUP_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sync_state';" | while read count; do
    if [ "$count" = "1" ]; then
        sqlite3 "$TARGET_DB" "DELETE FROM sync_state;"
        sqlite3 "$BACKUP_DB" ".dump sync_state" | sqlite3 "$TARGET_DB"
    fi
done

# Migrate sync_log (if exists)
echo "Migrating sync log..."
sqlite3 "$BACKUP_DB" "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sync_log';" | while read count; do
    if [ "$count" = "1" ]; then
        sqlite3 "$TARGET_DB" "DELETE FROM sync_log;"
        sqlite3 "$BACKUP_DB" ".dump sync_log" | sqlite3 "$TARGET_DB"
    fi
done

echo
echo "=== Migration complete! ==="
echo "Checking counts..."
echo -n "Companies: "
sqlite3 "$TARGET_DB" "SELECT COUNT(*) FROM objects WHERE kind='Company';"
echo -n "Contacts: "
sqlite3 "$TARGET_DB" "SELECT COUNT(*) FROM objects WHERE kind='Contact';"
echo -n "Deals: "
sqlite3 "$TARGET_DB" "SELECT COUNT(*) FROM objects WHERE kind='Deal';"
echo -n "Relationships: "
sqlite3 "$TARGET_DB" "SELECT COUNT(*) FROM relationships;"
