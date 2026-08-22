// ABOUTME: Exercises version reporting through a real CRM binary built from this checkout.
// ABOUTME: Ensures local builds stay on the dev version despite embedded VCS metadata.

package test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLICheckoutBuildReportsDev(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "crm")
	//nolint:gosec // The fixed Go tool builds this repository's CLI to a test-owned path.
	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/crm")
	build.Dir = ".."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build crm: %v\n%s", err, output)
	}

	commands := []struct {
		name string
		args []string
	}{
		{name: "version flag", args: []string{"--version"}},
		{name: "version command", args: []string{"version"}},
	}
	for _, command := range commands {
		t.Run(command.name, func(t *testing.T) {
			//nolint:gosec // The executable is the test-built CRM binary and arguments are fixed above.
			run := exec.Command(binaryPath, command.args...)
			output, err := run.CombinedOutput()
			if err != nil {
				t.Fatalf("crm %s: %v\n%s", strings.Join(command.args, " "), err, output)
			}
			firstLine, _, _ := strings.Cut(string(output), "\n")
			if firstLine != "crm version dev" {
				t.Fatalf("crm %s first line = %q, want %q", strings.Join(command.args, " "), firstLine, "crm version dev")
			}
		})
	}
}
