package paintress

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractValue_Normal(t *testing.T) {
	v := extractValue("- **Status**: success")
	if v != "success" {
		t.Errorf("got %q, want 'success'", v)
	}
}

func TestExtractValue_WithBoldMarkers(t *testing.T) {
	v := extractValue("- **Status**: **failed**")
	if v != "failed" {
		t.Errorf("got %q, want 'failed'", v)
	}
}

func TestExtractValue_NoColon(t *testing.T) {
	v := extractValue("no colon here")
	if v != "" {
		t.Errorf("got %q, want empty", v)
	}
}

func TestExtractValue_EmptyValue(t *testing.T) {
	v := extractValue("- **Status**:")
	if v != "" {
		t.Errorf("got %q, want empty", v)
	}
}

func TestExtractValue_ValueWithColons(t *testing.T) {
	v := extractValue("- **Reason**: test failed: assertion error: expected 5")
	// SplitN with 2 keeps everything after first ":"
	if v != "test failed: assertion error: expected 5" {
		t.Errorf("got %q", v)
	}
}

func TestFormatLuminaForPrompt_Empty(t *testing.T) {
	result := FormatLuminaForPrompt(nil)
	if !containsStr(result, "No Lumina learned") {
		t.Errorf("empty luminas should say no luminas: %q", result)
	}
}

func TestFormatLuminaForPrompt_WithLuminas(t *testing.T) {
	luminas := []Lumina{
		{Pattern: "[WARN] Avoid — failed 3 times: tests failing", Source: "failure-pattern", Uses: 3},
		{Pattern: "[OK] Proven approach (4x successful): implement", Source: "success-pattern", Uses: 4},
	}
	result := FormatLuminaForPrompt(luminas)

	if !containsStr(result, "Defensive") {
		t.Errorf("should contain Defensive section header: %q", result)
	}
	if !containsStr(result, "Offensive") {
		t.Errorf("should contain Offensive section header: %q", result)
	}
	if !containsStr(result, "tests failing") {
		t.Errorf("should contain failure pattern: %q", result)
	}
	if !containsStr(result, "implement") {
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

	luminas := ScanJournalsForLumina(dir)
	for _, l := range luminas {
		if containsStr(l.Pattern, "test failed") {
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

	luminas := ScanJournalsForLumina(dir)
	for _, l := range luminas {
		if containsStr(l.Pattern, "verify") {
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

	luminas := ScanJournalsForLumina(dir)

	hasFailure := false
	hasSuccess := false
	for _, l := range luminas {
		if containsStr(l.Pattern, "lint error") {
			hasFailure = true
		}
		if containsStr(l.Pattern, "implement") && containsStr(l.Pattern, "Proven approach") {
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

	luminas := ScanJournalsForLumina(dir)
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

	luminas := ScanJournalsForLumina(dir)

	hasInsight := false
	for _, l := range luminas {
		if containsStr(l.Pattern, "Redis connection required") {
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

	luminas := ScanJournalsForLumina(dir)

	hasInsight := false
	for _, l := range luminas {
		if containsStr(l.Pattern, "TDD cycle with bun test") {
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

	luminas := ScanJournalsForLumina(dir)

	hasReason := false
	for _, l := range luminas {
		if containsStr(l.Pattern, "lint error") {
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

	luminas := ScanJournalsForLumina(dir)

	hasMission := false
	for _, l := range luminas {
		if containsStr(l.Pattern, "implement") {
			hasMission = true
		}
	}
	if !hasMission {
		t.Error("should fall back to mission when insight is empty")
	}
}
