package main

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

func TestLuminaPath(t *testing.T) {
	p := LuminaPath("/some/repo")
	want := filepath.Join("/some/repo", ".expedition", "lumina.md")
	if p != want {
		t.Errorf("LuminaPath = %q, want %q", p, want)
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
		{Pattern: "[WARN] Failed 3 times: tests failing", Source: "failure-pattern", Uses: 3},
		{Pattern: "[OK] implement mission: 4 proven successes", Source: "success-pattern", Uses: 4},
	}
	result := FormatLuminaForPrompt(luminas)

	if !containsStr(result, "tests failing") {
		t.Errorf("should contain failure pattern: %q", result)
	}
	if !containsStr(result, "implement mission") {
		t.Errorf("should contain success pattern: %q", result)
	}
}

func TestWriteLumina_Empty(t *testing.T) {
	err := WriteLumina("/tmp/test", nil)
	if err != nil {
		t.Errorf("empty luminas should return nil, got %v", err)
	}
}

func TestWriteLumina_WritesFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	luminas := []Lumina{
		{Pattern: "[WARN] Failed 2 times: timeout", Source: "failure-pattern", Uses: 2},
		{Pattern: "[OK] fix mission: 3 proven successes", Source: "success-pattern", Uses: 3},
	}

	err := WriteLumina(dir, luminas)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(LuminaPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	if !containsStr(s, "Learned Passive Skills") {
		t.Error("should contain header")
	}
	if !containsStr(s, "Defensive") {
		t.Error("should contain defensive section")
	}
	if !containsStr(s, "Offensive") {
		t.Error("should contain offensive section")
	}
	if !containsStr(s, "timeout") {
		t.Error("should contain failure pattern")
	}
	if !containsStr(s, "fix mission") {
		t.Error("should contain success pattern")
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
		if containsStr(l.Pattern, "verify mission") {
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
		if containsStr(l.Pattern, "implement mission") {
			hasSuccess = true
		}
	}

	if !hasFailure {
		t.Error("should have failure lumina for 'lint error'")
	}
	if !hasSuccess {
		t.Error("should have success lumina for 'implement mission'")
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
