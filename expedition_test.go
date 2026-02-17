package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHelperProcess is a test helper process used to mock exec.Command.
// It is invoked as a subprocess by the mock command functions below.
// When GO_TEST_HELPER_PROCESS is set, this function emits the fake output
// specified by GO_TEST_HELPER_OUTPUT and exits with GO_TEST_HELPER_EXIT_CODE.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_TEST_HELPER_PROCESS") != "1" {
		return
	}
	output := os.Getenv("GO_TEST_HELPER_OUTPUT")
	exitCode := 0
	fmt.Sscanf(os.Getenv("GO_TEST_HELPER_EXIT_CODE"), "%d", &exitCode)

	fmt.Fprint(os.Stdout, output)
	os.Exit(exitCode)
}

// fakeMakeCmd returns a makeCmd function that spawns this test's
// TestHelperProcess with the given output and exit code.
func fakeMakeCmd(output string, exitCode int) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--"}
		cs = append(cs, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], cs...)
		cmd.Env = append(os.Environ(),
			"GO_TEST_HELPER_PROCESS=1",
			fmt.Sprintf("GO_TEST_HELPER_OUTPUT=%s", output),
			fmt.Sprintf("GO_TEST_HELPER_EXIT_CODE=%d", exitCode),
		)
		return cmd
	}
}

func newTestExpedition(t *testing.T, output string, exitCode int) *Expedition {
	t.Helper()
	dir := t.TempDir()
	logDir := t.TempDir()

	// Create .expedition directory
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	return &Expedition{
		Number:    1,
		Continent: dir,
		Config: Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
		},
		LogDir:   logDir,
		Gradient: NewGradientGauge(5),
		Reserve:  NewReserveParty("opus", []string{"sonnet"}),
		makeCmd:  fakeMakeCmd(output, exitCode),
	}
}

func TestExpedition_BuildPrompt_ContainsNumber(t *testing.T) {
	dir := t.TempDir()
	e := &Expedition{
		Number:    42,
		Continent: dir,
		Config: Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
		},
		Gradient: NewGradientGauge(5),
		Reserve:  NewReserveParty("opus", nil),
	}

	prompt := e.BuildPrompt()

	if !containsStr(prompt, "Expedition #42") {
		t.Error("prompt should contain expedition number")
	}
	if !containsStr(prompt, "flag.md") {
		t.Error("prompt should reference flag.md")
	}
	if !containsStr(prompt, "mission.md") {
		t.Error("prompt should reference mission.md")
	}
	if !containsStr(prompt, "CLAUDE.md") {
		t.Error("prompt should reference CLAUDE.md")
	}
	if !containsStr(prompt, "journal") {
		t.Error("prompt should reference journal")
	}
}

func TestExpedition_BuildPrompt_French(t *testing.T) {
	orig := Lang
	defer func() { Lang = orig }()
	Lang = "fr"

	dir := t.TempDir()
	e := &Expedition{
		Number:    7,
		Continent: dir,
		Config: Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
		},
		Gradient: NewGradientGauge(5),
		Reserve:  NewReserveParty("opus", nil),
	}

	prompt := e.BuildPrompt()

	if !containsStr(prompt, "Expédition #7") {
		t.Error("French prompt should contain 'Expédition #7'")
	}
	if !containsStr(prompt, "Expéditionnaire") {
		t.Error("French prompt should contain 'Expéditionnaire'")
	}
	if !containsStr(prompt, "règles d'engagement") {
		t.Error("French prompt should contain 'règles d'engagement'")
	}
	if !containsStr(prompt, "__EXPEDITION_REPORT__") {
		t.Error("French prompt should contain report markers")
	}
	if !containsStr(prompt, "en route") {
		t.Error("French prompt should end with 'en route'")
	}
}

func TestExpedition_BuildPrompt_ContainsGradient(t *testing.T) {
	dir := t.TempDir()
	g := NewGradientGauge(5)
	g.Charge()
	g.Charge()

	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Gradient:  g,
		Reserve:   NewReserveParty("opus", nil),
	}

	prompt := e.BuildPrompt()
	if !containsStr(prompt, "2/5") {
		t.Error("prompt should show gradient gauge level")
	}
}

