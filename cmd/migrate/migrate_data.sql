-- Attach the backup database
ATTACH DATABASE ? AS backup;

-- Migrate Companies
INSERT INTO main.objects (id, kind, created_at, updated_at, created_by, acl, tags, fields)
SELECT
    id,
    'Company',
    created_at,
    updated_at,
    'system',
    '[]',
    '[]',
    json_object(
        'name', name,
        'domain', COALESCE(domain, ''),
        'industry', COALESCE(industry, ''),
        'notes', COALESCE(notes, '')
    )
FROM backup.companies;

-- Migrate Contacts
INSERT INTO main.objects (id, kind, created_at, updated_at, created_by, acl, tags, fields)
SELECT
    id,
    'Contact',
    created_at,
    updated_at,
    'system',
    '[]',
    '[]',
    json_object(
        'name', name,
        'email', COALESCE(email, ''),
        'phone', COALESCE(phone, ''),
        'company_id', COALESCE(company_id, ''),
        'notes', COALESCE(notes, ''),
        'last_contacted_at', COALESCE(last_contacted_at, '')
    )
FROM backup.contacts;

-- Migrate Deals
INSERT INTO main.objects (id, kind, created_at, updated_at, created_by, acl, tags, fields)
SELECT
    id,
    'Deal',
    created_at,
    updated_at,
    'system',
    '[]',
    '[]',
    json_object(
        'title', title,
        'amount', COALESCE(amount, 0),
        'currency', COALESCE(currency, 'USD'),
        'stage', COALESCE(stage, ''),
        'company_id', company_id,
        'contact_id', COALESCE(contact_id, ''),
        'expected_close_date', COALESCE(expected_close_date, ''),
        'last_activity_at', last_activity_at
    )
FROM backup.deals
WHERE EXISTS (SELECT 1 FROM sqlite_master WHERE type='table' AND name='deals');

-- Detach backup database
DETACH DATABASE backup;
