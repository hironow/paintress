package session_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestFetchIssuesViaMCP_ParsesOutput(t *testing.T) {
	// given — fake claude script that writes issues JSON to the output path
	dir := t.TempDir()
	fakeScript := filepath.Join(dir, "fake-claude.sh")

	// The script extracts the output path from the prompt and writes test data
	script := `#!/bin/bash
# Extract output path from prompt (last argument after -p)
prompt=""
while [ $# -gt 0 ]; do
  case "$1" in
    -p) shift; prompt="$1" ;;
  esac
  shift
done
# Extract path after "JSON array to " (POSIX-compatible)
output_path=$(echo "$prompt" | sed -n 's/.*JSON array to \([^ ]*\).*/\1/p')
cat > "$output_path" << 'JSONEOF'
[
  {"id":"MY-42","title":"Fix login bug","priority":2,"status":"Todo","labels":["Bug"]},
  {"id":"MY-43","title":"Add dark mode","priority":3,"status":"In Progress","labels":["Enhancement"]}
]
JSONEOF
`
	if err := os.WriteFile(fakeScript, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	// when
	issues, err := session.FetchIssuesViaMCP(context.Background(), fakeScript, "MY", "", dir)

	// then
	if err != nil {
		t.Fatalf("FetchIssuesViaMCP: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].ID != "MY-42" {
		t.Errorf("issues[0].ID = %q, want MY-42", issues[0].ID)
	}
	if issues[0].Status != "Todo" {
		t.Errorf("issues[0].Status = %q, want Todo", issues[0].Status)
	}
	if len(issues[0].Labels) != 1 || issues[0].Labels[0] != "Bug" {
		t.Errorf("issues[0].Labels = %v, want [Bug]", issues[0].Labels)
	}
	if issues[1].Title != "Add dark mode" {
		t.Errorf("issues[1].Title = %q, want %q", issues[1].Title, "Add dark mode")
	}
}

func TestFetchIssuesViaMCP_EmptyResult(t *testing.T) {
	// given — fake claude that writes empty array
	dir := t.TempDir()
	fakeScript := filepath.Join(dir, "fake-claude.sh")
	script := `#!/bin/bash
prompt=""
while [ $# -gt 0 ]; do
  case "$1" in
    -p) shift; prompt="$1" ;;
  esac
  shift
done
output_path=$(echo "$prompt" | sed -n 's/.*JSON array to \([^ ]*\).*/\1/p')
echo "[]" > "$output_path"
`
	os.WriteFile(fakeScript, []byte(script), 0755)

	// when
	issues, err := session.FetchIssuesViaMCP(context.Background(), fakeScript, "MY", "", dir)

	// then
	if err != nil {
		t.Fatalf("FetchIssuesViaMCP: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestFetchIssuesViaMCP_ClaudeFailure(t *testing.T) {
	// given — fake claude that exits with error
	dir := t.TempDir()
	fakeScript := filepath.Join(dir, "fake-claude.sh")
	os.WriteFile(fakeScript, []byte("#!/bin/bash\nexit 1\n"), 0755)

	// when
	_, err := session.FetchIssuesViaMCP(context.Background(), fakeScript, "MY", "", dir)

	// then
	if err == nil {
		t.Fatal("expected error for claude failure")
	}
}

func TestFetchIssuesViaMCP_MalformedJSON(t *testing.T) {
	// given — fake claude that writes invalid JSON
	dir := t.TempDir()
	fakeScript := filepath.Join(dir, "fake-claude.sh")
	script := `#!/bin/bash
prompt=""
while [ $# -gt 0 ]; do
  case "$1" in
    -p) shift; prompt="$1" ;;
  esac
  shift
done
output_path=$(echo "$prompt" | sed -n 's/.*JSON array to \([^ ]*\).*/\1/p')
echo "{invalid json" > "$output_path"
`
	os.WriteFile(fakeScript, []byte(script), 0755)

	// when
	_, err := session.FetchIssuesViaMCP(context.Background(), fakeScript, "MY", "", dir)

	// then
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should mention parsing: %v", err)
	}
}

// Keep unused import suppressed
var _ = json.Marshal
var _ = exec.Command
var _ domain.Issue
