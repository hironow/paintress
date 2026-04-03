package session_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness/policy"
	"github.com/hironow/paintress/internal/session"
)

func TestExtractValue_Normal(t *testing.T) {
	v := session.ExportExtractValue("- **Status**: success")
	if v != "success" {
		t.Errorf("got %q, want 'success'", v)
	}
}

func TestExtractValue_WithBoldMarkers(t *testing.T) {
	v := session.ExportExtractValue("- **Status**: **failed**")
	if v != "failed" {
		t.Errorf("got %q, want 'failed'", v)
	}
}

func TestExtractValue_NoColon(t *testing.T) {
	v := session.ExportExtractValue("no colon here")
	if v != "" {
		t.Errorf("got %q, want empty", v)
	}
}

func TestExtractValue_EmptyValue(t *testing.T) {
	v := session.ExportExtractValue("- **Status**:")
	if v != "" {
		t.Errorf("got %q, want empty", v)
	}
}

func TestExtractValue_ValueWithColons(t *testing.T) {
	v := session.ExportExtractValue("- **Reason**: test failed: assertion error: expected 5")
	// SplitN with 2 keeps everything after first ":"
	if v != "test failed: assertion error: expected 5" {
		t.Errorf("got %q", v)
	}
}

func TestFormatLuminaForPrompt_Empty(t *testing.T) {
	result := policy.FormatLuminaForPrompt(nil)
	if !strings.Contains(result, "No Lumina learned") {
		t.Errorf("empty luminas should say no luminas: %q", result)
	}
}

func TestFormatLuminaForPrompt_WithLuminas(t *testing.T) {
	luminas := []domain.Lumina{
		{Pattern: "[WARN] Avoid — failed 3 times: tests failing", Source: "failure-pattern", Uses: 3},
		{Pattern: "[OK] Proven approach (4x successful): implement", Source: "success-pattern", Uses: 4},
	}
	result := policy.FormatLuminaForPrompt(luminas)

	if !strings.Contains(result, "Defensive") {
		t.Errorf("should contain Defensive section header: %q", result)
	}
	if !strings.Contains(result, "Offensive") {
		t.Errorf("should contain Offensive section header: %q", result)
	}
	if !strings.Contains(result, "tests failing") {
		t.Errorf("should contain failure pattern: %q", result)
	}
	if !strings.Contains(result, "implement") {
		t.Errorf("should contain success pattern: %q", result)
	}
}

func TestScanJournalsForLumina_FailureThreshold(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// Only 1 failure — should NOT become a lumina (threshold is 2)
	os.WriteFile(filepath.Join(jDir, "001.md"), []byte(`# Expedition #1 — Journal

- **Status**: failed
- **Reason**: test failed
- **Mission**: implement
`), 0644)

	luminas := session.ScanJournalsForLumina(dir)
	for _, l := range luminas {
		if strings.Contains(l.Pattern, "test failed") {
			t.Error("single failure should not create lumina")
		}
	}
}

func TestScanJournalsForLumina_SuccessThreshold(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// Only 2 successes — should NOT become a lumina (threshold is 3)
	for i := 1; i <= 2; i++ {
		content := `# Expedition

- **Status**: success
- **Mission**: verify
`
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", i)), []byte(content), 0644)
	}

	luminas := session.ScanJournalsForLumina(dir)
	for _, l := range luminas {
		if strings.Contains(l.Pattern, "verify") {
			t.Error("2 successes should not reach threshold of 3")
		}
	}
}

func TestScanJournalsForLumina_MixedStatuses(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// 3 failures with same reason + 3 successes with same mission
	for i := 1; i <= 3; i++ {
		content := `# Expedition

- **Status**: failed
- **Reason**: lint error
- **Mission**: fix
`
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", i)), []byte(content), 0644)
	}
	for i := 4; i <= 6; i++ {
		content := `# Expedition

- **Status**: success
- **Mission**: implement
`
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", i)), []byte(content), 0644)
	}

	luminas := session.ScanJournalsForLumina(dir)

	hasFailure := false
	hasSuccess := false
	for _, l := range luminas {
		if strings.Contains(l.Pattern, "lint error") {
			hasFailure = true
		}
		if strings.Contains(l.Pattern, "implement") && strings.Contains(l.Pattern, "Proven approach") {
			hasSuccess = true
		}
	}

	if !hasFailure {
		t.Error("should have failure lumina for 'lint error'")
	}
	if !hasSuccess {
		t.Error("should have success lumina for 'implement'")
	}
}

