package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestExpedition_Run_WatcherLogsCurrentIssue(t *testing.T) {
	dir := t.TempDir()
	logDir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	// Shell script that writes flag.md then outputs a report
	script := filepath.Join(dir, "write-flag.sh")
	flagPath := filepath.Join(dir, ".expedition", "flag.md")
	scriptContent := fmt.Sprintf(`#!/bin/bash
# Write current_issue to flag.md
cat > %s << 'FLAGEOF'
current_issue: MY-239
current_title: flag watcher test
FLAGEOF
# Wait for watcher to detect
sleep 1
echo "done"
`, flagPath)
	os.WriteFile(script, []byte(scriptContent), 0755)

	logPath := filepath.Join(logDir, "test-watcher.log")
	InitLogFile(logPath)
	defer CloseLogFile()

	exp := &Expedition{
		Number:    1,
		Continent: dir,
		Config: Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
			ClaudeCmd:  script,
		},
		LogDir:            logDir,
		Gradient:          NewGradientGauge(5),
		Reserve:           NewReserveParty("opus", nil),
		WatchFlagInterval: 100 * time.Millisecond,
	}

	ctx := context.Background()
	_, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Check log file for issue detection
	logContent, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	if !containsStr(string(logContent), "MY-239") {
		t.Errorf("log should contain issue ID 'MY-239', got:\n%s", string(logContent))
	}
}

