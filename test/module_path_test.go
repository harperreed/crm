// ABOUTME: Guards the repository's canonical major-version Go module identity.
// ABOUTME: Prevents v2 release tags from becoming invisible to the Go toolchain.
package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModulePathMatchesMajorVersion(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	const want = "module github.com/harperreed/crm/v2"
	for line := range strings.SplitSeq(string(contents), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			if line != want {
				t.Fatalf("go.mod module directive = %q, want %q", line, want)
			}
			return
		}
	}

	t.Fatal("go.mod has no module directive")
}

func TestReadmeUsesVersionedGoInstallPath(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}

	readme := string(contents)
	const want = "go install github.com/harperreed/crm/v2/cmd/crm@latest"
	if !strings.Contains(readme, want) {
		t.Fatalf("README.md does not contain %q", want)
	}

	const stale = "go install github.com/harperreed/crm/cmd/crm@"
	if strings.Contains(readme, stale) {
		t.Fatalf("README.md contains stale install path %q", stale)
	}
}
