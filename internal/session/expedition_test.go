package session

// white-box-reason: session internals: tests unexported localGitExecutor mock process

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness"
	"github.com/hironow/paintress/internal/platform"
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

	// Emit stream-json NDJSON: assistant message with text, then result message.
	assistantMsg := map[string]any{
		"type": "assistant",
		"message": map[string]any{
			"id":    "msg_test",
			"role":  "assistant",
			"model": "claude-sonnet-4-20250514",
			"content": []map[string]any{
				{"type": "text", "text": output},
			},
		},
	}
	resultMsg := map[string]any{
		"type":   "result",
		"result": output,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(assistantMsg)
	enc.Encode(resultMsg)
	os.Exit(exitCode)
}

// fakeMakeCmd returns a makeCmd function that spawns this test's
// TestHelperProcess with the given output and exit code.
func fakeMakeCmd(output string, exitCode int) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--"}
		cs = append(cs, args...)
		// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command -- standard Go subprocess test pattern: os.Args[0] is this test binary, cs is a fixed test-driver flag list [permanent]
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
		Config: domain.Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
		},
		LogDir:   logDir,
		Logger:   platform.NewLogger(io.Discard, false),
		Gradient: harness.NewGradientGauge(5),
		Reserve:  harness.NewReserveParty("opus", []string{"sonnet"}, platform.NewLogger(io.Discard, false)),
		makeCmd:  fakeMakeCmd(output, exitCode),
	}
}

func TestExpedition_BuildPrompt_ContainsNumber(t *testing.T) {
	dir := t.TempDir()
	e := &Expedition{
		Number:    42,
		Continent: dir,
		Config: domain.Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
		},
		Logger:   platform.NewLogger(io.Discard, false),
		Gradient: harness.NewGradientGauge(5),
		Reserve:  harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	prompt := e.BuildPrompt()

	if !containsStr(prompt, "Expedition #42") {
		t.Error("prompt should contain expedition number")
	}
	if !containsStr(prompt, "flag.md") {
		t.Error("prompt should reference flag.md")
	}
	if !containsStr(prompt, "Rules of Engagement") {
		t.Error("prompt should contain mission rules of engagement")
	}
	if !containsStr(prompt, "CLAUDE.md") {
		t.Error("prompt should reference CLAUDE.md")
	}
	if !containsStr(prompt, "journal") {
		t.Error("prompt should reference journal")
	}
}

func TestExpedition_BuildPrompt_French(t *testing.T) {
	orig := domain.Lang
	defer func() { domain.Lang = orig }()
	domain.Lang = "fr"

	dir := t.TempDir()
	e := &Expedition{
		Number:    7,
		Continent: dir,
		Config: domain.Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
		},
		Logger:   platform.NewLogger(io.Discard, false),
		Gradient: harness.NewGradientGauge(5),
		Reserve:  harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
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
	g := harness.NewGradientGauge(5)
	g.Charge()
	g.Charge()

	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    domain.Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  g,
		Reserve:   harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
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
		Config:    domain.Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  harness.NewGradientGauge(5),
		Reserve:   harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
		Luminas: []domain.Lumina{
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
		Config:    domain.Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  harness.NewGradientGauge(5),
		Reserve:   harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	prompt := e.BuildPrompt()
	if !containsStr(prompt, "No Lumina learned") {
		t.Error("prompt should say no luminas")
	}
}

func TestExpedition_BuildPrompt_ReserveInfo(t *testing.T) {
	dir := t.TempDir()
	rp := harness.NewReserveParty("opus", []string{"sonnet"}, platform.NewLogger(io.Discard, false))
	rp.CheckOutput("rate limit") // Switch to reserve

	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    domain.Config{BaseBranch: "develop", DevURL: "http://localhost:5173"},
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  harness.NewGradientGauge(5),
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
		Config:    domain.Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  harness.NewGradientGauge(5),
		Reserve:   harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
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
		Config:    domain.Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Luminas:   nil,
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  harness.NewGradientGauge(5),
		Reserve:   harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
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
		Config:    domain.Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Luminas:   nil,
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  harness.NewGradientGauge(5),
		Reserve:   harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	prompt := exp.BuildPrompt()

	if strings.Contains(prompt, "Injected Context") {
		t.Error("context section should not appear when no context files exist")
	}
}

// TestExpedition_Run_IsDeprecatedPostMCPPivot replaces nine legacy Run()
// behaviour tests that previously asserted on stdout streaming, prompt
// file writes, output file writes, rate-limit detection, and context
// cancellation paths. The Phase 1 completion commit on the
// feat/jun15-mcp-pivot branch removed the entire `claude -p` invocation
// block (~335 lines) and replaced Run() with a fail-fast stub that
// returns ErrMCPPivotDeprecated. The legacy expectations belong to the
// pre-pivot Go CLI control plane and no longer apply; the new
// expectation is that Run() always reports the deprecation so callers
// migrate to the claude code /expedition-next skill.
func TestExpedition_Run_IsDeprecatedPostMCPPivot(t *testing.T) {
	exp := newTestExpedition(t, "ignored output", 0)
	ctx := context.Background()

	out, err := exp.Run(ctx)
	if err == nil {
		t.Fatal("expected Run() to return an error post jun15 MCP pivot, got nil")
	}
	if !errors.Is(err, ErrMCPPivotDeprecated) {
		t.Errorf("Run() error = %v, want ErrMCPPivotDeprecated", err)
	}
	if out != "" {
		t.Errorf("Run() output = %q, want empty (no LLM invocation)", out)
	}
}

func TestNewPaintress_BasicConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := domain.Config{
		Continent:      dir,
		MaxExpeditions: 10,
		TimeoutSec:     60,
		Model:          "opus",
		BaseBranch:     "main",
		DevCmd:         "npm run dev",
		DevURL:         "http://localhost:3000",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)
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
	cfg := domain.Config{
		Continent: dir,
		Model:     "opus,sonnet,haiku",
		DevCmd:    "npm run dev",
		DevURL:    "http://localhost:3000",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)
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
	cfg := domain.Config{
		Continent: dir,
		Model:     "opus , sonnet , haiku",
		DevCmd:    "npm run dev",
		DevURL:    "http://localhost:3000",
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)
	if p.reserve.ActiveModel() != "opus" {
		t.Errorf("primary should be opus, got %q", p.reserve.ActiveModel())
	}
}