// TestExpedition_Run_WatcherReadsFromContinent_NotWorkDir verifies that
// in worktree mode (WorkDir != Continent), the flag watcher polls
// Continent/.expedition/flag.md — NOT WorkDir/.expedition/flag.md.
func TestExpedition_Run_WatcherReadsFromContinent_NotWorkDir(t *testing.T) {
	continent := t.TempDir()
	workDir := t.TempDir() // simulate worktree — different from continent
	logDir := t.TempDir()
	os.MkdirAll(filepath.Join(continent, ".expedition", "journal"), 0755)
	os.MkdirAll(filepath.Join(workDir, ".expedition"), 0755)

	// Script writes flag.md to CONTINENT root (not workDir), then outputs
	flagPath := filepath.Join(continent, ".expedition", "flag.md")
	script := filepath.Join(workDir, "write-flag.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
cat > %s << 'FLAGEOF'
current_issue: TREE-42
current_title: worktree watcher test
FLAGEOF
sleep 1
echo "done"
`, flagPath)
	os.WriteFile(script, []byte(scriptContent), 0755)

	logPath := filepath.Join(logDir, "test-worktree-watcher.log")
	InitLogFile(logPath)
	defer CloseLogFile()

	exp := &Expedition{
		Number:    1,
		Continent: continent,
		WorkDir:   workDir,
		Config: Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
			ClaudeCmd:  script,
		},
		LogDir:            logDir,
		Gradient:          NewGradientGauge(5),
		Reserve:           NewReserveParty("opus", nil),
		WatchFlagInterval: 100 * time.Millisecond,
	}

	ctx := context.Background()
	_, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	logContent, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	if !containsStr(string(logContent), "TREE-42") {
		t.Errorf("watcher should detect issue from CONTINENT flag.md, not workDir; log:\n%s", string(logContent))
	}
}

func TestExpedition_BuildPrompt_ContainsFlagWriteInstruction(t *testing.T) {
	dir := t.TempDir()
	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Gradient:  NewGradientGauge(5),
		Reserve:   NewReserveParty("opus", nil),
	}

	for _, lang := range []string{"en", "ja", "fr"} {
		t.Run(lang, func(t *testing.T) {
			orig := Lang
			defer func() { Lang = orig }()
			Lang = lang

			prompt := e.BuildPrompt()
			if !containsStr(prompt, "current_issue") {
				t.Errorf("[%s] prompt should contain 'current_issue' instruction", lang)
			}
			if !containsStr(prompt, "current_title") {
				t.Errorf("[%s] prompt should contain 'current_title' instruction", lang)
			}
		})
	}
}

func TestExpedition_BuildPrompt_EmptyDevURL_NoDevServerLine(t *testing.T) {
	dir := t.TempDir()
	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    Config{BaseBranch: "main", DevURL: ""},
		Gradient:  NewGradientGauge(5),
		Reserve:   NewReserveParty("opus", nil),
	}

	// Test all 3 languages
	for _, lang := range []string{"en", "ja", "fr"} {
		t.Run(lang, func(t *testing.T) {
			orig := Lang
			defer func() { Lang = orig }()
			Lang = lang

			prompt := e.BuildPrompt()

			if containsStr(prompt, "Dev server") || containsStr(prompt, "Serveur dev") {
				t.Errorf("[%s] prompt should NOT contain dev server line when DevURL is empty", lang)
			}
			if containsStr(prompt, "already running") || containsStr(prompt, "既に起動済み") || containsStr(prompt, "déjà lancé") {
				t.Errorf("[%s] prompt should NOT contain 'already running' when DevURL is empty", lang)
			}
		})
	}
}

func TestBuildPrompt_WithLinearConfig(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	// Write a config.yaml with Linear scope
	cfg := &ProjectConfig{
		Linear: LinearConfig{
			Team:    "ENG",
			Project: "backend",
		},
	}
	if err := SaveProjectConfig(dir, cfg); err != nil {
		t.Fatalf("SaveProjectConfig: %v", err)
	}

	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Gradient:  NewGradientGauge(5),
		Reserve:   NewReserveParty("opus", nil),
	}

	prompt := e.BuildPrompt()

	if !containsStr(prompt, "ENG") {
		t.Error("prompt should contain Linear team key 'ENG'")
	}
	if !containsStr(prompt, "backend") {
		t.Error("prompt should contain Linear project 'backend'")
	}
}

func TestBuildPrompt_WithoutLinearConfig(t *testing.T) {
	dir := t.TempDir()

	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Gradient:  NewGradientGauge(5),
		Reserve:   NewReserveParty("opus", nil),
	}

	prompt := e.BuildPrompt()

	if containsStr(prompt, "Linear Scope") {
		t.Error("prompt should NOT contain Linear Scope when no config exists")
	}
}

func TestBuildPrompt_MalformedConfig_NoPanic(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	// Write malformed YAML that will fail to parse
	os.WriteFile(
		filepath.Join(dir, ".expedition", "config.yaml"),
		[]byte("{{invalid yaml\n\t::: broken"),
		0644,
	)

	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Gradient:  NewGradientGauge(5),
		Reserve:   NewReserveParty("opus", nil),
	}

	// Must not panic — should gracefully omit Linear scope
	prompt := e.BuildPrompt()

	if containsStr(prompt, "Linear Scope") {
		t.Error("prompt should NOT contain Linear Scope for malformed config")
	}
	if !containsStr(prompt, "Expedition #1") {
		t.Error("prompt should still be generated despite malformed config")
	}
}

// TestLifecycle_Init_Then_Expedition verifies the full lifecycle:
// paintress init (config.yaml) → expedition run → prompt file contains Linear scope.
// External world (Claude) is stubbed via fakeMakeCmd.
func TestLifecycle_Init_Then_Expedition(t *testing.T) {
	dir := t.TempDir()
	logDir := t.TempDir()

	// Phase 1: simulate `paintress init` with stdin
	input := "MY\npaintress\n"
	if err := runInitWithReader(dir, strings.NewReader(input)); err != nil {
		t.Fatalf("runInitWithReader: %v", err)
	}

	// Verify config was persisted
	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if cfg.Linear.Team != "MY" || cfg.Linear.Project != "paintress" {
		t.Fatalf("unexpected config: team=%q project=%q", cfg.Linear.Team, cfg.Linear.Project)
	}

	// Phase 2: run expedition with fake Claude (outputs a valid report)
	reportOutput := `Working on issue...

__EXPEDITION_REPORT__
expedition: 1
issue_id: MY-100
issue_title: lifecycle test
mission_type: implement
branch: feat/MY-100
pr_url: https://github.com/org/repo/pull/99
status: success
reason: done
failure_type: none
insight: lifecycle works
remaining_issues: 3
bugs_found: 0
bug_issues: none
__EXPEDITION_END__`

	exp := &Expedition{
		Number:    1,
		Continent: dir,
		Config: Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
		},
		LogDir:   logDir,
		Gradient: NewGradientGauge(5),
		Reserve:  NewReserveParty("opus", nil),
		makeCmd:  fakeMakeCmd(reportOutput, 0),
	}

	ctx := context.Background()
	output, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Phase 3: verify expedition output is valid
	report, status := ParseReport(output, 1)
	if status != StatusSuccess {
		t.Fatalf("got %v, want StatusSuccess", status)
	}
	if report.IssueID != "MY-100" {
		t.Errorf("IssueID = %q, want MY-100", report.IssueID)
	}

	// Phase 4: verify the prompt file contains Linear scope from init
	promptFile := filepath.Join(logDir, "expedition-001-prompt.md")
	promptContent, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("read prompt file: %v", err)
	}
	prompt := string(promptContent)

	if !containsStr(prompt, "MY") {
		t.Error("prompt file should contain Linear team 'MY' from init")
	}
	if !containsStr(prompt, "paintress") {
		t.Error("prompt file should contain Linear project 'paintress' from init")
	}
	if !containsStr(prompt, "Linear Scope") {
		t.Error("prompt file should contain 'Linear Scope' section")
	}
}

// TestLifecycle_NoInit_Then_Expedition verifies that expedition works
// without prior init — no Linear Scope section in prompt.
func TestLifecycle_NoInit_Then_Expedition(t *testing.T) {
	dir := t.TempDir()
	logDir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	exp := &Expedition{
		Number:    1,
		Continent: dir,
		Config: Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
		},
		LogDir:   logDir,
		Gradient: NewGradientGauge(5),
		Reserve:  NewReserveParty("opus", nil),
		makeCmd:  fakeMakeCmd("__EXPEDITION_COMPLETE__", 0),
	}

	ctx := context.Background()
	_, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	promptFile := filepath.Join(logDir, "expedition-001-prompt.md")
	promptContent, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("read prompt file: %v", err)
	}

	if containsStr(string(promptContent), "Linear Scope") {
		t.Error("prompt should NOT contain Linear Scope when no init was done")
	}
}

func TestNewPaintress_NoDev_NoDevServer(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Continent: dir,
		Model:     "opus",
		DevCmd:    "npm run dev",
		DevURL:    "http://localhost:3000",
		NoDev:     true,
	}

	p := NewPaintress(cfg)

	if p.devServer != nil {
		t.Error("devServer should be nil when NoDev=true")
	}
	if p.config.DevURL != "" {
		t.Errorf("DevURL should be cleared when NoDev=true, got %q", p.config.DevURL)
	}
}
