# Google Sync Phase 3: Gmail Integration (High-Signal Only)

**Date:** 2025-11-28
**Goal:** Import high-signal email interactions without drowning in noise

## Philosophy: Less is More

**Problem:** Full email import = 10,000+ emails = noise overload
**Solution:** Import ONLY emails that indicate real human interaction

## What We Import (High-Signal Only)

### ✅ Include:
1. **Sent emails with replies** - You sent it, they responded = real interaction
2. **Received emails you replied to** - They sent it, you responded = real interaction
3. **Starred/Important emails** - You marked it = it matters
4. **Recent emails (last 30 days)** - Fresh conversations

### ❌ Exclude:
1. Newsletters/marketing (from:noreply, unsubscribe links)
2. Automated emails (no-reply, donotreply)
3. Group emails (5+ recipients)
4. Your own emails to yourself
5. Calendar invites (already have those from Calendar sync)
6. Spam/Trash

## Data We Extract

**For each qualifying email:**
- Sender email → create/match contact
- Recipients → identify relationship strength
- Subject line → notes field (NOT full body)
- Date → interaction timestamp
- Thread ID → group related emails
- Message ID → prevent duplicates in sync_log

**We do NOT store:**
- Full email body (privacy + storage)
- Attachments
- HTML formatting
- Email headers (except From/To/Date/Subject)

## Implementation Plan

### Task 1: Gmail API Client
**File:** `sync/gmail_client.go`

```go
func NewGmailClient(token *oauth2.Token) (*gmail.Service, error)
```

Same pattern as Calendar and People clients.

### Task 2: Email Filter
**File:** `sync/gmail_filters.go`

```go
// BuildHighSignalQuery builds Gmail API query for high-signal emails
func BuildHighSignalQuery(userEmail string, since time.Time) string {
    // Returns: "from:me is:replied OR to:me is:replied OR is:starred"
    // Excludes: newsletters, automated, large groups
}

// IsHighSignalEmail checks if email meets our criteria
func IsHighSignalEmail(message *gmail.Message, userEmail string) bool {
    // Check sender not noreply/automated
    // Check recipient count < 5
    // Check has real subject (not auto-generated)
}
```

### Task 3: Gmail Importer
**File:** `sync/gmail_importer.go`

```go
func ImportGmail(database *sql.DB, client *gmail.Service, initial bool) error {
    // 1. Get user email
    // 2. Build query for high-signal emails
    // 3. Fetch with pagination
    // 4. For each message:
    //    - Check sync_log (skip if already imported)
    //    - Extract sender/recipients
    //    - Create/update contacts
    //    - Log interaction with subject as notes
    //    - Record in sync_log
}

func extractEmailContact(address string) (*GoogleContact, error) {
    // Parse "John Doe <john@example.com>" format
    // Extract domain for company detection
}
```

### Task 4: CLI Command
**File:** `cli/sync.go`

```go
func SyncGmailCommand(database *sql.DB, args []string) error {
    fs := flag.NewFlagSet("gmail", flag.ExitOnError)
    initial := fs.Bool("initial", false, "Import last 30 days")
    _ = fs.Parse(args)

    // Load token
    // Create Gmail client
    // Run import
}
```

### Task 5: Main Router
**File:** `main.go`

Add "gmail" case to sync switch.

## API Integration

### Gmail API Query

**Initial sync (last 30 days):**
```
query: "(from:me is:replied) OR (to:me is:replied) OR is:starred"
after: 2025-10-28
maxResults: 500
```

**Incremental sync:**
Use `historyId` from last sync to get only new messages since then.

### OAuth Scope

Already have this from Phase 1:
```
https://www.googleapis.com/auth/gmail.readonly
```

## Database Operations

### Interaction Logging

```go
interaction := &models.InteractionLog{
    ContactID:       contactID,
    InteractionType: models.InteractionEmail,
    Timestamp:       emailDate,
    Notes:           subject,  // Just subject, not body
    Metadata:        `{"thread_id": "...", "message_id": "..."}`,
}
```

### Sync Log

```go
syncLog := &models.SyncLog{
    SourceService: "gmail",
    SourceID:      message.Id,
    EntityType:    "interaction",
    EntityID:      interaction.ID,
}
```

## CLI Commands

```bash
pagen sync gmail              # Incremental sync
pagen sync gmail --initial    # Last 30 days
pagen sync                    # Syncs contacts + calendar + gmail
```

## User Experience

### Initial Sync

```
$ pagen sync gmail --initial

Syncing Gmail (last 30 days, high-signal only)...
  → Fetching replied and starred emails...
  ✓ Fetched 234 emails
  ✓ Skipped 89 automated/newsletter emails
  ✓ Skipped 12 group emails (5+ recipients)

  → Processing 133 high-signal emails...
  ✓ Created 23 new contacts from email addresses
  ✓ Logged 133 email interactions
  ✓ Updated cadence for 87 contacts

Summary:
  Emails imported: 133
  Contacts created: 23
  Interactions logged: 133

History ID saved. Next sync will be incremental.
```

### Incremental Sync

```
$ pagen sync gmail

Syncing Gmail...
  ✓ 7 new emails since last sync
  ✓ Logged 7 interactions

All up to date!
```

## Error Handling

### Rate Limits

Gmail API: 250 quota units per user per second
- 1 request = 5 units
- Can fetch ~50 emails/second

**Mitigation:** Batch requests, add small delay if hitting limits.

### Invalid Email Addresses

Some emails have malformed sender addresses.

**Fix:** Try parsing, fall back to domain-only if name extraction fails.

## Testing Strategy

### Unit Tests

1. Email filtering logic (what gets imported)
2. Email address parsing
3. Subject extraction
4. Duplicate detection

### Integration Tests

1. Import emails from test account
2. Verify interactions created
3. Verify contacts matched correctly
4. Verify no duplicates on re-sync

## Success Criteria

- [ ] Only replied-to and starred emails imported
- [ ] No newsletters/automated emails imported
- [ ] No group emails (5+ recipients) imported
- [ ] Email subject stored in interaction notes
- [ ] Email body NOT stored anywhere
- [ ] Contacts auto-created from email addresses
- [ ] Incremental sync uses historyId
- [ ] No duplicate interactions on re-sync
- [ ] Sync completes in <30 seconds for 200 emails

## Data Volume Estimates

**Conservative filtering:**
- Average user: ~50 emails/day
- High-signal only: ~5 emails/day (90% reduction)
- 30-day initial: ~150 emails
- Weekly incremental: ~35 emails

This is manageable and high-value.

## Privacy Notes

- Only subject line stored (not body)
- No attachments downloaded
- Email content never leaves Google servers
- Interaction metadata shows "email" type but not content

## Open Questions

1. **Should we extract company from email domain?**
   - Example: john@acme.com → Acme Corp
   - Decision: YES, but only for unknown domains (use DNS/whois)

2. **Should we track email thread depth?**
   - Example: 1 = first message, 5 = fifth reply
   - Decision: Store thread_id in metadata, can analyze later

3. **Should we differentiate sent vs received?**
   - Both are interactions, but direction might matter
   - Decision: Store in metadata, interaction_type stays "email"

## Dependencies

**New packages:**
```
google.golang.org/api/gmail/v1  # Gmail API
```

All other dependencies already added in Phase 1 & 2.

## Out of Scope (Future)

- Full-text email search
- Email body analysis for deal detection
- Attachment metadata extraction
- Email labels → tags mapping
- Sentiment analysis
- Auto-reply detection (mark as lower signal)

These can be added later without changing the core design.