// Removed in refs/issues/0027 Phase 1 (jun15 MCP pivot completion):
// Run() no longer invokes claude -p, so flag.md / watcher / streaming
// path tests retired with it.

// TestExpedition_Run_WatcherReadsFromWorkDir_NotContinent (removed):
// Worktree watcher test removed in jun15 MCP pivot: Run() no longer
// spawns claude -p so WorkDir flag.md polling has nothing to wait on.

func TestExpedition_BuildPrompt_ContainsFlagWriteInstruction(t *testing.T) {
	dir := t.TempDir()
	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    domain.Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  harness.NewGradientGauge(5),
		Reserve:   harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	for _, lang := range []string{"en", "ja", "fr"} {
		t.Run(lang, func(t *testing.T) {
			orig := domain.Lang
			defer func() { domain.Lang = orig }()
			domain.Lang = lang

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
		Config:    domain.Config{BaseBranch: "main", DevURL: ""},
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  harness.NewGradientGauge(5),
		Reserve:   harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	// Test all 3 languages
	for _, lang := range []string{"en", "ja", "fr"} {
		t.Run(lang, func(t *testing.T) {
			orig := domain.Lang
			defer func() { domain.Lang = orig }()
			domain.Lang = lang

			prompt := e.BuildPrompt()

			if containsStr(prompt, "- Dev server:") || containsStr(prompt, "- Serveur dev :") {
				t.Errorf("[%s] prompt should NOT contain dev server environment line when DevURL is empty", lang)
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
	cfg := &domain.ProjectConfig{
		Tracker: domain.IssueTrackerConfig{
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
		Config:    domain.Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  harness.NewGradientGauge(5),
		Reserve:   harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
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
		Config:    domain.Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  harness.NewGradientGauge(5),
		Reserve:   harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
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
		Config:    domain.Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  harness.NewGradientGauge(5),
		Reserve:   harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
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
	t.Skip("Run() deprecated post jun15 MCP pivot (refs/issues/0027); legacy LLM invocation test retired pending full deletion in sub-B commit")
	dir := t.TempDir()
	logDir := t.TempDir()

	// Phase 1: set up config as if `paintress init` was run
	initCfg := &domain.ProjectConfig{
		Tracker: domain.IssueTrackerConfig{
			Team:    "MY",
			Project: "paintress",
		},
	}
	if err := SaveProjectConfig(dir, initCfg); err != nil {
		t.Fatalf("SaveProjectConfig: %v", err)
	}

	// Verify config was persisted
	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if cfg.Tracker.Team != "MY" || cfg.Tracker.Project != "paintress" {
		t.Fatalf("unexpected config: team=%q project=%q", cfg.Tracker.Team, cfg.Tracker.Project)
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
		Config: domain.Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
		},
		LogDir:   logDir,
		Logger:   platform.NewLogger(io.Discard, false),
		Gradient: harness.NewGradientGauge(5),
		Reserve:  harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
		makeCmd:  fakeMakeCmd(reportOutput, 0),
	}

	ctx := context.Background()
	output, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Phase 3: verify expedition output is valid
	report, status := domain.ParseReport(output, 1)
	if status != domain.StatusSuccess {
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
	t.Skip("Run() deprecated post jun15 MCP pivot (refs/issues/0027)")
	dir := t.TempDir()
	logDir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	exp := &Expedition{
		Number:    1,
		Continent: dir,
		Config: domain.Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
		},
		LogDir:   logDir,
		Logger:   platform.NewLogger(io.Discard, false),
		Gradient: harness.NewGradientGauge(5),
		Reserve:  harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
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

func TestBuildPrompt_ContainsMissionSection(t *testing.T) {
	dir := t.TempDir()

	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    domain.Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Logger:    platform.NewLogger(io.Discard, false),
		Gradient:  harness.NewGradientGauge(5),
		Reserve:   harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	prompt := e.BuildPrompt()

	// Mission content should be embedded directly in the prompt
	if !containsStr(prompt, "Rules of Engagement") {
		t.Error("prompt should contain mission 'Rules of Engagement' section")
	}
	if !containsStr(prompt, "implement") && !containsStr(prompt, "verify") {
		t.Error("prompt should contain mission type descriptions")
	}
}

func TestContainsIssue_Match(t *testing.T) {
	if !domain.ContainsIssue([]string{"MY-42", "MY-43"}, "MY-42") {
		t.Error("should match MY-42 in list")
	}
}

func TestContainsIssue_NoMatch(t *testing.T) {
	if domain.ContainsIssue([]string{"MY-42", "MY-43"}, "MY-99") {
		t.Error("should not match MY-99")
	}
}

func TestContainsIssue_EmptyList(t *testing.T) {
	if domain.ContainsIssue(nil, "MY-42") {
		t.Error("empty list should not match")
	}
}

func TestContainsIssue_EmptyTarget(t *testing.T) {
	if domain.ContainsIssue([]string{"MY-42"}, "") {
		t.Error("empty target should not match")
	}
}

func TestContainsIssue_CaseInsensitive(t *testing.T) {
	if !domain.ContainsIssue([]string{"my-42"}, "MY-42") {
		t.Error("should match case-insensitively")
	}
}

func TestMidMatchedDMails_Empty(t *testing.T) {
	exp := &Expedition{}
	got := exp.MidMatchedDMails()
	if got == nil {
		t.Fatal("should return non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("should be empty, got %d", len(got))
	}
}

func TestMidMatchedDMails_ReturnsCopy(t *testing.T) {
	exp := &Expedition{}
	exp.midMatchedMu.Lock() // nosemgrep: adr0005-mutex-lock-without-defer-unlock -- intentional short critical section with explicit Unlock [permanent]
	exp.midMatchedMails = []domain.DMail{{Name: "spec-1", Kind: "specification"}}
	exp.midMatchedMu.Unlock()

	got := exp.MidMatchedDMails()
	if len(got) != 1 || got[0].Name != "spec-1" {
		t.Fatalf("unexpected result: %v", got)
	}

	// mutating returned slice must not affect internal state
	got[0].Name = "MUTATED"
	internal := exp.MidMatchedDMails()
	if internal[0].Name != "spec-1" {
		t.Error("MidMatchedDMails should return a defensive copy")
	}
}

func TestExpedition_MidMatchedRouting_MatchesCurrentIssue(t *testing.T) {
	t.Skip("Run() deprecated post jun15 MCP pivot (refs/issues/0027)")
	// given — expedition with a shell script that:
	//   1. writes current_issue to flag.md
	//   2. writes a matching D-Mail to inbox/
	//   3. writes a non-matching D-Mail to inbox/
	dir := t.TempDir()
	logDir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", "inbox"), 0755)

	flagPath := filepath.Join(dir, ".expedition", ".run", "flag.md")
	inboxDir := filepath.Join(dir, ".expedition", "inbox")

	script := filepath.Join(dir, "route-test.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
# Step 1: Write current_issue to flag.md
cat > %s << 'FLAGEOF'
current_issue: MY-42
current_title: route test issue
FLAGEOF
# Step 2: Wait for watcher to pick up flag
sleep 1
# Step 3: Write matching D-Mail (issues contains MY-42)
cat > %s/spec-matched.md << 'DMEOF'
---
name: spec-matched
kind: specification
description: matched d-mail
issues:
  - MY-42
---

Matched body
DMEOF
# Step 4: Write non-matching D-Mail (issues contains MY-99)
cat > %s/spec-unmatched.md << 'DMEOF2'
---
name: spec-unmatched
kind: specification
description: unmatched d-mail
issues:
  - MY-99
---

Unmatched body
DMEOF2
# Step 5: Wait for inbox watcher to process
sleep 1
echo "done"
`, flagPath, inboxDir, inboxDir)
	os.WriteFile(script, []byte(scriptContent), 0755)

	exp := &Expedition{
		Number:    1,
		Continent: dir,
		Config: domain.Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
			ClaudeCmd:  script,
		},
		LogDir:   logDir,
		Logger:   platform.NewLogger(io.Discard, false),
		DataOut:  io.Discard,
		Gradient: harness.NewGradientGauge(5),
		Reserve:  harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	// when
	ctx := context.Background()
	_, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// then — only the matching D-Mail should be in midMatchedMails
	matched := exp.MidMatchedDMails()
	if len(matched) != 1 {
		t.Fatalf("expected 1 matched d-mail, got %d: %v", len(matched), matched)
	}
	if matched[0].Name != "spec-matched" {
		t.Errorf("matched[0].Name = %q, want spec-matched", matched[0].Name)
	}
}

func TestExpedition_MidMatchedRouting_NoCurrentIssue_NoMatch(t *testing.T) {
	t.Skip("Run() deprecated post jun15 MCP pivot (refs/issues/0027)")
	// given — expedition that writes D-Mails but never sets current_issue
	dir := t.TempDir()
	logDir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", "inbox"), 0755)

	inboxDir := filepath.Join(dir, ".expedition", "inbox")
	script := filepath.Join(dir, "no-issue-test.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
# Write D-Mail without setting current_issue in flag
cat > %s/spec-orphan.md << 'DMEOF'
---
name: spec-orphan
kind: specification
description: orphan d-mail
issues:
  - MY-42
---

Orphan body
DMEOF
sleep 1
echo "done"
`, inboxDir)
	os.WriteFile(script, []byte(scriptContent), 0755)

	exp := &Expedition{
		Number:    1,
		Continent: dir,
		Config: domain.Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
			ClaudeCmd:  script,
		},
		LogDir:   logDir,
		Logger:   platform.NewLogger(io.Discard, false),
		DataOut:  io.Discard,
		Gradient: harness.NewGradientGauge(5),
		Reserve:  harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	// when
	ctx := context.Background()
	_, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// then — no matches (currentIssue was never set)
	matched := exp.MidMatchedDMails()
	if len(matched) != 0 {
		t.Errorf("expected 0 matched d-mails when no current_issue, got %d", len(matched))
	}
}

// TestExpedition_MidMatchedRouting_StaleFlagIgnored verifies that a
// stale current_issue in flag.md from a previous interrupted expedition
// does not cause incorrect routing. Run() should clear stale current_issue
// before starting the watchFlag watcher.
func TestExpedition_MidMatchedRouting_StaleFlagIgnored(t *testing.T) {
	t.Skip("Run() deprecated post jun15 MCP pivot (refs/issues/0027)")
	// given — flag.md already has a stale current_issue from a prior run
	dir := t.TempDir()
	logDir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", "inbox"), 0755)

	flagPath := filepath.Join(dir, ".expedition", ".run", "flag.md")
	inboxDir := filepath.Join(dir, ".expedition", "inbox")

	// Pre-populate flag.md with stale current_issue (simulates interrupted prior run)
	os.WriteFile(flagPath, []byte("current_issue: STALE-99\ncurrent_title: stale issue\n"), 0644)

	// Script does NOT write current_issue — only drops a D-Mail for STALE-99
	script := filepath.Join(dir, "stale-test.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
# Write a D-Mail that matches the STALE issue (should NOT be routed)
sleep 0.5
cat > %s/spec-stale.md << 'DMEOF'
---
name: spec-stale
kind: specification
description: d-mail for stale issue
issues:
  - STALE-99
---

Stale body
DMEOF
sleep 1
echo "done"
`, inboxDir)
	os.WriteFile(script, []byte(scriptContent), 0755)

	exp := &Expedition{
		Number:    1,
		Continent: dir,
		Config: domain.Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
			ClaudeCmd:  script,
		},
		LogDir:   logDir,
		Logger:   platform.NewLogger(io.Discard, false),
		DataOut:  io.Discard,
		Gradient: harness.NewGradientGauge(5),
		Reserve:  harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	// when
	ctx := context.Background()
	_, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// then — stale current_issue should have been cleared;
	// the D-Mail for STALE-99 should NOT be in midMatchedMails
	matched := exp.MidMatchedDMails()
	if len(matched) != 0 {
		t.Errorf("expected 0 matched d-mails (stale issue should be ignored), got %d: %v", len(matched), matched)
	}
}

// TestExpedition_MidMatchedRouting_HighSeverityAlsoRouted verifies that
// a HIGH severity D-Mail matching the current issue is collected in both
// midHighNames (for notification) and midMatchedMails (for follow-up).
func TestExpedition_MidMatchedRouting_HighSeverityAlsoRouted(t *testing.T) {
	t.Skip("Run() deprecated post jun15 MCP pivot (refs/issues/0027)")
	// given — expedition writes current_issue, then a HIGH severity D-Mail matching it
	dir := t.TempDir()
	logDir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run"), 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", "inbox"), 0755)

	flagPath := filepath.Join(dir, ".expedition", ".run", "flag.md")
	inboxDir := filepath.Join(dir, ".expedition", "inbox")

	script := filepath.Join(dir, "high-route-test.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
# Step 1: Write current_issue to flag.md
cat > %s << 'FLAGEOF'
current_issue: MY-42
current_title: high severity route test
FLAGEOF
# Step 2: Wait for watcher to pick up flag
sleep 1
# Step 3: Write HIGH severity D-Mail matching the issue
cat > %s/urgent-spec.md << 'DMEOF'
---
name: urgent-spec
kind: specification
description: urgent matched d-mail
severity: high
issues:
  - MY-42
---

Urgent body
DMEOF
# Step 4: Wait for inbox watcher to process
sleep 1
echo "done"
`, flagPath, inboxDir)
	os.WriteFile(script, []byte(scriptContent), 0755)

	exp := &Expedition{
		Number:    1,
		Continent: dir,
		Config: domain.Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
			ClaudeCmd:  script,
		},
		LogDir:   logDir,
		Logger:   platform.NewLogger(io.Discard, false),
		DataOut:  io.Discard,
		Gradient: harness.NewGradientGauge(5),
		Reserve:  harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	// when
	ctx := context.Background()
	_, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// then — should appear in BOTH midHighNames and midMatchedMails
	highNames := exp.MidHighSeverityDMails()
	if len(highNames) != 1 || highNames[0] != "urgent-spec" {
		t.Errorf("expected midHighNames=[urgent-spec], got %v", highNames)
	}

	matched := exp.MidMatchedDMails()
	if len(matched) != 1 {
		t.Fatalf("expected 1 matched d-mail (HIGH severity), got %d: %v", len(matched), matched)
	}
	if matched[0].Name != "urgent-spec" {
		t.Errorf("matched[0].Name = %q, want urgent-spec", matched[0].Name)
	}
}

func TestMidMatchedDMails_ConcurrentSafe(t *testing.T) {
	exp := &Expedition{}

	// concurrent writes
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			exp.midMatchedMu.Lock() // nosemgrep: adr0005-mutex-lock-without-defer-unlock -- intentional short critical section with explicit Unlock [permanent]
			exp.midMatchedMails = append(exp.midMatchedMails, domain.DMail{Name: fmt.Sprintf("dm-%d", n)})
			exp.midMatchedMu.Unlock()
		}(i)
	}
	wg.Wait()

	got := exp.MidMatchedDMails()
	if len(got) != 10 {
		t.Errorf("expected 10 d-mails, got %d", len(got))
	}
}

// TestExpedition_MidMatchedRouting_WorkDirIsolation verifies that when
// WorkDir differs from Continent (Workers>0 worktree mode), watchFlag
// monitors {WorkDir}/.expedition/.run/flag.md — not Continent's.
// This ensures per-worker isolation: each worker detects only the issue
// written by its own Claude process running in its worktree.
func TestExpedition_MidMatchedRouting_WorkDirIsolation(t *testing.T) {
	t.Skip("Run() deprecated post jun15 MCP pivot (refs/issues/0027)")
	// given — separate Continent and WorkDir (simulates Workers>0 worktree)
	continent := t.TempDir()
	workDir := t.TempDir()
	logDir := t.TempDir()
	os.MkdirAll(filepath.Join(continent, ".expedition", "journal"), 0755)
	os.MkdirAll(filepath.Join(continent, ".expedition", "inbox"), 0755)
	// WorkDir's .expedition/.run/ will be created by Run()

	workDirFlagPath := filepath.Join(workDir, ".expedition", ".run", "flag.md")
	continentInboxDir := filepath.Join(continent, ".expedition", "inbox")

	// Script writes current_issue to WorkDir (where Claude runs),
	// then drops a matching D-Mail into Continent's inbox (shared).
	script := filepath.Join(workDir, "workdir-isolation-test.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
# Write current_issue to WorkDir flag.md (Claude writes relative to cmd.Dir)
mkdir -p %s
cat > %s << 'FLAGEOF'
current_issue: WORKER-1
current_title: worker isolation test
FLAGEOF
# Wait for watcher to detect
sleep 1
# Drop matching D-Mail into Continent's shared inbox
cat > %s/spec-worker1.md << 'DMEOF'
---
name: spec-worker1
kind: specification
description: d-mail for worker 1
issues:
  - WORKER-1
---

Worker 1 body
DMEOF
# Wait for inbox watcher to process
sleep 1
echo "done"
`, filepath.Dir(workDirFlagPath), workDirFlagPath, continentInboxDir)
	os.WriteFile(script, []byte(scriptContent), 0755)

	exp := &Expedition{
		Number:    1,
		Continent: continent,
		WorkDir:   workDir,
		Config: domain.Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
			ClaudeCmd:  script,
		},
		LogDir:   logDir,
		Logger:   platform.NewLogger(io.Discard, false),
		DataOut:  io.Discard,
		Gradient: harness.NewGradientGauge(5),
		Reserve:  harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	// when
	ctx := context.Background()
	_, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// then — watchFlag should have detected current_issue from WorkDir,
	// and the matching D-Mail should be routed
	matched := exp.MidMatchedDMails()
	if len(matched) != 1 {
		t.Fatalf("expected 1 matched d-mail (from WorkDir flag), got %d: %v", len(matched), matched)
	}
	if matched[0].Name != "spec-worker1" {
		t.Errorf("matched[0].Name = %q, want spec-worker1", matched[0].Name)
	}
}

// TestExpedition_StaleFlagClearedOnWorkDir verifies that when WorkDir
// differs from Continent, a stale current_issue in {WorkDir}/.expedition/.run/flag.md
// is cleared before the expedition starts — preventing incorrect D-Mail routing.
func TestExpedition_StaleFlagClearedOnWorkDir(t *testing.T) {
	t.Skip("Run() deprecated post jun15 MCP pivot (refs/issues/0027)")
	// given — WorkDir has stale current_issue from a prior interrupted run
	continent := t.TempDir()
	workDir := t.TempDir()
	logDir := t.TempDir()
	os.MkdirAll(filepath.Join(continent, ".expedition", "journal"), 0755)
	os.MkdirAll(filepath.Join(continent, ".expedition", "inbox"), 0755)
	os.MkdirAll(filepath.Join(workDir, ".expedition", ".run"), 0755)

	// Pre-populate WorkDir's flag.md with stale current_issue
	staleFlagPath := filepath.Join(workDir, ".expedition", ".run", "flag.md")
	os.WriteFile(staleFlagPath, []byte("current_issue: STALE-77\ncurrent_title: stale from prior run\n"), 0644)

	continentInboxDir := filepath.Join(continent, ".expedition", "inbox")

	// Script does NOT write current_issue — only drops a D-Mail for STALE-77
	script := filepath.Join(workDir, "stale-workdir-test.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
sleep 0.5
cat > %s/spec-stale77.md << 'DMEOF'
---
name: spec-stale77
kind: specification
description: d-mail for stale issue
issues:
  - STALE-77
---

Stale body
DMEOF
sleep 1
echo "done"
`, continentInboxDir)
	os.WriteFile(script, []byte(scriptContent), 0755)

	exp := &Expedition{
		Number:    1,
		Continent: continent,
		WorkDir:   workDir,
		Config: domain.Config{
			BaseBranch: "main",
			DevURL:     "http://localhost:3000",
			TimeoutSec: 30,
			ClaudeCmd:  script,
		},
		LogDir:   logDir,
		Logger:   platform.NewLogger(io.Discard, false),
		DataOut:  io.Discard,
		Gradient: harness.NewGradientGauge(5),
		Reserve:  harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	// when
	ctx := context.Background()
	_, err := exp.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// then — stale current_issue should have been cleared from WorkDir's flag.md;
	// the D-Mail for STALE-77 should NOT be in midMatchedMails
	matched := exp.MidMatchedDMails()
	if len(matched) != 0 {
		t.Errorf("expected 0 matched d-mails (stale WorkDir flag should be cleared), got %d: %v", len(matched), matched)
	}

	// Also verify the stale current_issue was actually removed from flag.md
	flag := ReadFlag(workDir)
	if flag.CurrentIssue != "" {
		t.Errorf("WorkDir flag.md should have current_issue cleared, got %q", flag.CurrentIssue)
	}
}

// TestExpedition_TwoWorkersConcurrent_NoContamination removed in
// refs/issues/0027 Phase 1 sub-B: the cross-worker D-Mail routing it
// exercised lived inside the deleted Run() body. The Phase 2 commit
// that wires sightjack → paintress through real MCP tools will
// reintroduce a concurrency test against the MCP-server contract.

func TestNewPaintress_NoDev_NoDevServer(t *testing.T) {
	dir := t.TempDir()
	cfg := domain.Config{
		Continent: dir,
		Model:     "opus",
		DevCmd:    "npm run dev",
		DevURL:    "http://localhost:3000",
		NoDev:     true,
	}

	p := NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)

	if p.devServer != nil {
		t.Error("devServer should be nil when NoDev=true")
	}
	if p.config.DevURL != "" {
		t.Errorf("DevURL should be cleared when NoDev=true, got %q", p.config.DevURL)
	}
}

// --- ReadContextFiles tests (merged from context_test.go) ---

func TestReadContextFiles_ReadsMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".expedition", "context")
	os.MkdirAll(ctxDir, 0755)

	os.WriteFile(filepath.Join(ctxDir, "architecture.md"), []byte("Use hexagonal architecture.\n"), 0644)
	os.WriteFile(filepath.Join(ctxDir, "naming.md"), []byte("Use snake_case for API fields.\n"), 0644)

	result, err := ReadContextFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "architecture") {
		t.Error("expected context to contain 'architecture' header")
	}
	if !strings.Contains(result, "Use hexagonal architecture.") {
		t.Error("expected context to contain architecture.md content")
	}
	if !strings.Contains(result, "naming") {
		t.Error("expected context to contain 'naming' header")
	}
	if !strings.Contains(result, "Use snake_case for API fields.") {
		t.Error("expected context to contain naming.md content")
	}
}

func TestReadContextFiles_EmptyWhenNoDirectory(t *testing.T) {
	dir := t.TempDir()

	result, err := ReadContextFiles(dir)

	if err != nil {
		t.Errorf("missing directory should not be an error, got %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string when no context dir, got %q", result)
	}
}

func TestReadContextFiles_ErrorOnPermissionDenied(t *testing.T) {
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".expedition", "context")
	os.MkdirAll(ctxDir, 0755)

	// Write a valid file, then remove read permission on the directory
	os.WriteFile(filepath.Join(ctxDir, "rules.md"), []byte("important rules\n"), 0644)
	os.Chmod(ctxDir, 0000)
	t.Cleanup(func() { os.Chmod(ctxDir, 0755) })

	_, err := ReadContextFiles(dir)

	if err == nil {
		t.Error("expected error for permission-denied directory, got nil")
	}
}

func TestReadContextFiles_IgnoresNonMarkdown(t *testing.T) {
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".expedition", "context")
	os.MkdirAll(ctxDir, 0755)

	os.WriteFile(filepath.Join(ctxDir, "notes.md"), []byte("important\n"), 0644)
	os.WriteFile(filepath.Join(ctxDir, "data.json"), []byte(`{"key":"val"}`), 0644)
	os.MkdirAll(filepath.Join(ctxDir, "subdir"), 0755)

	result, err := ReadContextFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "important") {
		t.Error("expected .md file to be included")
	}
	if strings.Contains(result, "key") {
		t.Error(".json files should be excluded")
	}
}

func TestReadContextFiles_OrdersByFilename(t *testing.T) {
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".expedition", "context")
	os.MkdirAll(ctxDir, 0755)

	os.WriteFile(filepath.Join(ctxDir, "b.md"), []byte("second\n"), 0644)
	os.WriteFile(filepath.Join(ctxDir, "a.md"), []byte("first\n"), 0644)

	result, err := ReadContextFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	firstIdx := strings.Index(result, "### a")
	secondIdx := strings.Index(result, "### b")
	if firstIdx == -1 || secondIdx == -1 {
		t.Fatalf("expected headers for a.md and b.md, got: %q", result)
	}
	if firstIdx >= secondIdx {
		t.Errorf("expected a.md before b.md, got indices %d >= %d", firstIdx, secondIdx)
	}
}

func TestReadContextFiles_EmptyFileStillCreatesHeader(t *testing.T) {
	dir := t.TempDir()
	ctxDir := filepath.Join(dir, ".expedition", "context")
	os.MkdirAll(ctxDir, 0755)

	os.WriteFile(filepath.Join(ctxDir, "empty.md"), []byte(""), 0644)

	result, err := ReadContextFiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "### empty") {
		t.Error("expected header for empty.md even when file is empty")
	}
}

// --- MissionText tests (merged from mission_test.go) ---

func TestMissionText_English(t *testing.T) {
	text := harness.MissionText(harness.MustDefaultPromptRegistry(), "en", false)
	if !containsStr(text, "Rules of Engagement") {
		t.Error("English mission should contain 'Rules of Engagement'")
	}
	if !containsStr(text, "implement") {
		t.Error("English mission should contain 'implement'")
	}
	if !containsStr(text, "verify") {
		t.Error("English mission should contain 'verify'")
	}
	if !containsStr(text, "fix") {
		t.Error("English mission should contain 'fix'")
	}
	if containsStr(text, "行動規範") {
		t.Error("English mission should not contain Japanese")
	}
}

func TestMissionText_Japanese(t *testing.T) {
	text := harness.MissionText(harness.MustDefaultPromptRegistry(), "ja", false)
	if !containsStr(text, "行動規範") {
		t.Error("Japanese mission should contain '行動規範'")
	}
	if !containsStr(text, "使命の取得") {
		t.Error("Japanese mission should contain '使命の取得'")
	}
	if !containsStr(text, "禁止事項") {
		t.Error("Japanese mission should contain '禁止事項'")
	}
}

func TestMissionText_French(t *testing.T) {
	text := harness.MissionText(harness.MustDefaultPromptRegistry(), "fr", false)
	if !containsStr(text, "engagement") {
		t.Error("French mission should contain 'engagement'")
	}
	if containsStr(text, "行動規範") {
		t.Error("French mission should not contain Japanese")
	}
}

func TestMissionText_FallbackToEnglish(t *testing.T) {
	text := harness.MissionText(harness.MustDefaultPromptRegistry(), "de", false)
	if !containsStr(text, "Rules of Engagement") {
		t.Error("unsupported lang should fall back to English")
	}
}

func TestMissionText_English_ContainsConsolidate(t *testing.T) {
	text := harness.MissionText(harness.MustDefaultPromptRegistry(), "en", false)
	if !containsStr(text, "consolidate") {
		t.Error("English mission should contain 'consolidate'")
	}
	if !containsStr(text, "consolidate Procedure") {
		t.Error("English mission should contain 'consolidate Procedure' section")
	}
	if !containsStr(text, "Protected branches") {
		t.Error("English mission should contain protected branches constraint")
	}
}

func TestMissionText_Japanese_ContainsConsolidate(t *testing.T) {
	text := harness.MissionText(harness.MustDefaultPromptRegistry(), "ja", false)
	if !containsStr(text, "consolidate") {
		t.Error("Japanese mission should contain 'consolidate'")
	}
	if !containsStr(text, "consolidate の手順") {
		t.Error("Japanese mission should contain 'consolidate の手順' section")
	}
}

func TestMissionText_French_ContainsConsolidate(t *testing.T) {
	text := harness.MissionText(harness.MustDefaultPromptRegistry(), "fr", false)
	if !containsStr(text, "consolidate") {
		t.Error("French mission should contain 'consolidate'")
	}
	if !containsStr(text, "Procédure consolidate") {
		t.Error("French mission should contain 'Procédure consolidate' section")
	}
}

func TestExpeditionPrompt_ContainsConsolidateMissionType(t *testing.T) {
	for _, lang := range []string{"en", "ja", "fr"} {
		t.Run(lang, func(t *testing.T) {
			orig := domain.Lang
			defer func() { domain.Lang = orig }()
			domain.Lang = lang

			data := domain.PromptData{
				Number:         1,
				Timestamp:      "2026-03-11",
				Bt:             "`",
				Cb:             "```",
				BaseBranch:     "main",
				ReserveSection: "Model: opus",
				MissionSection: harness.MissionText(harness.MustDefaultPromptRegistry(), lang, false),
			}

			result := harness.RenderExpeditionPrompt(harness.MustDefaultPromptRegistry(), lang, data)
			if !containsStr(result, "implement|verify|fix|consolidate") {
				t.Errorf("expedition prompt (%s) should contain 'implement|verify|fix|consolidate'", lang)
			}
		})
	}
}

func TestMissionText_WaveMode_NoLinearReference(t *testing.T) {
	for _, lang := range []string{"en", "ja", "fr"} {
		t.Run(lang, func(t *testing.T) {
			// when
			text := harness.MissionText(harness.MustDefaultPromptRegistry(), lang, true)

			// then
			if strings.Contains(text, "Linear") {
				t.Errorf("lang=%s: wave mode mission should not reference Linear", lang)
			}
			if !strings.Contains(text, "gh") && !strings.Contains(text, "Wave") {
				t.Errorf("lang=%s: wave mode mission should reference gh CLI or Wave", lang)
			}
		})
	}
}

func TestMissionText_LinearMode_HasLinearReference(t *testing.T) {
	for _, lang := range []string{"en", "ja", "fr"} {
		t.Run(lang, func(t *testing.T) {
			// when
			text := harness.MissionText(harness.MustDefaultPromptRegistry(), lang, false)

			// then
			if !strings.Contains(text, "Linear") {
				t.Errorf("lang=%s: linear mode mission should reference Linear", lang)
			}
		})
	}
}

func TestExpeditionPrompt_WaveMode_NoLinearReference(t *testing.T) {
	for _, lang := range []string{"en", "ja", "fr"} {
		t.Run(lang, func(t *testing.T) {
			data := domain.PromptData{
				Number:         1,
				Timestamp:      "2026-03-11",
				Bt:             "`",
				Cb:             "```",
				BaseBranch:     "main",
				ReserveSection: "Model: opus",
				MissionSection: harness.MissionText(harness.MustDefaultPromptRegistry(), lang, true),
				WaveTarget:     &domain.ExpeditionTarget{Title: "Test Step"},
			}

			result := harness.RenderExpeditionPrompt(harness.MustDefaultPromptRegistry(), lang, data)
			if strings.Contains(result, "Linear MCP") {
				t.Errorf("lang=%s: wave mode expedition should not reference Linear MCP", lang)
			}
		})
	}
}

// TestExpedition_Run_ShortTimeout removed in refs/issues/0027 Phase 1
// sub-B: Run() no longer invokes claude -p, so the timeout-relative
// behavior it exercised no longer applies.
