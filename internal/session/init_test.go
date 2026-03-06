package session_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestInitProject_WritesConfig(t *testing.T) {
	// given
	dir := t.TempDir()

	// when
	err := session.InitProject(dir, "ENG", "backend", io.Discard)

	// then
	if err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	cfg, loadErr := session.LoadProjectConfig(dir)
	if loadErr != nil {
		t.Fatalf("LoadProjectConfig: %v", loadErr)
	}
	if cfg.Tracker.Team != "ENG" {
		t.Errorf("Team = %q, want %q", cfg.Tracker.Team, "ENG")
	}
	if cfg.Tracker.Project != "backend" {
		t.Errorf("Project = %q, want %q", cfg.Tracker.Project, "backend")
	}
}

func TestInitProject_SkipOptionalProject(t *testing.T) {
	// given
	dir := t.TempDir()

	// when
	err := session.InitProject(dir, "MY", "", io.Discard)

	// then
	if err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	cfg, loadErr := session.LoadProjectConfig(dir)
	if loadErr != nil {
		t.Fatalf("LoadProjectConfig: %v", loadErr)
	}
	if cfg.Tracker.Team != "MY" {
		t.Errorf("Team = %q, want %q", cfg.Tracker.Team, "MY")
	}
	if cfg.Tracker.Project != "" {
		t.Errorf("Project = %q, want empty", cfg.Tracker.Project)
	}
}

func TestInitProject_CreatesExpeditionDir(t *testing.T) {
	// given
	dir := t.TempDir()

	// when
	err := session.InitProject(dir, "MY", "", io.Discard)

	// then
	if err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	expeditionDir := filepath.Join(dir, ".expedition")
	info, statErr := os.Stat(expeditionDir)
	if statErr != nil {
		t.Fatalf(".expedition dir not created: %v", statErr)
	}
	if !info.IsDir() {
		t.Error(".expedition should be a directory")
	}
}

func TestInitProject_ConfigFileExists(t *testing.T) {
	// given
	dir := t.TempDir()

	// when — first init succeeds
	err := session.InitProject(dir, "MY", "", io.Discard)
	if err != nil {
		t.Fatalf("first InitProject: %v", err)
	}

	// then — verify config path exists
	cfgPath := domain.ProjectConfigPath(dir)
	if _, statErr := os.Stat(cfgPath); statErr != nil {
		t.Fatalf("config file not created: %v", statErr)
	}
}
