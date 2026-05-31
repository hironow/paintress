//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
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

func runDoctorJSON(t *testing.T, ctx context.Context, c testcontainers.Container, dir string, repair bool) doctorJSONOutput {
	t.Helper()
	args := []string{"doctor", "--output", "json"}
	if repair {
		args = append(args, "--repair")
	}
	args = append(args, dir)
	stdout, _, err := runCmd(t, ctx, c, dir, args...)
	if err != nil {
		t.Logf("doctor returned non-zero (expected for failed checks): %v", err)
	}

	var result doctorJSONOutput
	parseJSONOutput(t, stdout, &result)
	return result
}

func TestDoctorRepair_StalePID(t *testing.T) {
	ctx := context.Background()
	c := buildTestContainer(t, ctx)
	dir := "/workspace/t_doctor_stale"
	initTestRepo(t, ctx, c, dir)

	pidPath := fmt.Sprintf("%s/.expedition/watch.pid", dir)
	execInContainer(t, ctx, c, []string{"sh", "-c", fmt.Sprintf("echo '99999' > %s", pidPath)})

	// when: run doctor --repair --json
	result := runDoctorJSON(t, ctx, c, dir, true)

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
	if fileExistsInContainer(t, ctx, c, pidPath) {
		t.Error("watch.pid should have been removed but still exists")
	}
}

func TestDoctorRepair_MissingSkillMD(t *testing.T) {
	ctx := context.Background()
	c := buildTestContainer(t, ctx)
	dir := "/workspace/t_doctor_skills"
	initTestRepo(t, ctx, c, dir)

	skillsDir := fmt.Sprintf("%s/.expedition/skills", dir)
	
	// List skills inside container to find subdirectory names (like "dmail-sendable")
	// For testing, we can just delete SKILL.md under all subdirectories of skills directory
	execInContainer(t, ctx, c, []string{"sh", "-c", fmt.Sprintf("rm -f %s/*/SKILL.md", skillsDir)})

	// when: run doctor --repair --json
	result := runDoctorJSON(t, ctx, c, dir, true)

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

	// Verify SKILL.md was actually regenerated (we can check a few known ones or just that they exist)
	expectedSkills := []string{"dmail-sendable", "dmail-readable"}
	for _, skill := range expectedSkills {
		path := fmt.Sprintf("%s/%s/SKILL.md", skillsDir, skill)
		if !fileExistsInContainer(t, ctx, c, path) {
			t.Errorf("SKILL.md not regenerated for %s: %s", skill, path)
		}
	}
}

func TestDoctorRepair_NoRepairFlag(t *testing.T) {
	ctx := context.Background()
	c := buildTestContainer(t, ctx)
	dir := "/workspace/t_doctor_norepair"
	initTestRepo(t, ctx, c, dir)

	pidPath := fmt.Sprintf("%s/.expedition/watch.pid", dir)
	execInContainer(t, ctx, c, []string{"sh", "-c", fmt.Sprintf("echo '99999' > %s", pidPath)})

	// when: run doctor --json WITHOUT --repair
	result := runDoctorJSON(t, ctx, c, dir, false)

	// then: stale-pid check should NOT appear
	for _, check := range result.Checks {
		if check.Name == "stale-pid" {
			t.Errorf("stale-pid check should not appear without --repair flag")
		}
	}

	// Verify PID file still exists
	if !fileExistsInContainer(t, ctx, c, pidPath) {
		t.Error("watch.pid should still exist but was removed")
	}
}
