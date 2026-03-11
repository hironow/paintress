package session_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/session"
)

func TestSanitizeJSONFile_ValidJSON(t *testing.T) {
	// given
	dir := t.TempDir()
	path := filepath.Join(dir, "valid.json")
	if err := os.WriteFile(path, []byte(`[{"id":"T-1","title":"test"}]`), 0644); err != nil {
		t.Fatal(err)
	}

	// when
	data, err := session.SanitizeJSONFile(path)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(data), `"id"`) {
		t.Errorf("expected valid JSON, got: %s", string(data))
	}
}

func TestSanitizeJSONFile_TextPrefixBeforeArray(t *testing.T) {
	// given: Claude returns text before JSON array
	dir := t.TempDir()
	path := filepath.Join(dir, "prefix.json")
	content := "Here are the issues:\n\n" + `[{"id":"T-1","title":"test"}]`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// when
	data, err := session.SanitizeJSONFile(path)

	// then
	if err != nil {
		t.Fatalf("expected text-prefixed JSON array to parse, got: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(data)), "[") {
		t.Errorf("expected JSON array, got: %s", string(data))
	}
}

func TestSanitizeJSONFile_MarkdownCodeBlock(t *testing.T) {
	// given
	dir := t.TempDir()
	path := filepath.Join(dir, "markdown.json")
	content := "```json\n" + `[{"id":"T-1"}]` + "\n```"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// when
	data, err := session.SanitizeJSONFile(path)

	// then
	if err != nil {
		t.Fatalf("expected markdown-wrapped JSON to parse, got: %v", err)
	}
	if strings.Contains(string(data), "```") {
		t.Errorf("expected markdown fences removed, got: %s", string(data))
	}
}

func TestSanitizeJSONFile_TextPrefixBeforeObject(t *testing.T) {
	// given
	dir := t.TempDir()
	path := filepath.Join(dir, "obj.json")
	content := "Certainly:\n\n" + `{"name":"test","value":42}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// when
	data, err := session.SanitizeJSONFile(path)

	// then
	if err != nil {
		t.Fatalf("expected text-prefixed JSON object to parse, got: %v", err)
	}
	if !strings.Contains(string(data), `"name"`) {
		t.Errorf("expected JSON object, got: %s", string(data))
	}
}
