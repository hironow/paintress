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

// TestLifecycle_Init_Then_Expedition removed in refs/issues/0027 Phase 1
// sub-C: the full init→expedition→prompt-file lifecycle relied on the
// deleted Run() body. Phase 2 will reintroduce a fresh test against the
// MCP-server-driven workflow, exercising init config + the new prompt
// assembly without invoking claude -p.

// TestLifecycle_NoInit_Then_Expedition removed in refs/issues/0027
// Phase 1 sub-C: the prompt file produced by Run() no longer exists
// since Run() returns ErrMCPPivotDeprecated. Phase 2 will reintroduce
// a fresh test against the MCP-server-driven prompt assembly.

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

// TestExpedition_MidMatchedRouting_MatchesCurrentIssue removed in
// refs/issues/0027 Phase 1 sub-C: the current-issue routing happy
// path lived inside Run() body. Phase 2 MCP wiring will reintroduce
// a fresh test against the new contract.

// TestExpedition_MidMatchedRouting_NoCurrentIssue_NoMatch removed in
// refs/issues/0027 Phase 1 sub-C: the absence-of-current-issue routing
// guard lived inside Run() body. Phase 2 MCP wiring will reintroduce a
// fresh test.

// TestExpedition_MidMatchedRouting_StaleFlagIgnored removed in
// refs/issues/0027 Phase 1 sub-C: stale flag-clearing path lived in
// Run() body. Phase 2 MCP wiring will reintroduce a fresh test.

// TestExpedition_MidMatchedRouting_HighSeverityAlsoRouted removed in
// refs/issues/0027 Phase 1 sub-C: HIGH severity mid-expedition routing
// lived inside Run() body. Phase 2 MCP wiring will reintroduce a
// severity-aware test against the new contract.

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
// TestExpedition_MidMatchedRouting_WorkDirIsolation removed in refs/issues/0027
// Phase 1 sub-C: the WorkDir-isolated flag watcher lived inside the deleted
// Run() body. Phase 2 wiring will reintroduce flag isolation as part of the
// MCP server contract, with a fresh per-worker test against that contract.

// TestExpedition_StaleFlagClearedOnWorkDir removed in refs/issues/0027
// Phase 1 sub-C: the WorkDir stale-flag clearing happened inside the
// deleted Run() body. Phase 2 wiring will reintroduce flag handling
// as an MCP tool, with a fresh test against that contract.

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
