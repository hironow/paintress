package cmd_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/cmd"
	"github.com/hironow/paintress/internal/domain"
)

func TestConfigCommand_ShowSubcommandExists(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	configCmd, _, err := root.Find([]string{"config"})
	if err != nil {
		t.Fatalf("find config command: %v", err)
	}

	// when
	var found bool
	for _, sub := range configCmd.Commands() {
		if sub.Name() == "show" {
			found = true
			break
		}
	}

	// then
	if !found {
		t.Error("config show subcommand not found")
	}
}

func TestConfigCommand_SetSubcommandExists(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	configCmd, _, err := root.Find([]string{"config"})
	if err != nil {
		t.Fatalf("find config command: %v", err)
	}

	// when
	var found bool
	for _, sub := range configCmd.Commands() {
		if sub.Name() == "set" {
			found = true
			break
		}
	}

	// then
	if !found {
		t.Error("config set subcommand not found")
	}
}

func TestConfigCommand_SetKeyValue_OnInitializedProject(t *testing.T) {
	// given: initialized project with config
	dir := t.TempDir()
	initRoot := cmd.NewRootCommand()
	initBuf := new(bytes.Buffer)
	initRoot.SetOut(initBuf)
	initRoot.SetErr(initBuf)
	initRoot.SetIn(strings.NewReader(""))
	initRoot.SetArgs([]string{"init", "--team", "MY", dir})
	if err := initRoot.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"config", "set", "lang", "en", dir})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	// verify the value was persisted
	data, readErr := os.ReadFile(domain.ProjectConfigPath(dir))
	if readErr != nil {
		t.Fatalf("read config: %v", readErr)
	}
	if !strings.Contains(string(data), "lang: en") {
		t.Errorf("expected lang=en in config, got:\n%s", string(data))
	}
}

// initProject creates an initialized paintress project in a temp directory.
func initProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader(""))
	root.SetArgs([]string{"init", "--team", "TEST", dir})
	if err := root.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	return dir
}

func TestConfigCommand_SetAllKeys(t *testing.T) {
	tests := []struct {
		key      string
		value    string
		contains string // substring expected in persisted config file
	}{
		// string keys
		{"tracker.team", "NEWTEAM", "team: NEWTEAM"},
		{"tracker.project", "MyProj", "project: MyProj"},
		{"tracker.cycle", "2026-Q1", "cycle: 2026-Q1"},
		{"model", "sonnet,haiku", "model: sonnet,haiku"},
		{"base_branch", "develop", "base_branch: develop"},
		{"claude_cmd", "claude-dev", "claude_cmd: claude-dev"},
		{"dev_cmd", "pnpm dev", "dev_cmd: pnpm dev"},
		{"dev_dir", "/tmp/devdir", "dev_dir: /tmp/devdir"},
		{"dev_url", "http://localhost:5173", "dev_url: http://localhost:5173"},
		{"review_cmd", "pnpm lint", "review_cmd: pnpm lint"},
		{"setup_cmd", "bun install", "setup_cmd: bun install"},
		{"notify_cmd", "notify-send", "notify_cmd: notify-send"},
		{"approve_cmd", "custom-approve", "approve_cmd: custom-approve"},
		// int keys
		{"max_expeditions", "100", "max_expeditions: 100"},
		{"timeout_sec", "600", "timeout_sec: 600"},
		{"workers", "3", "workers: 3"},
		{"max_retries", "5", "max_retries: 5"},
		// bool keys
		{"no_dev", "true", "no_dev: true"},
		{"auto_approve", "true", "auto_approve: true"},
		// duration key
		{"idle_timeout", "45m0s", "idle_timeout: 45m0s"},
		// lang key (enum)
		{"lang", "en", "lang: en"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			// given
			dir := initProject(t)

			root := cmd.NewRootCommand()
			buf := new(bytes.Buffer)
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs([]string{"config", "set", tt.key, tt.value, dir})

			// when
			err := root.Execute()

			// then
			if err != nil {
				t.Fatalf("config set %s=%s failed: %v", tt.key, tt.value, err)
			}

			data, readErr := os.ReadFile(domain.ProjectConfigPath(dir))
			if readErr != nil {
				t.Fatalf("read config: %v", readErr)
			}
			if !strings.Contains(string(data), tt.contains) {
				t.Errorf("expected %q in config, got:\n%s", tt.contains, string(data))
			}
		})
	}
}

func TestConfigCommand_Set_RejectsUnknownKey(t *testing.T) {
	// given
	dir := initProject(t)

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"config", "set", "nonexistent.key", "value", dir})

	// when
	err := root.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for unknown config key")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected error to mention 'unknown', got: %s", err.Error())
	}
}

func TestConfigCommand_Set_RejectsInvalidIntValue(t *testing.T) {
	// given
	dir := initProject(t)

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"config", "set", "workers", "not-a-number", dir})

	// when
	err := root.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for non-integer workers value")
	}
}

func TestConfigCommand_Set_RejectsInvalidBoolValue(t *testing.T) {
	// given
	dir := initProject(t)

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"config", "set", "no_dev", "maybe", dir})

	// when
	err := root.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for invalid bool value")
	}
}

func TestConfigCommand_Set_RejectsInvalidLang(t *testing.T) {
	// given
	dir := initProject(t)

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"config", "set", "lang", "fr", dir})

	// when
	err := root.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for invalid lang 'fr'")
	}
	if !strings.Contains(err.Error(), "invalid lang") {
		t.Errorf("expected 'invalid lang' in error, got: %s", err.Error())
	}
}

func TestConfigCommand_Show_DisplaysConfig(t *testing.T) {
	// given: initialized project
	dir := t.TempDir()
	initRoot := cmd.NewRootCommand()
	initBuf := new(bytes.Buffer)
	initRoot.SetOut(initBuf)
	initRoot.SetErr(initBuf)
	initRoot.SetIn(strings.NewReader(""))
	initRoot.SetArgs([]string{"init", "--team", "SHOW", "--project", "TestProject", dir})
	if err := initRoot.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"config", "show", dir})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("config show failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "SHOW") {
		t.Errorf("expected team 'SHOW' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "TestProject") {
		t.Errorf("expected project 'TestProject' in output, got:\n%s", output)
	}
}
