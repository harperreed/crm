// ABOUTME: Exercises CRM history through the built CLI against both storage backends.
// ABOUTME: Verifies real mutations remain inspectable after their current records are deleted.
package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/harperreed/crm/v2/internal/config"
)

var cliUUIDPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

func TestCLIHistoryBothBackends(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "crm")
	//nolint:gosec // The fixed Go tool builds this repository's CLI to a test-owned path.
	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/crm")
	build.Dir = ".."
	build.Env = os.Environ()
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build crm: %v\n%s", err, output)
	}

	for _, backend := range []string{"sqlite", "markdown"} {
		t.Run(backend, func(t *testing.T) {
			runCLIHistoryScenario(t, binaryPath, backend)
		})
	}
}

func runCLIHistoryScenario(t *testing.T, binaryPath, backend string) {
	t.Helper()
	env := configureCLIHistory(t, backend)
	contactOutput := runCLIHistory(t, binaryPath, env, "contact", "add", "Ada Byron", "--email", "ada@old.example")
	contactID := parseCLIUUID(t, contactOutput)
	companyOutput := runCLIHistory(t, binaryPath, env, "company", "add", "Analytical Engines", "--domain", "engines.example")
	companyID := parseCLIUUID(t, companyOutput)

	runCLIHistory(t, binaryPath, env, "contact", "edit", contactID[:8], "--name", "Ada Lovelace", "--email", "ada@new.example")
	relationshipOutput := runCLIHistory(t, binaryPath, env, "link", contactID[:8], companyID[:8], "--type", "founded", "--context", "history test")
	relationshipID := parseCLIUUID(t, relationshipOutput)
	runCLIHistory(t, binaryPath, env, "unlink", relationshipID)
	runCLIHistory(t, binaryPath, env, "contact", "rm", contactID[:8])
	runCLIHistory(t, binaryPath, env, "company", "rm", companyID[:8])

	contactHistory := runCLIHistory(t, binaryPath, env, "history", contactID, "--limit", "100")
	assertCLIHistoryLines(t, contactHistory, 5)
	if !strings.Contains(contactHistory, "UPDATE contact [cli]") ||
		!strings.Contains(contactHistory, "email, name") ||
		!strings.Contains(contactHistory, "CREATE relationship [cli]") ||
		!strings.Contains(contactHistory, "DELETE relationship [cli]") ||
		!strings.Contains(contactHistory, "DELETE contact [cli]") {
		t.Errorf("contact history omitted expected events:\n%s", contactHistory)
	}

	companyHistory := runCLIHistory(t, binaryPath, env, "history", companyID)
	assertCLIHistoryLines(t, companyHistory, 4)
	if !strings.Contains(companyHistory, "DELETE company [cli]") {
		t.Errorf("company history after deletion omitted delete event:\n%s", companyHistory)
	}

	relationshipHistory := runCLIHistory(t, binaryPath, env, "history", relationshipID)
	assertCLIHistoryLines(t, relationshipHistory, 2)
	if !strings.Contains(relationshipHistory, "CREATE relationship [cli]") ||
		!strings.Contains(relationshipHistory, "DELETE relationship [cli]") {
		t.Errorf("relationship history omitted link or unlink:\n%s", relationshipHistory)
	}

	deleteEventPrefix := historyEventPrefix(t, contactHistory, "DELETE contact [cli]")
	eventOutput := runCLIHistory(t, binaryPath, env, "history", "show", deleteEventPrefix)
	for _, want := range []string{
		"Entity: contact " + contactID,
		"Action: DELETE",
		"Source: cli",
		"Before:\n{",
		`"name": "Ada Lovelace"`,
		`"email": "ada@new.example"`,
		"After:\nnull\n",
	} {
		if !strings.Contains(eventOutput, want) {
			t.Errorf("history show output missing %q:\n%s", want, eventOutput)
		}
	}
}

func configureCLIHistory(t *testing.T, backend string) []string {
	t.Helper()
	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "config")
	dataHome := filepath.Join(tempDir, "data-home")
	dataDir := filepath.Join(tempDir, "crm-data")
	configDir := filepath.Join(configHome, "crm")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("MkdirAll config: %v", err)
	}
	encoded, err := json.Marshal(&config.Config{Backend: backend, DataDir: dataDir})
	if err != nil {
		t.Fatalf("Marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), encoded, 0o600); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}
	env := withoutEnvironment(os.Environ(), "XDG_CONFIG_HOME", "XDG_DATA_HOME")
	return append(env, "XDG_CONFIG_HOME="+configHome, "XDG_DATA_HOME="+dataHome)
}

func runCLIHistory(t *testing.T, binaryPath string, env []string, args ...string) string {
	t.Helper()
	//nolint:gosec // The executable is the test-built CRM binary and arguments are fixed by this test.
	command := exec.Command(binaryPath, args...)
	command.Env = env
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("crm %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func parseCLIUUID(t *testing.T, output string) string {
	t.Helper()
	id := cliUUIDPattern.FindString(output)
	if id == "" {
		t.Fatalf("output contains no UUID: %q", output)
	}
	return id
}

func assertCLIHistoryLines(t *testing.T, output string, wantCount int) {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != wantCount {
		t.Fatalf("history line count = %d, want %d:\n%s", len(lines), wantCount, output)
	}
	var previousTimestamp time.Time
	for index, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[3] != "[cli]" || len(fields[4]) != 8 {
			t.Errorf("malformed history line %q", line)
			continue
		}
		timestamp, err := time.Parse(time.RFC3339, fields[0])
		if err != nil {
			t.Errorf("history timestamp %q is not RFC3339: %v", fields[0], err)
			continue
		}
		if index > 0 && timestamp.After(previousTimestamp) {
			t.Errorf("history timestamps are not newest first: %s appears after %s", timestamp, previousTimestamp)
		}
		previousTimestamp = timestamp
	}
}

func historyEventPrefix(t *testing.T, output, marker string) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, marker) {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				return fields[4]
			}
		}
	}
	t.Fatalf("history output contains no %q event:\n%s", marker, output)
	return ""
}

func withoutEnvironment(environ []string, names ...string) []string {
	blocked := make(map[string]struct{}, len(names))
	for _, name := range names {
		blocked[name+"="] = struct{}{}
	}
	filtered := make([]string, 0, len(environ))
	for _, item := range environ {
		keep := true
		for prefix := range blocked {
			if strings.HasPrefix(item, prefix) {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
