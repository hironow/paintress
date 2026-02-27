package session

import (
	"testing"
)

func TestRunDoctor_GitFound(t *testing.T) {
	checks := RunDoctor("claude")
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
	checks := RunDoctor("claude")
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
	checks := RunDoctor("nonexistent-paintress-cmd-12345")
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
