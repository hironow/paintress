package paintress

import (
	"testing"
)

func TestRunDoctor_GitFound(t *testing.T) {
	checks := RunDoctor("claude")
	var gitCheck *DoctorCheck
	for i := range checks {
		if checks[i].Name == "git" {
			gitCheck = &checks[i]
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
	var dockerCheck *DoctorCheck
	for i := range checks {
		if checks[i].Name == "docker" {
			dockerCheck = &checks[i]
			break
		}
	}
	if dockerCheck == nil {
		t.Fatal("expected docker check in results")
	}
	if dockerCheck.Required {
		t.Error("docker should be optional (Required=false), used only for tracing and container tests")
	}
}

func TestRunDoctor_MissingCommand(t *testing.T) {
	checks := RunDoctor("nonexistent-paintress-cmd-12345")
	var found *DoctorCheck
	for i := range checks {
		if checks[i].Name == "nonexistent-paintress-cmd-12345" {
			found = &checks[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected claude cmd check in results")
	}
	if found.OK {
		t.Error("nonexistent command should not be OK")
	}
	if found.Path != "" {
		t.Errorf("path should be empty for missing command, got %q", found.Path)
	}
}
