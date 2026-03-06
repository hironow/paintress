//go:build contract

package domain_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

const contractGoldenDir = "testdata/contract"

func contractGoldenFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(contractGoldenDir)
	if err != nil {
		t.Fatalf("read contract golden dir: %v", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			files = append(files, e.Name())
		}
	}
	if len(files) == 0 {
		t.Fatal("no contract golden files found")
	}
	return files
}

func readContractGolden(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(contractGoldenDir, name))
	if err != nil {
		t.Fatalf("read contract golden %s: %v", name, err)
	}
	return data
}

// TestContract_ParseDMail verifies that paintress's ParseDMail can
// parse all cross-tool golden files. Paintress is Postel-liberal at
// the parse level — unknown kinds and future schemas parse without error.
func TestContract_domain.ParseDMail(t *testing.T) {
	for _, name := range contractGoldenFiles(t) {
		t.Run(name, func(t *testing.T) {
			data := readContractGolden(t, name)
			dm, err := domain.ParseDMail(data)
			if err != nil {
				t.Fatalf("ParseDMail error: %v", err)
			}
			if dm.Name == "" {
				t.Error("parsed name is empty")
			}
			if dm.Kind == "" {
				t.Error("parsed kind is empty")
			}
			if dm.Description == "" {
				t.Error("parsed description is empty")
			}
			if dm.SchemaVersion == "" {
				t.Error("parsed schema version is empty")
			}
		})
	}
}

// TestContract_ValidateDMailRejectsEdgeCases verifies that paintress's
// ValidateDMail rejects D-Mails with unsupported schema version.
// NOTE: paintress ValidateDMail does NOT validate kind enum (unlike
// phonewave/sightjack). This is a known divergence — kind validation
// is deferred to the consuming tool's business logic.
func TestContract_ValidateDMailRejectsEdgeCases(t *testing.T) {
	// future-schema.md has dmail-schema-version "2" — should be rejected
	data := readContractGolden(t, "future-schema.md")
	dm, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}
	if err := domain.ValidateDMail(dm); err == nil {
		t.Error("expected ValidateDMail to fail for schema version '2', but it passed")
	}

	// unknown-kind.md parses and validates (paintress doesn't enforce kind enum)
	data = readContractGolden(t, "unknown-kind.md")
	dm, err = domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail error: %v", err)
	}
	if err := domain.ValidateDMail(dm); err != nil {
		t.Errorf("unexpected ValidateDMail error for unknown kind: %v", err)
	}
}
