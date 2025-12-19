// ABOUTME: CLI commands for Charm KV sync operations
// ABOUTME: Simplified sync with SSH key auth - no login/logout needed

package charm

import (
	"flag"
	"fmt"

	"github.com/charmbracelet/charm/client"
)

// SyncLinkCommand links this device to a Charm account
// Uses SSH key auth - first run on a new device will create an account
// Subsequent devices can link via QR code: `charm link`.
func SyncLinkCommand(args []string) error {
	fs := flag.NewFlagSet("sync link", flag.ExitOnError)
	_ = fs.Parse(args)

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Linked {
		fmt.Println("✓ Device already linked")
		return showSyncStatus(cfg)
	}

	fmt.Println("Linking device to Charm account...")
	fmt.Printf("Server: %s\n\n", cfg.Host)

	// The charm client uses SSH keys automatically
	// On first run, it creates an account
	// On subsequent devices, use `charm link` to scan QR code
	cc, err := client.NewClientWithDefaults()
	if err != nil {
		return fmt.Errorf("failed to create charm client: %w", err)
	}

	id, err := cc.ID()
	if err != nil {
		fmt.Println("No account found. Creating new account...")
		fmt.Println("\nTo link additional devices:")
		fmt.Println("  1. Run 'charm link' to display a QR code")
		fmt.Println("  2. Scan the code on your other device")
		fmt.Println("  3. Both devices will share the same data")
		return fmt.Errorf("run 'charm link' to complete setup: %w", err)
	}

	// Mark as linked
	if err := cfg.MarkLinked(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Linked to account: %s\n", id)
	fmt.Printf("✓ Auto-sync: %v\n", cfg.AutoSync)

	return nil
}

// SyncStatusCommand shows current sync configuration and status.
func SyncStatusCommand(args []string) error {
	fs := flag.NewFlagSet("sync status", flag.ExitOnError)
	_ = fs.Parse(args)

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	return showSyncStatus(cfg)
}

func showSyncStatus(cfg *Config) error {
	fmt.Println("Charm Sync Status")
	fmt.Println("─────────────────")
	fmt.Printf("Server:    %s\n", cfg.Host)
	fmt.Printf("Linked:    %v\n", cfg.Linked)
	fmt.Printf("Auto-sync: %v\n", cfg.AutoSync)

	if cfg.Linked {
		// Try to get user ID
		cc, err := client.NewClientWithDefaults()
		if err == nil {
			if id, err := cc.ID(); err == nil {
				fmt.Printf("User ID:   %s\n", id)
			}
		}

		// Show KV stats
		c, err := GetClient()
		if err == nil {
			keys, err := c.Keys()
			if err == nil {
				fmt.Printf("Keys:      %d\n", len(keys))
			}
		}
	}

	return nil
}

// SyncUnlinkCommand disconnects this device from the Charm account
// Local data is preserved but will no longer sync.
func SyncUnlinkCommand(args []string) error {
	fs := flag.NewFlagSet("sync unlink", flag.ExitOnError)
	_ = fs.Parse(args)

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.Linked {
		fmt.Println("Device is not linked")
		return nil
	}

	// Mark as unlinked
	if err := cfg.MarkUnlinked(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("✓ Device unlinked")
	fmt.Println("  Local data preserved but sync disabled")
	fmt.Println("  Run 'pagen sync link' to re-enable sync")

	return nil
}

// SyncWipeCommand completely resets the KV store
// WARNING: This deletes all local data!
func SyncWipeCommand(args []string) error {
	fs := flag.NewFlagSet("sync wipe", flag.ExitOnError)
	confirm := fs.Bool("confirm", false, "Confirm data wipe")
	_ = fs.Parse(args)

	if !*confirm {
		fmt.Println("WARNING: This will delete ALL local data!")
		fmt.Println()
		fmt.Println("To confirm, run:")
		fmt.Println("  pagen sync wipe --confirm")
		return nil
	}

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	// Reset the KV store
	if err := c.Reset(); err != nil {
		return fmt.Errorf("failed to reset KV store: %w", err)
	}

	// Mark as unlinked
	_ = cfg.MarkUnlinked()

	fmt.Println("✓ All data wiped")
	fmt.Println("  Run 'pagen sync link' to set up again")

	return nil
}

// SyncNowCommand performs an immediate sync.
func SyncNowCommand(args []string) error {
	fs := flag.NewFlagSet("sync now", flag.ExitOnError)
	verbose := fs.Bool("verbose", false, "Show verbose output")
	_ = fs.Parse(args)

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.Linked {
		return fmt.Errorf("device not linked. Run 'pagen sync link' first")
	}

	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if *verbose {
		fmt.Println("Syncing with server...")
	}

	if err := c.Sync(); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	if *verbose {
		fmt.Println("✓ Sync complete")
	} else {
		fmt.Println("✓ Synced")
	}

	return nil
}

// SetAutoSyncCommand enables or disables auto-sync.
func SetAutoSyncCommand(args []string) error {
	fs := flag.NewFlagSet("sync auto", flag.ExitOnError)
	enable := fs.Bool("enable", false, "Enable auto-sync")
	disable := fs.Bool("disable", false, "Disable auto-sync")
	_ = fs.Parse(args)

	if !*enable && !*disable {
		fmt.Println("Usage: pagen sync auto --enable|--disable")
		return nil
	}

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if *enable {
		if err := cfg.SetAutoSync(true); err != nil {
			return fmt.Errorf("failed to enable auto-sync: %w", err)
		}
		fmt.Println("✓ Auto-sync enabled")
	} else if *disable {
		if err := cfg.SetAutoSync(false); err != nil {
			return fmt.Errorf("failed to disable auto-sync: %w", err)
		}
		fmt.Println("✓ Auto-sync disabled")
	}

	return nil
}
