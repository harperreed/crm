// ABOUTME: Gmail importer for high-signal email interactions
// ABOUTME: Imports replied-to and starred emails as interactions with contacts
package sync

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/api/gmail/v1"

	"github.com/harperreed/pagen/db"
	"github.com/harperreed/pagen/models"
)

const (
	gmailService          = "gmail"
	maxGmailResults       = 500 // Gmail API max per page
	defaultImportDays     = 30  // Last 30 days for initial sync
	skipReasonAutomated   = "automated sender"
	skipReasonGroup       = "group email"
	skipReasonCalendar    = "calendar invite"
	skipReasonAutoSubject = "auto-generated subject"
)

// ImportGmail fetches and imports high-signal emails from Gmail
func ImportGmail(database *sql.DB, client *gmail.Service, initial bool) error {
	// Update sync state to 'syncing'
	fmt.Println("Syncing Gmail...")
	if err := db.UpdateSyncStatus(database, gmailService, "syncing", nil); err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	// Get user's email address
	profile, err := client.Users.GetProfile("me").Do()
	if err != nil {
		errMsg := fmt.Sprintf("failed to get user profile: %v", err)
		_ = db.UpdateSyncStatus(database, gmailService, "error", &errMsg)
		return fmt.Errorf("failed to get user profile: %w", err)
	}
	userEmail := profile.EmailAddress

	// Load existing contacts for matching
	allContacts, err := db.FindContacts(database, "", nil, 20000)
	if err != nil {
		errMsg := err.Error()
		_ = db.UpdateSyncStatus(database, gmailService, "error", &errMsg)
		return fmt.Errorf("failed to load existing contacts: %w", err)
	}

	// Create contact matcher
	matcher := NewContactMatcher(allContacts)

	// Build query based on initial vs incremental
	var query string
	if initial {
		// Last 30 days of high-signal emails
		since := time.Now().AddDate(0, 0, -defaultImportDays)
		query = BuildHighSignalQuery(userEmail, since)
		fmt.Printf("  → Initial sync (last %d days, high-signal only)...\n", defaultImportDays)
	} else {
		// TODO: Use historyId for incremental sync
		// For now, just fetch last 7 days
		since := time.Now().AddDate(0, 0, -7)
		query = BuildHighSignalQuery(userEmail, since)
		fmt.Printf("  → Incremental sync (last 7 days)...\n")
	}

	// Fetch messages
	totalFetched := 0
	totalProcessed := 0
	newContacts := 0
	skipCounts := make(map[string]int)
	pageToken := ""

	for {
		// Build request
		call := client.Users.Messages.List("me").
			Q(query).
			MaxResults(maxGmailResults)

		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		// Fetch page
		response, err := call.Do()
		if err != nil {
			errMsg := fmt.Sprintf("failed to fetch messages: %v", err)
			_ = db.UpdateSyncStatus(database, gmailService, "error", &errMsg)
			return fmt.Errorf("failed to fetch messages: %w", err)
		}

		// Handle empty response
		if response == nil || response.Messages == nil {
			break
		}

		totalFetched += len(response.Messages)

		// Process each message
		for _, msgRef := range response.Messages {
			// Get full message details
			message, err := client.Users.Messages.Get("me", msgRef.Id).
				Format("metadata").
				MetadataHeaders("From", "To", "Cc", "Subject", "Date").
				Do()

			if err != nil {
				fmt.Printf("  ✗ Failed to fetch message %s: %v\n", msgRef.Id, err)
				continue
			}

			// Check if already imported
			exists, err := db.CheckSyncLogExists(database, gmailService, message.Id)
			if err != nil {
				fmt.Printf("  ✗ Failed to check sync log: %v\n", err)
				continue
			}
			if exists {
				skipCounts["already imported"]++
				continue
			}

			// Apply high-signal filtering
			isHighSignal, reason := IsHighSignalEmail(message, userEmail)
			if !isHighSignal {
				skipCounts[reason]++
				continue
			}

			// Extract email data
			headers := parseHeaders(message.Payload)
			from := headers["From"]
			subject := headers["Subject"]
			dateStr := headers["Date"]

			// Parse date
			emailDate, err := parseEmailDate(dateStr)
			if err != nil {
				fmt.Printf("  ✗ Failed to parse date for message: %v\n", err)
				continue
			}

			// Extract sender contact
			senderName, senderEmail, senderDomain := ExtractEmailAddress(from)
			if senderEmail == "" || senderEmail == userEmail {
				// Skip if no sender or email to self
				continue
			}

			// Find or create contact
			contactID, isNew, err := findOrCreateEmailContact(database, matcher, senderName, senderEmail, senderDomain)
			if err != nil {
				fmt.Printf("  ✗ Failed to create contact for %s: %v\n", senderEmail, err)
				continue
			}

			if isNew {
				newContacts++
			}

			// Log interaction
			interaction := &models.InteractionLog{
				ContactID:       contactID,
				InteractionType: models.InteractionEmail,
				Timestamp:       emailDate,
				Notes:           subject, // Just subject, not body
				Metadata: fmt.Sprintf(`{"message_id": "%s", "thread_id": "%s"}`,
					message.Id, message.ThreadId),
			}

			if err := db.LogInteraction(database, interaction); err != nil {
				fmt.Printf("  ✗ Failed to log interaction: %v\n", err)
				continue
			}

			// Record in sync log
			syncLogID := uuid.New().String()
			metadata := fmt.Sprintf(`{"subject": %s}`, jsonEscape(subject))
			if err := db.CreateSyncLog(database, syncLogID, gmailService, message.Id, "interaction", interaction.ID.String(), metadata); err != nil {
				fmt.Printf("  ✗ Failed to create sync log: %v\n", err)
				continue
			}

			totalProcessed++
		}

		// Check for next page
		pageToken = response.NextPageToken
		if pageToken == "" {
			break
		}

		// Show progress
		if totalProcessed > 0 {
			fmt.Printf("  → Processed %d emails so far...\n", totalProcessed)
		}
	}

	// Update sync state to 'idle' on success
	if err := db.UpdateSyncStatus(database, gmailService, "idle", nil); err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	// Print summary
	fmt.Printf("\n✓ Fetched %d emails from Gmail\n", totalFetched)

	// Print skip summary
	if len(skipCounts) > 0 {
		totalSkipped := 0
		for _, count := range skipCounts {
			totalSkipped += count
		}

		for reason, count := range skipCounts {
			fmt.Printf("  ✓ Skipped %d %s\n", count, reason)
		}
	}

	if totalProcessed == 0 {
		fmt.Println("  ✓ No new emails to import (all up to date)")
	} else {
		fmt.Printf("\n  → Processed %d high-signal emails\n", totalProcessed)
		if newContacts > 0 {
			fmt.Printf("  ✓ Created %d new contacts from email addresses\n", newContacts)
		}
		fmt.Printf("  ✓ Logged %d email interactions\n", totalProcessed)
	}

	return nil
}

