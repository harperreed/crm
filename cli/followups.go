// ABOUTME: Follow-up tracking CLI commands
// ABOUTME: Commands for listing follow-ups, logging interactions, setting cadence
package cli

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/harperreed/pagen/charm"
)

// FollowupListCommand lists contacts needing follow-up.
func FollowupListCommand(client *charm.Client, args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	overdueOnly := fs.Bool("overdue-only", false, "Show only overdue contacts")
	strength := fs.String("strength", "", "Filter by relationship strength (weak/medium/strong)")
	limit := fs.Int("limit", 10, "Maximum number of contacts to show")
	_ = fs.Parse(args)

	followups, err := client.GetFollowupList(*limit)
	if err != nil {
		return fmt.Errorf("failed to get followup list: %w", err)
	}

	// Apply filters
	var filtered []*charm.FollowupContact
	for _, f := range followups {
		if *overdueOnly && f.PriorityScore <= 0 {
			continue
		}
		if *strength != "" && f.RelationshipStrength != *strength {
			continue
		}
		filtered = append(filtered, f)
	}

	// Print results
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tDAYS SINCE\tPRIORITY\tSTRENGTH\tEMAIL")
	_, _ = fmt.Fprintln(w, "----\t----------\t--------\t--------\t-----")

	for _, f := range filtered {
		indicator := "🟢"
		if f.DaysSinceContact > f.CadenceDays+7 {
			indicator = "🔴"
		} else if f.DaysSinceContact >= f.CadenceDays-3 {
			indicator = "🟡"
		}

		_, _ = fmt.Fprintf(w, "%s %s\t%d\t%.1f\t%s\t%s\n",
			indicator, f.Name, f.DaysSinceContact, f.PriorityScore,
			f.RelationshipStrength, f.Email)
	}

	_ = w.Flush()
	return nil
}

// FollowupStatsCommand shows follow-up statistics.
func FollowupStatsCommand(client *charm.Client, args []string) error {
	cadences, err := client.ListContactCadences()
	if err != nil {
		return fmt.Errorf("failed to get cadences: %w", err)
	}

	// Aggregate by relationship strength
	stats := make(map[string]struct {
		count   int
		avgDays float64
		total   float64
	})

	for _, cadence := range cadences {
		if cadence.LastInteractionDate == nil {
			continue
		}

		s := stats[cadence.RelationshipStrength]
		s.count++
		daysSince := time.Since(*cadence.LastInteractionDate).Hours() / 24
		s.total += daysSince
		s.avgDays = s.total / float64(s.count)
		stats[cadence.RelationshipStrength] = s
	}

	fmt.Println("NETWORK HEALTH")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Display stats for each strength level
	for _, strength := range []string{charm.StrengthWeak, charm.StrengthMedium, charm.StrengthStrong} {
		if s, exists := stats[strength]; exists {
			icon := "🟢"
			switch strength {
			case charm.StrengthWeak:
				icon = "🔴"
			case charm.StrengthMedium:
				icon = "🟡"
			}

			fmt.Printf("  %s %s relationships: %d (avg contact: %.0f days)\n",
				icon, strength, s.count, s.avgDays)
		}
	}

	return nil
}

// LogInteractionCommand logs an interaction with a contact.
func LogInteractionCommand(client *charm.Client, args []string) error {
	fs := flag.NewFlagSet("log", flag.ExitOnError)
	contactIDStr := fs.String("contact", "", "Contact ID or name (required)")
	interactionType := fs.String("type", "meeting", "Interaction type (meeting/call/email/message/event)")
	notes := fs.String("notes", "", "Notes about the interaction")
	sentiment := fs.String("sentiment", "", "Sentiment (positive/neutral/negative)")
	_ = fs.Parse(args)

	if *contactIDStr == "" {
		return fmt.Errorf("--contact is required")
	}

	// Try to parse as UUID, otherwise search by name
	var contactID uuid.UUID
	parsedID, err := uuid.Parse(*contactIDStr)
	if err == nil {
		contactID = parsedID
	} else {
		// Search by name
		contacts, err := client.ListContacts(&charm.ContactFilter{Query: *contactIDStr, Limit: 10})
		if err != nil {
			return fmt.Errorf("failed to find contact: %w", err)
		}
		if len(contacts) == 0 {
			return fmt.Errorf("no contact found matching: %s", *contactIDStr)
		}
		if len(contacts) > 1 {
			return fmt.Errorf("multiple contacts found, please use ID")
		}
		contactID = contacts[0].ID
	}

	timestamp := time.Now()
	interaction := &charm.InteractionLog{
		ContactID:       contactID,
		InteractionType: *interactionType,
		Timestamp:       timestamp,
		Notes:           *notes,
	}

	if *sentiment != "" {
		interaction.Sentiment = sentiment
	}

	if err := client.CreateInteractionLog(interaction); err != nil {
		return fmt.Errorf("failed to log interaction: %w", err)
	}

	// Update contact's last_contacted_at
	contact, err := client.GetContact(contactID)
	if err != nil {
		return fmt.Errorf("failed to get contact: %w", err)
	}
	contact.LastContactedAt = &timestamp
	if err := client.UpdateContact(contact); err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	// Update cadence
	if err := client.UpdateCadenceAfterInteraction(contactID, timestamp); err != nil {
		return fmt.Errorf("failed to update cadence: %w", err)
	}

	fmt.Printf("✓ Logged %s interaction with contact\n", *interactionType)
	return nil
}