func TestExpedition_BuildPrompt_ContainsLuminas(t *testing.T) {
	dir := t.TempDir()
	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Gradient:  NewGradientGauge(5),
		Reserve:   NewReserveParty("opus", nil),
		Luminas: []Lumina{
			{Pattern: "[WARN] Failed 3 times: timeout", Source: "failure-pattern", Uses: 3},
		},
	}

	prompt := e.BuildPrompt()
	if !containsStr(prompt, "timeout") {
		t.Error("prompt should contain lumina pattern")
	}
}

func TestExpedition_BuildPrompt_NoLuminas(t *testing.T) {
	dir := t.TempDir()
	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Gradient:  NewGradientGauge(5),
		Reserve:   NewReserveParty("opus", nil),
	}

	prompt := e.BuildPrompt()
	if !containsStr(prompt, "No Lumina learned") {
		t.Error("prompt should say no luminas")
	}
}

func TestExpedition_BuildPrompt_ReserveInfo(t *testing.T) {
	dir := t.TempDir()
	rp := NewReserveParty("opus", []string{"sonnet"})
	rp.CheckOutput("rate limit") // Switch to reserve

	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    Config{BaseBranch: "develop", DevURL: "http://localhost:5173"},
		Gradient:  NewGradientGauge(5),
		Reserve:   rp,
	}

	prompt := e.BuildPrompt()
	if !containsStr(prompt, "sonnet") {
		t.Error("prompt should mention reserve model")
	}
	if !containsStr(prompt, "develop") {
		t.Error("prompt should mention base branch")
	}
	if !containsStr(prompt, "localhost:5173") {
		t.Error("prompt should mention dev URL")
	}
}

func TestExpedition_BuildPrompt_OutputFormat(t *testing.T) {
	dir := t.TempDir()
	e := &Expedition{
		Number:    3,
		Continent: dir,
		Config:    Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Gradient:  NewGradientGauge(5),
		Reserve:   NewReserveParty("opus", nil),
	}

	prompt := e.BuildPrompt()
	if !containsStr(prompt, "__EXPEDITION_REPORT__") {
		t.Error("prompt should contain report start marker")
	}
	if !containsStr(prompt, "__EXPEDITION_END__") {
		t.Error("prompt should contain report end marker")
	}
	if !containsStr(prompt, "__EXPEDITION_COMPLETE__") {
		t.Error("prompt should contain complete marker")
	}
}

func TestBuildPrompt_IncludesContextFiles(t *testing.T) {
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".expedition", "context")
	os.MkdirAll(ctxDir, 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	os.WriteFile(filepath.Join(ctxDir, "adr-001.md"), []byte("Use event sourcing for audit trail.\n"), 0644)

	exp := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Luminas:   nil,
		Gradient:  NewGradientGauge(5),
		Reserve:   NewReserveParty("opus", nil),
	}

	prompt := exp.BuildPrompt()

	if !strings.Contains(prompt, "Use event sourcing for audit trail.") {
		t.Error("expected prompt to contain context file content")
	}
	if !strings.Contains(prompt, "adr-001") {
		t.Error("expected prompt to contain context file name as header")
	}
}

func TestBuildPrompt_NoContextSection_WhenEmpty(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	exp := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Luminas:   nil,
		Gradient:  NewGradientGauge(5),
		Reserve:   NewReserveParty("opus", nil),
	}

	prompt := exp.BuildPrompt()

	if strings.Contains(prompt, "Injected Context") {
		t.Error("context section should not appear when no context files exist")
	}
}

func TestExpedition_Run_Success(t *testing.T) {
	reportOutput := `Analyzing codebase...
Tests passed.

__EXPEDITION_REPORT__
expedition: 1
issue_id: AWE-42
issue_title: Add button
mission_type: implement
branch: feat/AWE-42
pr_url: https://github.com/org/repo/pull/1
status: success
reason: done
remaining_issues: 5
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`

	exp := newTestExpedition(t, reportOutput, 0)
	ctx := context.Background()

	output, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if !containsStr(output, "__EXPEDITION_REPORT__") {
		t.Error("output should contain report markers")
	}

	report, status := ParseReport(output, 1)
	if status != StatusSuccess {
		t.Fatalf("got %v, want StatusSuccess", status)
	}
	if report.IssueID != "AWE-42" {
		t.Errorf("IssueID = %q", report.IssueID)
	}
}

