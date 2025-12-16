// ABOUTME: Vault E2E sync CLI commands for encrypted CRM synchronization
// ABOUTME: Handles device initialization, authentication, sync operations, and status monitoring
package cli

import (
	"context"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/harperreed/sweet/vault"

	"github.com/harperreed/pagen/sync"
	"golang.org/x/term"
)

// SyncVaultInitCommand initializes device ID for vault sync.
func SyncVaultInitCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("sync vault-init", flag.ExitOnError)
	_ = fs.Parse(args)

	// Load existing config or create new one
	cfg, err := sync.LoadVaultConfig()
	if err != nil {
		return fmt.Errorf("failed to load vault config: %w", err)
	}

	// Generate device ID if not already set
	if cfg.DeviceID == "" {
		cfg.DeviceID = sync.GenerateVaultDeviceID()
		fmt.Printf("✓ Generated new device ID: %s\n", cfg.DeviceID)
	} else {
		fmt.Printf("✓ Device already initialized: %s\n", cfg.DeviceID)
	}

	// Save config
	if err := sync.SaveVaultConfig(cfg); err != nil {
		return fmt.Errorf("failed to save vault config: %w", err)
	}

	fmt.Printf("✓ Configuration saved to %s\n", sync.VaultConfigPath())
	fmt.Println("\nNext step: Run 'pagen sync vault-login' to authenticate")

	return nil
}

// SyncVaultLoginCommand authenticates with vault server.
func SyncVaultLoginCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("sync vault-login", flag.ExitOnError)
	server := fs.String("server", "https://api.storeusa.org", "Vault server URL")
	_ = fs.Parse(args)

	// Load config
	cfg, err := sync.LoadVaultConfig()
	if err != nil {
		return fmt.Errorf("failed to load vault config: %w", err)
	}

	// Ensure device ID is set
	if cfg.DeviceID == "" {
		return fmt.Errorf("device not initialized. Run 'pagen sync vault-init' first")
	}

	// Prompt for email
	fmt.Print("Email: ")
	var email string
	if _, err := fmt.Scanln(&email); err != nil {
		return fmt.Errorf("failed to read email: %w", err)
	}
	email = strings.TrimSpace(email)

	// Prompt for password (hidden)
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println() // New line after hidden input
	password := string(passwordBytes)

	// Prompt for recovery phrase
	fmt.Println("Recovery phrase (24 BIP39 words):")
	fmt.Print("> ")
	var mnemonic string
	// Read entire line including spaces
	reader := strings.Builder{}
	var word string
	for i := 0; i < 24; i++ {
		if _, err := fmt.Scan(&word); err != nil {
			return fmt.Errorf("failed to read recovery phrase: %w", err)
		}
		if i > 0 {
			reader.WriteString(" ")
		}
		reader.WriteString(word)
	}
	mnemonic = reader.String()

	// Validate mnemonic
	if _, err := vault.ParseMnemonic(mnemonic); err != nil {
		return fmt.Errorf("invalid recovery phrase: %w", err)
	}

	// Authenticate with vault (v0.3+ requires device ID at login)
	client := vault.NewPBAuthClient(*server)
	result, err := client.Login(context.Background(), email, password, cfg.DeviceID)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Convert mnemonic to hex-encoded seed
	seed, err := vault.ParseSeedPhrase(mnemonic)
	if err != nil {
		return fmt.Errorf("failed to parse seed phrase: %w", err)
	}
	derivedKey := hex.EncodeToString(seed.Raw)

	// Update config with auth credentials
	cfg.Server = *server
	cfg.UserID = result.UserID
	cfg.Token = result.Token.Token
	cfg.RefreshToken = result.RefreshToken
	cfg.TokenExpires = result.Token.Expires.Format(time.RFC3339)
	cfg.DerivedKey = derivedKey

	// Save config
	if err := sync.SaveVaultConfig(cfg); err != nil {
		return fmt.Errorf("failed to save vault config: %w", err)
	}

	fmt.Println("\n✓ Authentication successful!")
	fmt.Printf("✓ User ID: %s\n", result.UserID)
	fmt.Printf("✓ Token expires: %s\n", result.Token.Expires.Format(time.RFC3339))
	fmt.Printf("✓ Configuration saved to %s\n", sync.VaultConfigPath())
	fmt.Println("\nReady to sync! Run 'pagen sync vault-now' to synchronize data")

	return nil
}

