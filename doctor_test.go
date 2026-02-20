package paintress

import (
	"encoding/json"
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

func TestDoctorCheck_JSONMarshal(t *testing.T) {
	// given
	check := DoctorCheck{
		Name:     "git",
		Required: true,
		Path:     "/usr/bin/git",
		Version:  "git version 2.53.0",
		OK:       true,
	}

	// when
	data, err := json.Marshal(check)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// then — verify lowercase JSON keys
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"name", "required", "path", "version", "ok"} {
		if _, exists := m[key]; !exists {
			t.Errorf("expected JSON key %q, got keys: %v", key, m)
		}
	}
	// Ensure no uppercase keys leaked through
	for _, key := range []string{"Name", "Required", "Path", "Version", "OK"} {
		if _, exists := m[key]; exists {
			t.Errorf("unexpected uppercase JSON key %q — add json struct tags", key)
		}
	}
}

func TestFormatDoctorJSON(t *testing.T) {
	// given
	checks := []DoctorCheck{
		{Name: "git", Required: true, OK: true, Path: "/usr/bin/git", Version: "git version 2.53.0"},
		{Name: "docker", Required: false, OK: false},
	}

	// when
	out, err := FormatDoctorJSON(checks)
	if err != nil {
		t.Fatalf("FormatDoctorJSON: %v", err)
	}

	// then — must be valid JSON array
	var parsed []DoctorCheck
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, out)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 checks, got %d", len(parsed))
	}
	if parsed[0].Name != "git" || !parsed[0].OK {
		t.Errorf("unexpected first check: %+v", parsed[0])
	}
	if parsed[1].Name != "docker" || parsed[1].OK {
		t.Errorf("unexpected second check: %+v", parsed[1])
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
