package cmd_test

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/cmd"

	_ "modernc.org/sqlite"
)

// setupDeadLetterDB creates a minimal outbox.db with the staged table and
// optional dead-lettered rows for testing.
func setupDeadLetterDB(t *testing.T, repoDir string, deadLetterCount int) {
	t.Helper()
	runDir := filepath.Join(repoDir, ".expedition", ".run")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("create .run dir: %v", err)
	}
	dbPath := filepath.Join(runDir, "outbox.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS staged (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		kind TEXT NOT NULL DEFAULT '',
		dest TEXT NOT NULL DEFAULT '',
		payload BLOB NOT NULL,
		flushed INTEGER NOT NULL DEFAULT 0,
		retry_count INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	for i := 0; i < deadLetterCount; i++ {
		_, err = db.Exec(`INSERT INTO staged (kind, dest, payload, flushed, retry_count) VALUES (?, ?, ?, 0, 3)`,
			"test", "test-dest", []byte("test-payload"))
		if err != nil {
			t.Fatalf("insert dead letter row: %v", err)
		}
	}
}

func TestDeadLettersPurge_NoDB(t *testing.T) {
	// given: no outbox.db exists
	dir := t.TempDir()

	root := cmd.NewRootCommand()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs([]string{"dead-letters", "purge", dir})

	// when
	err := root.Execute()

	// then: should succeed (store is created on open) with 0 dead letters
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(errBuf.String(), "No dead-lettered") && !strings.Contains(errBuf.String(), "no dead") {
		t.Errorf("expected 'no dead-lettered' message, got: %s", errBuf.String())
	}
}

func TestDeadLettersPurge_NoDeadLetters(t *testing.T) {
	// given: outbox.db exists but has no dead letters
	dir := t.TempDir()
	setupDeadLetterDB(t, dir, 0)

	root := cmd.NewRootCommand()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs([]string{"dead-letters", "purge", dir})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(errBuf.String(), "No dead-lettered items") {
		t.Errorf("expected 'No dead-lettered items' in output, got: %q", errBuf.String())
	}
}

func TestDeadLettersPurge_DryRun(t *testing.T) {
	// given: outbox.db with 2 dead-lettered items
	dir := t.TempDir()
	setupDeadLetterDB(t, dir, 2)

	root := cmd.NewRootCommand()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs([]string{"dead-letters", "purge", dir})

	// when
	err := root.Execute()

	// then: dry-run should report count but not purge
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := errBuf.String()
	if !strings.Contains(output, "2 dead-lettered") {
		t.Errorf("expected count in output, got: %q", output)
	}
	if !strings.Contains(output, "dry-run") {
		t.Errorf("expected dry-run hint in output, got: %q", output)
	}
	// stdout must be empty in text mode
	if outBuf.Len() != 0 {
		t.Errorf("text mode should not write to stdout, got: %q", outBuf.String())
	}
}

func TestDeadLettersPurge_Execute(t *testing.T) {
	// given: outbox.db with 3 dead-lettered items
	dir := t.TempDir()
	setupDeadLetterDB(t, dir, 3)

	root := cmd.NewRootCommand()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs([]string{"dead-letters", "purge", "--execute", "--yes", dir})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := errBuf.String()
	if !strings.Contains(output, "Purged 3") {
		t.Errorf("expected 'Purged 3' in output, got: %q", output)
	}
}

func TestDeadLettersPurge_JSONOutput_DryRun(t *testing.T) {
	// given: outbox.db with 2 dead-lettered items
	dir := t.TempDir()
	setupDeadLetterDB(t, dir, 2)

	root := cmd.NewRootCommand()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs([]string{"dead-letters", "purge", "-o", "json", dir})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := outBuf.String()
	if !strings.Contains(out, `"count":2`) {
		t.Errorf("expected count:2 in JSON output, got: %q", out)
	}
	if !strings.Contains(out, `"purged":0`) {
		t.Errorf("expected purged:0 in JSON dry-run output, got: %q", out)
	}
}

func TestDeadLettersPurge_JSONOutput_Execute(t *testing.T) {
	// given: outbox.db with 2 dead-lettered items
	dir := t.TempDir()
	setupDeadLetterDB(t, dir, 2)

	root := cmd.NewRootCommand()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs([]string{"dead-letters", "purge", "-o", "json", "--execute", dir})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := outBuf.String()
	if !strings.Contains(out, `"count":2`) {
		t.Errorf("expected count:2 in JSON output, got: %q", out)
	}
	if !strings.Contains(out, `"purged":2`) {
		t.Errorf("expected purged:2 in JSON execute output, got: %q", out)
	}
}

func TestDeadLettersPurge_CommandRegistered(t *testing.T) {
	// given
	root := cmd.NewRootCommand()

	// when: find dead-letters subcommand
	dlCmd, _, err := root.Find([]string{"dead-letters", "purge"})

	// then
	if err != nil {
		t.Fatalf("find dead-letters purge: %v", err)
	}
	if dlCmd == nil {
		t.Fatal("dead-letters purge subcommand not found")
	}
	if dlCmd.Name() != "purge" {
		t.Errorf("expected command name 'purge', got %q", dlCmd.Name())
	}
}

func TestDeadLettersPurge_FlagDefaults(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	dlCmd, _, err := root.Find([]string{"dead-letters", "purge"})
	if err != nil {
		t.Fatalf("find dead-letters purge: %v", err)
	}

	// then: --execute defaults to false
	execFlag := dlCmd.Flags().Lookup("execute")
	if execFlag == nil {
		t.Fatal("--execute flag not found")
	}
	if execFlag.DefValue != "false" {
		t.Errorf("--execute default = %q, want %q", execFlag.DefValue, "false")
	}

	// then: --yes defaults to false
	yesFlag := dlCmd.Flags().Lookup("yes")
	if yesFlag == nil {
		t.Fatal("--yes flag not found")
	}
	if yesFlag.DefValue != "false" {
		t.Errorf("--yes default = %q, want %q", yesFlag.DefValue, "false")
	}
}