// SyncVaultStatusCommand shows vault sync status.
func SyncVaultStatusCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("sync vault-status", flag.ExitOnError)
	_ = fs.Parse(args)

	// Load config
	cfg, err := sync.LoadVaultConfig()
	if err != nil {
		return fmt.Errorf("failed to load vault config: %w", err)
	}

	fmt.Println("Vault Sync Status:")
	fmt.Printf("  Config path:  %s\n", sync.VaultConfigPath())
	fmt.Printf("  Server:       %s\n", cfg.Server)
	fmt.Printf("  User ID:      %s\n", cfg.UserID)
	fmt.Printf("  Device ID:    %s\n", cfg.DeviceID)

	// Show configuration status
	if cfg.IsConfigured() {
		fmt.Printf("  Configured:   ✓ Yes\n")
	} else {
		fmt.Printf("  Configured:   ✗ No (run 'pagen sync vault-login')\n")
		return nil
	}

	// Show token validity
	if cfg.TokenExpires != "" {
		expiresAt, err := time.Parse(time.RFC3339, cfg.TokenExpires)
		if err == nil {
			if time.Now().Before(expiresAt) {
				fmt.Printf("  Token valid:  ✓ Yes (expires %s)\n", expiresAt.Format(time.RFC3339))
			} else {
				fmt.Printf("  Token valid:  ✗ Expired (run 'pagen sync vault-login')\n")
			}
		}
	}

	// Try to get syncer stats
	if cfg.IsConfigured() {
		syncer, err := sync.NewVaultSyncer(cfg, database)
		if err != nil {
			fmt.Printf("  Syncer:       ✗ Error: %v\n", err)
			return nil
		}
		defer func() { _ = syncer.Close() }()

		ctx := context.Background()

		// Get pending count
		pendingCount, err := syncer.PendingCount(ctx)
		if err != nil {
			fmt.Printf("  Pending:      ✗ Error: %v\n", err)
		} else {
			fmt.Printf("  Pending:      %d changes\n", pendingCount)
		}

		// Get last synced sequence
		lastSeq, err := syncer.LastSyncedSeq(ctx)
		if err != nil {
			fmt.Printf("  Last sync:    ✗ Error: %v\n", err)
		} else {
			fmt.Printf("  Last seq:     %d\n", lastSeq)
		}
	}

	return nil
}

// SyncVaultNowCommand triggers manual vault sync.
func SyncVaultNowCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("sync vault-now", flag.ExitOnError)
	verbose := fs.Bool("verbose", false, "Show detailed progress")
	_ = fs.Parse(args)

	// Load config
	cfg, err := sync.LoadVaultConfig()
	if err != nil {
		return fmt.Errorf("failed to load vault config: %w", err)
	}

	if !cfg.IsConfigured() {
		return fmt.Errorf("vault not configured. Run 'pagen sync vault-login' first")
	}

	// Create syncer
	syncer, err := sync.NewVaultSyncer(cfg, database)
	if err != nil {
		return fmt.Errorf("failed to create syncer: %w", err)
	}
	defer func() { _ = syncer.Close() }()

	ctx := context.Background()
	startTime := time.Now()

	if *verbose {
		// Sync with events
		events := vault.SyncEvents{
			OnStart: func() {
				fmt.Println("Starting sync...")
			},
			OnPush: func(pushed, remaining int) {
				fmt.Printf("⬆ Pushed %d changes (%d remaining)\n", pushed, remaining)
			},
			OnPull: func(pulled int) {
				fmt.Printf("⬇ Pulled %d changes\n", pulled)
			},
			OnComplete: func(pushed, pulled int) {
				fmt.Printf("✓ Sync complete: pushed %d, pulled %d\n", pushed, pulled)
			},
		}

		if err := syncer.SyncWithEvents(ctx, events); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}
	} else {
		// Simple sync without events
		if err := syncer.Sync(ctx); err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}
	}

	duration := time.Since(startTime)
	fmt.Printf("\n✓ Sync completed in %.2fs\n", duration.Seconds())

	return nil
}

