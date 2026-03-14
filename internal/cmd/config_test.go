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