// SetCadenceCommand sets the follow-up cadence for a contact.
func SetCadenceCommand(client *charm.Client, args []string) error {
	fs := flag.NewFlagSet("set-cadence", flag.ExitOnError)
	contactIDStr := fs.String("contact", "", "Contact ID or name (required)")
	days := fs.Int("days", 30, "Cadence in days")
	strength := fs.String("strength", "medium", "Relationship strength (weak/medium/strong)")
	_ = fs.Parse(args)

	if *contactIDStr == "" {
		return fmt.Errorf("--contact is required")
	}

	// Resolve contact ID
	var contactID uuid.UUID
	parsedID, err := uuid.Parse(*contactIDStr)
	if err == nil {
		contactID = parsedID
	} else {
		contacts, err := client.ListContacts(&charm.ContactFilter{Query: *contactIDStr, Limit: 10})
		if err != nil {
			return fmt.Errorf("failed to find contact: %w", err)
		}
		if len(contacts) == 0 {
			return fmt.Errorf("no contact found matching: %s", *contactIDStr)
		}
		contactID = contacts[0].ID
	}

	// Get or create cadence
	cadence, err := client.GetContactCadence(contactID)
	if err != nil {
		return fmt.Errorf("failed to get cadence: %w", err)
	}

	if cadence == nil {
		cadence = &charm.ContactCadence{
			ContactID: contactID,
		}
	}

	cadence.CadenceDays = *days
	cadence.RelationshipStrength = *strength

	// Compute priority score
	if cadence.LastInteractionDate != nil {
		daysSinceContact := int(time.Since(*cadence.LastInteractionDate).Hours() / 24)
		daysOverdue := daysSinceContact - cadence.CadenceDays

		if daysOverdue <= 0 {
			cadence.PriorityScore = 0.0
		} else {
			baseScore := float64(daysOverdue * 2)
			multiplier := 1.0
			switch cadence.RelationshipStrength {
			case charm.StrengthStrong:
				multiplier = 2.0
			case charm.StrengthMedium:
				multiplier = 1.5
			case charm.StrengthWeak:
				multiplier = 1.0
			}
			cadence.PriorityScore = baseScore * multiplier
		}

		// Update next followup
		next := cadence.LastInteractionDate.AddDate(0, 0, cadence.CadenceDays)
		cadence.NextFollowupDate = &next
	}

	if err := client.SaveContactCadence(cadence); err != nil {
		return fmt.Errorf("failed to save cadence: %w", err)
	}

	fmt.Printf("✓ Set cadence to %d days (%s strength)\n", *days, *strength)
	return nil
}

// DigestCommand generates a daily follow-up digest.
func DigestCommand(client *charm.Client, args []string) error {
	fs := flag.NewFlagSet("digest", flag.ExitOnError)
	format := fs.String("format", "text", "Output format (text/json/html)")
	_ = fs.Parse(args)

	followups, err := client.GetFollowupList(50)
	if err != nil {
		return fmt.Errorf("failed to get followup list: %w", err)
	}

	switch *format {
	case "text":
		return printTextDigest(followups)
	case "json":
		return printJSONDigest(followups)
	case "html":
		return printHTMLDigest(followups)
	}

	return fmt.Errorf("unsupported format: %s", *format)
}

func printTextDigest(followups []*charm.FollowupContact) error {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  FOLLOW-UPS FOR %s\n", time.Now().Format("2006-01-02"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Split into categories
	var overdue, dueSoon []*charm.FollowupContact
	for _, f := range followups {
		if f.DaysSinceContact > f.CadenceDays+7 {
			overdue = append(overdue, f)
		} else if f.DaysSinceContact >= f.CadenceDays-3 {
			dueSoon = append(dueSoon, f)
		}
	}

	if len(overdue) > 0 {
		fmt.Printf("🔴 OVERDUE (%d contacts)\n", len(overdue))
		for _, f := range overdue {
			fmt.Printf("  %-20s  %3d days  (priority: %.0f)\n", f.Name, f.DaysSinceContact, f.PriorityScore)
		}
		fmt.Println()
	}

	if len(dueSoon) > 0 {
		fmt.Printf("🟡 DUE SOON (%d contacts)\n", len(dueSoon))
		for _, f := range dueSoon {
			fmt.Printf("  %-20s  %3d days  (priority: %.0f)\n", f.Name, f.DaysSinceContact, f.PriorityScore)
		}
		fmt.Println()
	}

	return nil
}

func printJSONDigest(followups []*charm.FollowupContact) error {
	// Simple JSON output for webhook integration
	fmt.Printf("{\"date\":\"%s\",\"followups\":[", time.Now().Format("2006-01-02"))
	for i, f := range followups {
		if i > 0 {
			fmt.Print(",")
		}
		fmt.Printf("{\"name\":\"%s\",\"days\":%d,\"priority\":%.1f}",
			f.Name, f.DaysSinceContact, f.PriorityScore)
	}
	fmt.Println("]}")
	return nil
}

func printHTMLDigest(followups []*charm.FollowupContact) error {
	fmt.Println("<html><body>")
	fmt.Printf("<h1>Follow-Ups for %s</h1>\n", time.Now().Format("2006-01-02"))
	fmt.Println("<table border='1'>")
	fmt.Println("<tr><th>Name</th><th>Days Since</th><th>Priority</th></tr>")
	for _, f := range followups {
		fmt.Printf("<tr><td>%s</td><td>%d</td><td>%.1f</td></tr>\n",
			f.Name, f.DaysSinceContact, f.PriorityScore)
	}
	fmt.Println("</table>")
	fmt.Println("</body></html>")
	return nil
}