func TestScanJournalsForLumina_DifferentFailureReasons(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// Different failure reasons — none should reach threshold
	reasons := []string{"timeout", "lint error", "test failure"}
	for i, reason := range reasons {
		content := fmt.Sprintf(`# Expedition

- **Status**: failed
- **Reason**: %s
- **Mission**: fix
`, reason)
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", i+1)), []byte(content), 0644)
	}

	luminas := session.ScanJournalsForLumina(dir)
	if len(luminas) != 0 {
		t.Errorf("different failure reasons should not create luminas, got %d", len(luminas))
	}
}

func TestScanJournalsForLumina_InsightUsedForDefensive(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// 2 failures with same insight — should become a defensive lumina using insight text
	for i := 1; i <= 2; i++ {
		content := fmt.Sprintf(`# Expedition #%d — Journal

- **Status**: failed
- **Reason**: test timeout
- **Mission**: implement
- **Insight**: Redis connection required for auth tests but REDIS_URL not set
`, i)
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", i)), []byte(content), 0644)
	}

	luminas := session.ScanJournalsForLumina(dir)

	hasInsight := false
	for _, l := range luminas {
		if strings.Contains(l.Pattern, "Redis connection required") {
			hasInsight = true
		}
		// Should NOT contain the raw reason "test timeout" as the pattern
		if l.Pattern == "[WARN] Avoid — failed 2 times: test timeout" {
			t.Error("should use insight text, not reason text")
		}
	}
	if !hasInsight {
		t.Error("should have defensive lumina with insight text")
	}
}

func TestScanJournalsForLumina_InsightUsedForOffensive(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// 3 successes with same insight — should become an offensive lumina using insight text
	for i := 1; i <= 3; i++ {
		content := fmt.Sprintf(`# Expedition #%d — Journal

- **Status**: success
- **Mission**: implement
- **Insight**: TDD cycle with bun test works reliably for this codebase
`, i)
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", i)), []byte(content), 0644)
	}

	luminas := session.ScanJournalsForLumina(dir)

	hasInsight := false
	for _, l := range luminas {
		if strings.Contains(l.Pattern, "TDD cycle with bun test") {
			hasInsight = true
		}
		// Should NOT contain the raw mission type as the pattern
		if l.Pattern == "[OK] Proven approach (3x successful): implement" {
			t.Error("should use insight text, not mission type")
		}
	}
	if !hasInsight {
		t.Error("should have offensive lumina with insight text")
	}
}

func TestScanJournalsForLumina_FallbackToReasonWhenNoInsight(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// 2 failures WITHOUT insight — should fall back to reason
	for i := 1; i <= 2; i++ {
		content := fmt.Sprintf(`# Expedition #%d — Journal

- **Status**: failed
- **Reason**: lint error
- **Mission**: fix
`, i)
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", i)), []byte(content), 0644)
	}

	luminas := session.ScanJournalsForLumina(dir)

	hasReason := false
	for _, l := range luminas {
		if strings.Contains(l.Pattern, "lint error") {
			hasReason = true
		}
	}
	if !hasReason {
		t.Error("should fall back to reason when insight is empty")
	}
}

func TestScanJournalsForLumina_FallbackToMissionWhenNoInsight(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// 3 successes WITHOUT insight — should fall back to mission type
	for i := 1; i <= 3; i++ {
		content := fmt.Sprintf(`# Expedition #%d — Journal

- **Status**: success
- **Mission**: implement
`, i)
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", i)), []byte(content), 0644)
	}

	luminas := session.ScanJournalsForLumina(dir)

	hasMission := false
	for _, l := range luminas {
		if strings.Contains(l.Pattern, "implement") {
			hasMission = true
		}
	}
	if !hasMission {
		t.Error("should fall back to mission when insight is empty")
	}
}

