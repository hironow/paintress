package paintress

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectConfigPath(t *testing.T) {
	got := ProjectConfigPath("/tmp/repo")
	want := "/tmp/repo/.expedition/config.yaml"
	if got != want {
		t.Errorf("ProjectConfigPath = %q, want %q", got, want)
	}
}

func TestSaveAndLoadProjectConfig(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	cfg := &ProjectConfig{
		Linear: LinearConfig{
			Team:    "MY",
			Project: "paintress",
		},
	}

	if err := SaveProjectConfig(dir, cfg); err != nil {
		t.Fatalf("SaveProjectConfig: %v", err)
	}

	loaded, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}

	if loaded.Linear.Team != "MY" {
		t.Errorf("Team = %q, want %q", loaded.Linear.Team, "MY")
	}
	if loaded.Linear.Project != "paintress" {
		t.Errorf("Project = %q, want %q", loaded.Linear.Project, "paintress")
	}
}

func TestLoadProjectConfig_FileNotFound(t *testing.T) {
	dir := t.TempDir()

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.Linear.Team != "" {
		t.Errorf("Team = %q, want empty", cfg.Linear.Team)
	}
}