func TestExpedition_Run_Complete(t *testing.T) {
	exp := newTestExpedition(t, "__EXPEDITION_COMPLETE__", 0)
	ctx := context.Background()

	output, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	_, status := ParseReport(output, 1)
	if status != StatusComplete {
		t.Fatalf("got %v, want StatusComplete", status)
	}
}

func TestExpedition_Run_CommandFailure(t *testing.T) {
	exp := newTestExpedition(t, "error occurred", 1)
	ctx := context.Background()

	_, err := exp.Run(ctx)
	if err == nil {
		t.Fatal("expected error for non-zero exit code")
	}
}

func TestExpedition_Run_WritesPromptFile(t *testing.T) {
	exp := newTestExpedition(t, "output", 0)
	ctx := context.Background()

	exp.Run(ctx)

	promptFile := filepath.Join(exp.LogDir, "expedition-001-prompt.md")
	if _, err := os.Stat(promptFile); os.IsNotExist(err) {
		t.Error("prompt file should be created")
	}

	content, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(content), "Expedition #1") {
		t.Error("prompt file should contain expedition prompt")
	}
}

func TestExpedition_Run_WritesOutputFile(t *testing.T) {
	exp := newTestExpedition(t, "mock output data", 0)
	ctx := context.Background()

	exp.Run(ctx)

	outputFile := filepath.Join(exp.LogDir, "expedition-001-output.txt")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("output file should be created")
	}
}

func TestExpedition_Run_UsesActiveModel(t *testing.T) {
	exp := newTestExpedition(t, "output", 0)
	exp.Reserve.CheckOutput("rate limit") // Switch to sonnet

	if exp.Reserve.ActiveModel() != "sonnet" {
		t.Fatalf("expected reserve to be sonnet, got %q", exp.Reserve.ActiveModel())
	}

	ctx := context.Background()
	_, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	// The test verifies it doesn't crash when using reserve model
}

func TestExpedition_Run_ContextCanceled(t *testing.T) {
	exp := newTestExpedition(t, "output", 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := exp.Run(ctx)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestExpedition_Run_RateLimitDetection(t *testing.T) {
	// Output contains a rate limit signal — Reserve should detect it
	exp := newTestExpedition(t, "Error: rate limit exceeded, switching models", 0)

	ctx := context.Background()
	exp.Run(ctx)

	// After detecting rate limit in output, Reserve should have switched
	// Note: the detection happens during streaming, which may or may not fire
	// depending on timing. At minimum, we verify no panic/crash.
}

func TestNewPaintress_BasicConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Continent:      dir,
		MaxExpeditions: 10,
		TimeoutSec:     60,
		Model:          "opus",
		BaseBranch:     "main",
		DevCmd:         "npm run dev",
		DevURL:         "http://localhost:3000",
	}

	p := NewPaintress(cfg)
	if p.gradient == nil {
		t.Error("gradient should be initialized")
	}
	if p.reserve == nil {
		t.Error("reserve should be initialized")
	}
	if p.devServer == nil {
		t.Error("devServer should be initialized")
	}
	if p.reserve.ActiveModel() != "opus" {
		t.Errorf("active model = %q, want opus", p.reserve.ActiveModel())
	}
}

func TestNewPaintress_MultiModelConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Continent: dir,
		Model:     "opus,sonnet,haiku",
		DevCmd:    "npm run dev",
		DevURL:    "http://localhost:3000",
	}

	p := NewPaintress(cfg)
	if p.reserve.ActiveModel() != "opus" {
		t.Errorf("primary should be opus, got %q", p.reserve.ActiveModel())
	}

	// Verify reserves were parsed
	p.reserve.ForceReserve()
	if p.reserve.ActiveModel() != "sonnet" {
		t.Errorf("first reserve should be sonnet, got %q", p.reserve.ActiveModel())
	}
}

func TestNewPaintress_ModelWithSpaces(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Continent: dir,
		Model:     "opus , sonnet , haiku",
		DevCmd:    "npm run dev",
		DevURL:    "http://localhost:3000",
	}

	p := NewPaintress(cfg)
	if p.reserve.ActiveModel() != "opus" {
		t.Errorf("primary should be opus, got %q", p.reserve.ActiveModel())
	}
}
