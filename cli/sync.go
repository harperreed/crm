// ABOUTME: Google sync CLI commands
// ABOUTME: Handles OAuth setup, sync operations, and suggestion review
package cli

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/harperreed/pagen/sync"
	"golang.org/x/oauth2"
)

// SyncInitCommand handles OAuth setup
func SyncInitCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	_ = fs.Parse(args)

	ctx := context.Background()

	config, err := sync.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get OAuth config: %w", err)
	}

	// Start local server for OAuth callback
	callbackChan := make(chan *oauth2.Token)
	errChan := make(chan error)

	http.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			return
		}

		token, err := config.Exchange(ctx, code)
		if err != nil {
			errChan <- fmt.Errorf("failed to exchange code: %w", err)
			return
		}

		callbackChan <- token
		_, _ = fmt.Fprintf(w, "Authorization successful! You can close this window.")
	})

	server := &http.Server{Addr: ":8080"}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Generate auth URL
	authURL := config.AuthCodeURL("state", oauth2.AccessTypeOffline)

	fmt.Println("Opening browser for Google OAuth...")
	fmt.Printf("\nIf browser doesn't open, visit this URL:\n%s\n\n", authURL)

	// Try to open browser
	_ = openBrowser(authURL)

	// Wait for callback or error
	select {
	case token := <-callbackChan:
		_ = server.Shutdown(ctx)

		if err := sync.SaveToken(token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Printf("\n✓ Authenticated successfully\n")
		fmt.Printf("✓ Tokens saved to %s\n\n", sync.TokenPath())
		fmt.Println("Ready to sync! Run 'pagen sync contacts' to import contacts.")

		return nil

	case err := <-errChan:
		_ = server.Shutdown(ctx)
		return fmt.Errorf("OAuth flow failed: %w", err)
	}
}

// SyncContactsCommand syncs Google Contacts
func SyncContactsCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("contacts", flag.ExitOnError)
	_ = fs.Parse(args)

	// Load OAuth token
	token, err := sync.LoadToken()
	if err != nil {
		return fmt.Errorf("no authentication token found. Run 'pagen sync init' first: %w", err)
	}

	// Create People API client
	client, err := sync.NewPeopleClient(token)
	if err != nil {
		return fmt.Errorf("failed to create People API client: %w", err)
	}

	// Import contacts
	if err := sync.ImportContacts(database, client); err != nil {
		return fmt.Errorf("contacts sync failed: %w", err)
	}

	return nil
}

// SyncCalendarCommand syncs Google Calendar events
func SyncCalendarCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("calendar", flag.ExitOnError)
	initial := fs.Bool("initial", false, "Full import (last 6 months)")
	_ = fs.Parse(args)

	// Load OAuth token
	token, err := sync.LoadToken()
	if err != nil {
		return fmt.Errorf("no authentication token found. Run 'pagen sync init' first: %w", err)
	}

	// Create Calendar client
	client, err := sync.NewCalendarClient(token)
	if err != nil {
		return fmt.Errorf("failed to create Calendar client: %w", err)
	}

	// Import calendar events
	if err := sync.ImportCalendar(database, client, *initial); err != nil {
		return fmt.Errorf("calendar sync failed: %w", err)
	}

	return nil
}

// SyncGmailCommand syncs Gmail emails
func SyncGmailCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("gmail", flag.ExitOnError)
	initial := fs.Bool("initial", false, "Import last 30 days")
	_ = fs.Parse(args)

	// Load OAuth token
	token, err := sync.LoadToken()
	if err != nil {
		return fmt.Errorf("no authentication token found. Run 'pagen sync init' first: %w", err)
	}

	// Create Gmail client
	client, err := sync.NewGmailClient(token)
	if err != nil {
		return fmt.Errorf("failed to create Gmail client: %w", err)
	}

	// Import emails
	if err := sync.ImportGmail(database, client, *initial); err != nil {
		return fmt.Errorf("gmail sync failed: %w", err)
	}

	return nil
}

// SyncResetCommand resets a stuck sync state
func SyncResetCommand(database *sql.DB, args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: pagen sync reset <service>")
		fmt.Println("\nAvailable services:")
		fmt.Println("  calendar - Reset calendar sync state")
		fmt.Println("  contacts - Reset contacts sync state")
		fmt.Println("  all      - Reset all sync states")
		return nil
	}

	service := args[0]

	if service == "all" {
		// Reset all services
		_, err := database.Exec(`UPDATE sync_state SET status='idle', last_sync_time=datetime('now')`)
		if err != nil {
			return fmt.Errorf("failed to reset sync states: %w", err)
		}
		fmt.Println("✓ Reset all sync states to 'idle'")
		return nil
	}

	// Reset specific service
	result, err := database.Exec(`UPDATE sync_state SET status='idle', last_sync_time=datetime('now') WHERE service=?`, service)
	if err != nil {
		return fmt.Errorf("failed to reset sync state for %s: %w", service, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		fmt.Printf("⚠ No sync state found for service '%s'\n", service)
		return nil
	}

	fmt.Printf("✓ Reset %s sync state to 'idle'\n", service)
	fmt.Println("\nNext sync will be incremental from now.")
	return nil
}

// openBrowser attempts to open URL in default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	command := exec.Command(cmd, args...)
	return command.Start()
}