// findOrCreateEmailContact finds existing contact by email or creates new one
func findOrCreateEmailContact(database *sql.DB, matcher *ContactMatcher, name, email, domain string) (uuid.UUID, bool, error) {
	// Try to find existing contact
	existing, found := matcher.FindMatch(email, name)
	if found {
		return existing.ID, false, nil
	}

	// Create new contact
	contact := &models.Contact{
		Name:  name,
		Email: email,
	}

	// If name is empty, use email username as name
	if contact.Name == "" && email != "" {
		parts := strings.Split(email, "@")
		if len(parts) > 0 {
			contact.Name = parts[0]
		}
	}

	// Try to find or create company from domain
	if domain != "" && !isCommonEmailDomain(domain) {
		company, err := findOrCreateCompanyFromDomain(database, domain)
		if err == nil && company != nil {
			contact.CompanyID = &company.ID
		}
	}

	// Create contact
	if err := db.CreateContact(database, contact); err != nil {
		return uuid.Nil, false, err
	}

	// Add to matcher
	matcher.AddContact(contact)

	return contact.ID, true, nil
}

// findOrCreateCompanyFromDomain creates company from email domain
func findOrCreateCompanyFromDomain(database *sql.DB, domain string) (*models.Company, error) {
	// Capitalize domain as company name
	companyName := capitalizeCompanyName(domain)

	// Try to find existing
	company, err := db.FindCompanyByName(database, companyName)
	if err != nil {
		return nil, err
	}
	if company != nil {
		return company, nil
	}

	// Create new company
	newCompany := &models.Company{
		Name:   companyName,
		Domain: domain,
	}

	if err := db.CreateCompany(database, newCompany); err != nil {
		// If creation failed, try finding again (race condition)
		company, findErr := db.FindCompanyByName(database, companyName)
		if findErr != nil {
			return nil, err
		}
		if company != nil {
			return company, nil
		}
		return nil, err
	}

	return newCompany, nil
}

// isCommonEmailDomain checks if domain is a common email provider (not company-specific)
func isCommonEmailDomain(domain string) bool {
	commonDomains := []string{
		"gmail.com",
		"googlemail.com",
		"yahoo.com",
		"hotmail.com",
		"outlook.com",
		"live.com",
		"msn.com",
		"icloud.com",
		"me.com",
		"mac.com",
		"aol.com",
		"protonmail.com",
		"pm.me",
	}

	lowerDomain := strings.ToLower(domain)
	for _, common := range commonDomains {
		if lowerDomain == common {
			return true
		}
	}

	return false
}

// capitalizeCompanyName converts domain to company name
func capitalizeCompanyName(domain string) string {
	// Remove common TLDs
	name := strings.TrimSuffix(domain, ".com")
	name = strings.TrimSuffix(name, ".org")
	name = strings.TrimSuffix(name, ".net")
	name = strings.TrimSuffix(name, ".io")

	// Capitalize first letter of each word (split by dot or dash)
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '.' || r == '-'
	})

	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}

	return strings.Join(parts, " ")
}

// parseEmailDate parses RFC 2822 email date
func parseEmailDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Now(), nil
	}

	// Strip trailing timezone name like "(UTC)" or "(PST)"
	if idx := strings.Index(dateStr, " ("); idx > 0 {
		dateStr = dateStr[:idx]
	}

	// Try RFC 2822 formats
	formats := []string{
		time.RFC1123Z,                    // "Mon, 02 Jan 2006 15:04:05 -0700"
		"Mon, 2 Jan 2006 15:04:05 -0700", // Single digit day with timezone
		time.RFC1123,                     // "Mon, 02 Jan 2006 15:04:05 MST"
		"Mon, 2 Jan 2006 15:04:05 MST",   // Single digit day without numeric timezone
		time.RFC822Z,                     // "02 Jan 06 15:04 -0700"
		time.RFC822,                      // "02 Jan 06 15:04 MST"
		time.RFC3339,                     // "2006-01-02T15:04:05Z07:00"
	}

	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			return t, nil
		}
	}

	return time.Now(), fmt.Errorf("failed to parse date: %s", dateStr)
}

// jsonEscape escapes a string for safe JSON embedding
func jsonEscape(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}