// SyncVaultPendingCommand lists pending vault changes.
func SyncVaultPendingCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("sync vault-pending", flag.ExitOnError)
	_ = fs.Parse(args)

	// Load config
	cfg, err := sync.LoadVaultConfig()
	if err != nil {
		return fmt.Errorf("failed to load vault config: %w", err)
	}

	if !cfg.IsConfigured() {
		return fmt.Errorf("vault not configured. Run 'pagen sync vault-login' first")
	}

	// Create syncer
	syncer, err := sync.NewVaultSyncer(cfg, database)
	if err != nil {
		return fmt.Errorf("failed to create syncer: %w", err)
	}
	defer func() { _ = syncer.Close() }()

	// Get pending changes
	ctx := context.Background()
	changes, err := syncer.PendingChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending changes: %w", err)
	}

	if len(changes) == 0 {
		fmt.Println("No pending changes")
		return nil
	}

	fmt.Printf("Pending Changes (%d):\n\n", len(changes))
	for _, item := range changes {
		createdAt := time.Unix(item.TS, 0)
		fmt.Printf("  Change ID: %s\n", item.ChangeID)
		fmt.Printf("  Entity:    %s\n", item.Entity)
		fmt.Printf("  Created:   %s\n", createdAt.Format(time.RFC3339))
		fmt.Println()
	}

	return nil
}

// SyncVaultLogoutCommand clears vault tokens.
func SyncVaultLogoutCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("sync vault-logout", flag.ExitOnError)
	_ = fs.Parse(args)

	// Load config
	cfg, err := sync.LoadVaultConfig()
	if err != nil {
		return fmt.Errorf("failed to load vault config: %w", err)
	}

	// Clear tokens but keep derived key
	cfg.Token = ""
	cfg.RefreshToken = ""
	cfg.TokenExpires = ""
	cfg.UserID = ""

	// Save config
	if err := sync.SaveVaultConfig(cfg); err != nil {
		return fmt.Errorf("failed to save vault config: %w", err)
	}

	fmt.Println("✓ Logged out successfully")
	fmt.Println("  Tokens cleared")
	fmt.Println("  Derived key preserved (recovery phrase is precious)")
	fmt.Println("\nRun 'pagen sync vault-login' to authenticate again")

	return nil
}

// SyncVaultWipeCommand clears all vault sync data.
func SyncVaultWipeCommand(database *sql.DB, args []string) error {
	fs := flag.NewFlagSet("sync vault-wipe", flag.ExitOnError)
	confirm := fs.Bool("confirm", false, "Confirm wipe operation")
	_ = fs.Parse(args)

	if !*confirm {
		fmt.Println("⚠️  WARNING: This will permanently delete all vault sync data!")
		fmt.Println("  - Vault database file will be deleted")
		fmt.Println("  - All configuration will be cleared")
		fmt.Println("  - This action cannot be undone")
		fmt.Println()
		fmt.Println("To proceed, run: pagen sync vault-wipe --confirm")
		return nil
	}

	// Load config
	cfg, err := sync.LoadVaultConfig()
	if err != nil {
		return fmt.Errorf("failed to load vault config: %w", err)
	}

	// Delete vault database if it exists
	if cfg.VaultDB != "" {
		if err := os.Remove(cfg.VaultDB); err != nil && !os.IsNotExist(err) {
			fmt.Printf("⚠️  Warning: failed to delete vault database: %v\n", err)
		} else if err == nil {
			fmt.Printf("✓ Deleted vault database: %s\n", cfg.VaultDB)
		}
	}

	// Clear all config fields
	cfg.Server = ""
	cfg.UserID = ""
	cfg.Token = ""
	cfg.RefreshToken = ""
	cfg.TokenExpires = ""
	cfg.DerivedKey = ""
	cfg.DeviceID = ""
	cfg.VaultDB = "vault.db"
	cfg.AutoSync = false

	// Save empty config
	if err := sync.SaveVaultConfig(cfg); err != nil {
		return fmt.Errorf("failed to save vault config: %w", err)
	}

	fmt.Println("✓ All vault sync data wiped")
	fmt.Println("\nRun 'pagen sync vault-init' to start fresh")

	return nil
}