func TestScanJournalsForLumina_HighSeverityAlert(t *testing.T) {
	// given — 1 journal with HIGH severity D-Mail recorded (threshold = 1)
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	content := `# Expedition #1 — Journal

- **Status**: success
- **Mission**: implement
- **HIGH severity D-Mail**: alert-critical, alert-deploy
`
	os.WriteFile(filepath.Join(jDir, "001.md"), []byte(content), 0644)

	// when
	luminas := session.ScanJournalsForLumina(dir)

	// then
	hasAlert := false
	for _, l := range luminas {
		if l.Source == "high-severity-alert" {
			hasAlert = true
			if !strings.Contains(l.Pattern, "alert-critical, alert-deploy") {
				t.Errorf("alert lumina should contain d-mail names, got: %q", l.Pattern)
			}
		}
	}
	if !hasAlert {
		t.Errorf("expected high-severity-alert lumina, got: %v", luminas)
	}
}

func TestScanJournalsForLumina_NoHighSeverity(t *testing.T) {
	// given — journals without HIGH severity D-Mail
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	content := `# Expedition #1 — Journal

- **Status**: success
- **Mission**: implement
- **HIGH severity D-Mail**:
`
	os.WriteFile(filepath.Join(jDir, "001.md"), []byte(content), 0644)

	// when
	luminas := session.ScanJournalsForLumina(dir)

	// then
	for _, l := range luminas {
		if l.Source == "high-severity-alert" {
			t.Errorf("empty HIGH severity D-Mail should not create alert lumina, got: %v", l)
		}
	}
}

func TestScanJournalsForLumina_CappedAt10(t *testing.T) {
	// given: create enough journals to produce more than 10 luminas
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// Create 2 high-severity alerts, 12 distinct failure patterns (each appearing 2x = 24 journals),
	// producing 2 + 12 = 14 luminas without cap.
	idx := 1

	// 2 high-severity alerts (1 journal each)
	for i := range 2 {
		content := fmt.Sprintf(`# Expedition #%d — Journal

- **Status**: failed
- **Reason**: alert-reason-%d
- **HIGH severity D-Mail**: critical-alert-%d
`, idx, i, i)
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", idx)), []byte(content), 0644)
		idx++
	}

	// 12 distinct failure patterns, each appearing exactly 2 times (threshold)
	for i := range 12 {
		for j := range 2 {
			content := fmt.Sprintf(`# Expedition #%d — Journal

- **Status**: failed
- **Reason**: failure-pattern-%d
- **Insight**: repeated-failure-%d
`, idx, i, i)
			_ = j
			os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", idx)), []byte(content), 0644)
			idx++
		}
	}

	// when
	luminas := session.ScanJournalsForLumina(dir)

	// then: should be capped at 10
	if len(luminas) > 10 {
		t.Errorf("expected at most 10 luminas, got %d", len(luminas))
	}

	// then: all high-severity alerts should be included (highest priority)
	alertCount := 0
	for _, l := range luminas {
		if l.Source == "high-severity-alert" {
			alertCount++
		}
	}
	if alertCount != 2 {
		t.Errorf("expected 2 high-severity alerts in capped result, got %d", alertCount)
	}
}

func TestFormatLuminaForPrompt_WithAlert(t *testing.T) {
	// given
	luminas := []domain.Lumina{
		{Pattern: "[ALERT] HIGH severity D-Mail in past expedition: alert-critical", Source: "high-severity-alert", Uses: 1},
		{Pattern: "[WARN] Avoid — failed 2 times: lint error", Source: "failure-pattern", Uses: 2},
	}

	// when
	result := policy.FormatLuminaForPrompt(luminas)

	// then
	if !strings.Contains(result, "Alert") {
		t.Errorf("should contain Alert section header, got: %q", result)
	}
	if !strings.Contains(result, "alert-critical") {
		t.Errorf("should contain alert d-mail name, got: %q", result)
	}
	if !strings.Contains(result, "Defensive") {
		t.Errorf("should still contain Defensive section, got: %q", result)
	}
}
