package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hironow/paintress"
)

func TestRunDoctor_GitFound(t *testing.T) {
	// given/when
	checks := RunDoctor("claude", "")
	var gitCheck *struct {
		Name    string
		OK      bool
		Path    string
		Version string
	}
	for i := range checks {
		if checks[i].Name == "git" {
			gitCheck = &struct {
				Name    string
				OK      bool
				Path    string
				Version string
			}{checks[i].Name, checks[i].OK, checks[i].Path, checks[i].Version}
			break
		}
	}

	// then
	if gitCheck == nil {
		t.Fatal("expected git check in results")
	}
	if !gitCheck.OK {
		t.Error("git should be found in test environment")
	}
	if gitCheck.Path == "" {
		t.Error("git path should not be empty")
	}
	if gitCheck.Version == "" {
		t.Error("git version should not be empty")
	}
}

func TestRunDoctor_DockerIsOptional(t *testing.T) {
	// given/when
	checks := RunDoctor("claude", "")

	// then
	for i := range checks {
		if checks[i].Name == "docker" {
			if checks[i].Required {
				t.Error("docker should be optional (Required=false), used only for tracing and container tests")
			}
			return
		}
	}
	t.Fatal("expected docker check in results")
}

func TestRunDoctor_MissingCommand(t *testing.T) {
	// given/when
	checks := RunDoctor("nonexistent-paintress-cmd-12345", "")

	// then
	for i := range checks {
		if checks[i].Name == "nonexistent-paintress-cmd-12345" {
			if checks[i].OK {
				t.Error("nonexistent command should not be OK")
			}
			if checks[i].Path != "" {
				t.Errorf("path should be empty for missing command, got %q", checks[i].Path)
			}
			return
		}
	}
	t.Fatal("expected claude cmd check in results")
}

func TestRunDoctor_CheckContinent_ValidStructure(t *testing.T) {
	// given — valid .expedition/ structure
	dir := t.TempDir()
	for _, sub := range []string{"journal", ".run", "inbox", "outbox", "archive"} {
		os.MkdirAll(filepath.Join(dir, ".expedition", sub), 0755)
	}

	// when
	checks := RunDoctor("claude", dir)

	// then — continent check should be OK
	for _, c := range checks {
		if c.Name == "continent" {
			if !c.OK {
				t.Error("continent check should pass for valid structure")
			}
			if c.Required {
				t.Error("continent check should NOT be required (warning only)")
			}
			return
		}
	}
	t.Error("expected continent check in doctor output")
}

func TestRunDoctor_CheckContinent_MissingDir(t *testing.T) {
	// given — empty continent (no .expedition/)
	dir := t.TempDir()

	// when
	checks := RunDoctor("claude", dir)

	// then — continent check should be NOT OK but NOT required (warning)
	for _, c := range checks {
		if c.Name == "continent" {
			if c.OK {
				t.Error("continent check should fail when .expedition/ is missing")
			}
			if c.Required {
				t.Error("continent check should NOT be required")
			}
			return
		}
	}
	t.Error("expected continent check in doctor output")
}

func TestRunDoctor_CheckContinent_Empty_Skipped(t *testing.T) {
	// given — no continent path provided
	// when
	checks := RunDoctor("claude", "")

	// then — no continent check should appear
	for _, c := range checks {
		if c.Name == "continent" {
			t.Error("continent check should not appear when no continent provided")
		}
	}
}

func TestRunDoctor_CheckConfig_ValidConfig(t *testing.T) {
	// given — valid config.yaml
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)
	os.WriteFile(paintress.ProjectConfigPath(dir), []byte("linear:\n  team: TEST\n"), 0644)

	// when
	checks := RunDoctor("claude", dir)

	// then — config check should be OK
	for _, c := range checks {
		if c.Name == "config" {
			if !c.OK {
				t.Errorf("config check should pass for valid config, version: %s", c.Version)
			}
			if c.Required {
				t.Error("config check should NOT be required")
			}
			return
		}
	}
	t.Error("expected config check in doctor output")
}

func TestRunDoctor_CheckConfig_MissingConfig(t *testing.T) {
	// given — continent exists but no config.yaml
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	// when
	checks := RunDoctor("claude", dir)

	// then — config check should NOT be OK but NOT required (warning)
	for _, c := range checks {
		if c.Name == "config" {
			if c.OK {
				t.Error("config check should fail when config.yaml is missing")
			}
			if c.Required {
				t.Error("config check should NOT be required")
			}
			return
		}
	}
	t.Error("expected config check in doctor output")
}
