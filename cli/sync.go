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
	initial := fs.Bool("initial", false, "Full import (not incremental)")
	_ = fs.Parse(args)

	// TODO: Real Google API integration
	// For now, just placeholder

	fmt.Println("Syncing Google Contacts...")
	fmt.Println("  → Fetching contacts...")

	// Placeholder - will implement real API in next iteration
	fmt.Println("  ✓ Google API integration pending")
	fmt.Println("\nTo complete setup:")
	fmt.Println("1. Enable Google People API in Cloud Console")
	fmt.Println("2. Run 'pagen sync init' to authenticate")
	fmt.Println("3. Re-run 'pagen sync contacts'")

	if *initial {
		fmt.Println("\nNote: --initial flag will be used for full import when API is integrated")
	}

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
