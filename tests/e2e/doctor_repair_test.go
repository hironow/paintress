//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type doctorJSONCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type doctorJSONOutput struct {
	Checks []doctorJSONCheck `json:"checks"`
}

func initPaintressProject(t *testing.T, dir string) {
	t.Helper()
	git := exec.Command("git", "init", dir)
	if err := git.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	cmd := exec.Command("paintress", "init", "--lang", "en", dir)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("paintress init: %v\n%s", err, out)
	}
}

func runDoctorJSON(t *testing.T, dir string, repair bool) doctorJSONOutput {
	t.Helper()
	args := []string{"doctor", "--output", "json"}
	if repair {
		args = append(args, "--repair")
	}
	args = append(args, dir)
	cmd := exec.Command("paintress", args...)
	out, _ := cmd.CombinedOutput()

	// Find JSON object start
	raw := string(out)
	idx := strings.Index(raw, "{")
	if idx < 0 {
		t.Fatalf("no JSON object found in doctor output: %s", raw)
	}
	var result doctorJSONOutput
	if err := json.Unmarshal([]byte(raw[idx:]), &result); err != nil {
		t.Fatalf("failed to parse doctor JSON: %v\nraw: %s", err, raw[idx:])
	}
	return result
}

func TestDoctorRepair_StalePID(t *testing.T) {
	// given: initialized project with stale watch.pid
	dir := t.TempDir()
	initPaintressProject(t, dir)

	pidDir := filepath.Join(dir, ".expedition")
	pidPath := filepath.Join(pidDir, "watch.pid")
	if err := os.WriteFile(pidPath, []byte("99999\n"), 0644); err != nil {
		t.Fatalf("write pid: %v", err)
	}

	// when: run doctor --repair --json
	result := runDoctorJSON(t, dir, true)

	// then: stale-pid check should be FIX
	found := false
	for _, check := range result.Checks {
		if check.Name == "stale-pid" {
			found = true
			if check.Status != "FIX" {
				t.Errorf("stale-pid: expected status FIX, got %s", check.Status)
			}
			if !strings.Contains(check.Message, "removed stale PID") {
				t.Errorf("stale-pid: unexpected message: %s", check.Message)
			}
		}
	}
	if !found {
		t.Errorf("stale-pid check not found in results: %+v", result.Checks)
	}

	// Verify PID file was actually removed
	if _, err := os.Stat(pidPath); err == nil {
		t.Error("watch.pid should have been removed but still exists")
	}
}

func TestDoctorRepair_MissingSkillMD(t *testing.T) {
	// given: initialized project with SKILL.md deleted
	dir := t.TempDir()
	initPaintressProject(t, dir)

	skillsDir := filepath.Join(dir, ".expedition", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("read skills dir: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			skillMD := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
			os.Remove(skillMD)
		}
	}

	// when: run doctor --repair --json
	result := runDoctorJSON(t, dir, true)

	// then: skills check should be FIX (regenerated)
	found := false
	for _, check := range result.Checks {
		if check.Name == "skills" && check.Status == "FIX" {
			found = true
			if !strings.Contains(check.Message, "regenerated") {
				t.Errorf("skills: expected regenerated message, got: %s", check.Message)
			}
		}
	}
	if !found {
		t.Logf("checks: %+v", result.Checks)
		t.Errorf("skills FIX check not found in results")
	}

	// Verify SKILL.md was actually regenerated
	for _, entry := range entries {
		if entry.IsDir() {
			skillMD := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillMD); err != nil {
				t.Errorf("SKILL.md not regenerated: %s", skillMD)
			}
		}
	}
}

func TestDoctorRepair_NoRepairFlag(t *testing.T) {
	// given: initialized project with stale watch.pid but no --repair
	dir := t.TempDir()
	initPaintressProject(t, dir)

	pidPath := filepath.Join(dir, ".expedition", "watch.pid")
	if err := os.WriteFile(pidPath, []byte("99999\n"), 0644); err != nil {
		t.Fatalf("write pid: %v", err)
	}

	// when: run doctor --json WITHOUT --repair
	result := runDoctorJSON(t, dir, false)

	// then: stale-pid check should NOT appear
	for _, check := range result.Checks {
		if check.Name == "stale-pid" {
			t.Errorf("stale-pid check should not appear without --repair flag")
		}
	}

	// Verify PID file still exists
	if _, err := os.Stat(pidPath); err != nil {
		t.Error("watch.pid should still exist but was removed")
	}
}
