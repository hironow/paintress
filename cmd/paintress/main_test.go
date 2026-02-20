package main

import (
	"testing"
)

func TestExtractSubcommand_Default(t *testing.T) {
	// Path only → subcmd="run", path="./repo"
	subcmd, repoPath, flags, err := extractSubcommand([]string{"./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	if len(flags) != 0 {
		t.Errorf("flags = %v, want empty", flags)
	}
}

func TestExtractSubcommand_Init(t *testing.T) {
	// "init ./repo" → subcmd="init", path="./repo"
	subcmd, repoPath, flags, err := extractSubcommand([]string{"init", "./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "init" {
		t.Errorf("subcmd = %q, want %q", subcmd, "init")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	if len(flags) != 0 {
		t.Errorf("flags = %v, want empty", flags)
	}
}

func TestExtractSubcommand_FlagsBeforePath(t *testing.T) {
	// "--model opus ./repo" → subcmd="run", path="./repo", flags=["--model", "opus"]
	subcmd, repoPath, flags, err := extractSubcommand([]string{"--model", "opus", "./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	wantFlags := []string{"--model", "opus"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
	for i, f := range flags {
		if f != wantFlags[i] {
			t.Errorf("flags[%d] = %q, want %q", i, f, wantFlags[i])
		}
	}
}

func TestExtractSubcommand_Doctor(t *testing.T) {
	// "doctor" → subcmd="doctor", path=""
	subcmd, repoPath, flags, err := extractSubcommand([]string{"doctor"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "doctor" {
		t.Errorf("subcmd = %q, want %q", subcmd, "doctor")
	}
	if repoPath != "" {
		t.Errorf("repoPath = %q, want empty", repoPath)
	}
	if len(flags) != 0 {
		t.Errorf("flags = %v, want empty", flags)
	}
}

func TestExtractSubcommand_VersionFlag(t *testing.T) {
	// "--version" → subcmd="run", path="", flags=["--version"]
	subcmd, _, flags, err := extractSubcommand([]string{"--version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	hasVersion := false
	for _, f := range flags {
		if f == "--version" {
			hasVersion = true
		}
	}
	if !hasVersion {
		t.Errorf("flags should contain --version, got %v", flags)
	}
}

func TestExtractSubcommand_FlagsAfterPath(t *testing.T) {
	// "./repo --model opus --dry-run" → subcmd="run", path="./repo", flags=["--model", "opus", "--dry-run"]
	subcmd, repoPath, flags, err := extractSubcommand([]string{"./repo", "--model", "opus", "--dry-run"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	wantFlags := []string{"--model", "opus", "--dry-run"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
	for i, f := range flags {
		if f != wantFlags[i] {
			t.Errorf("flags[%d] = %q, want %q", i, f, wantFlags[i])
		}
	}
}

func TestExtractSubcommand_FlagEqualsValue(t *testing.T) {
	// "--model=opus ./repo" → subcmd="run", path="./repo", flags=["--model=opus"]
	subcmd, repoPath, flags, err := extractSubcommand([]string{"--model=opus", "./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	wantFlags := []string{"--model=opus"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
	for i, f := range flags {
		if f != wantFlags[i] {
			t.Errorf("flags[%d] = %q, want %q", i, f, wantFlags[i])
		}
	}
}

func TestExtractSubcommand_BoolFlagEqualsValue(t *testing.T) {
	// "--no-dev=false ./repo" → subcmd="run", path="./repo", flags=["--no-dev=false"]
	subcmd, repoPath, flags, err := extractSubcommand([]string{"--no-dev=false", "./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	wantFlags := []string{"--no-dev=false"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
}

func TestExtractSubcommand_DoubleDashTerminator(t *testing.T) {
	// "-- ./repo" → subcmd="run", path="./repo", flags=[]
	// "--" signals end of flags; everything after is positional
	subcmd, repoPath, flags, err := extractSubcommand([]string{"--", "./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	if len(flags) != 0 {
		t.Errorf("flags = %v, want empty", flags)
	}
}

func TestExtractSubcommand_FlagsBeforeDoubleDash(t *testing.T) {
	// "--model opus -- ./repo" → subcmd="run", path="./repo", flags=["--model", "opus"]
	subcmd, repoPath, flags, err := extractSubcommand([]string{"--model", "opus", "--", "./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	wantFlags := []string{"--model", "opus"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
	for i, f := range flags {
		if f != wantFlags[i] {
			t.Errorf("flags[%d] = %q, want %q", i, f, wantFlags[i])
		}
	}
}

func TestExtractSubcommand_BoolFlagWithExplicitValue(t *testing.T) {
	// "--no-dev false ./repo" → subcmd="run", path="./repo", flags=["--no-dev", "false"]
	subcmd, repoPath, flags, err := extractSubcommand([]string{"--no-dev", "false", "./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	wantFlags := []string{"--no-dev", "false"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
	for i, f := range flags {
		if f != wantFlags[i] {
			t.Errorf("flags[%d] = %q, want %q", i, f, wantFlags[i])
		}
	}
}

func TestExtractSubcommand_BoolFlagWithoutValue(t *testing.T) {
	// "--dry-run ./repo" → subcmd="run", path="./repo", flags=["--dry-run"]
	// "true"/"false" not following, so path should be "./repo"
	subcmd, repoPath, flags, err := extractSubcommand([]string{"--dry-run", "./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	wantFlags := []string{"--dry-run"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
}

func TestExtractSubcommand_BoolFlagWithNumericValue(t *testing.T) {
	// "--no-dev 0 ./repo" → path="./repo", flags=["--no-dev", "0"]
	// Go's strconv.ParseBool accepts 1/0/t/f/T/F in addition to true/false
	subcmd, repoPath, flags, err := extractSubcommand([]string{"--no-dev", "0", "./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	wantFlags := []string{"--no-dev", "0"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
	for i, f := range flags {
		if f != wantFlags[i] {
			t.Errorf("flags[%d] = %q, want %q", i, f, wantFlags[i])
		}
	}
}

func TestExtractSubcommand_BoolFlagWithT(t *testing.T) {
	// "--dry-run T ./repo" → path="./repo", flags=["--dry-run", "T"]
	subcmd, repoPath, flags, err := extractSubcommand([]string{"--dry-run", "T", "./repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	wantFlags := []string{"--dry-run", "T"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
}

func TestExtractSubcommand_DoctorWithOutputFlag(t *testing.T) {
	// "doctor --output json" → subcmd="doctor", path="", flags=["--output", "json"]
	subcmd, repoPath, flags, err := extractSubcommand([]string{"doctor", "--output", "json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "doctor" {
		t.Errorf("subcmd = %q, want %q", subcmd, "doctor")
	}
	if repoPath != "" {
		t.Errorf("repoPath = %q, want empty", repoPath)
	}
	wantFlags := []string{"--output", "json"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
	for i, f := range flags {
		if f != wantFlags[i] {
			t.Errorf("flags[%d] = %q, want %q", i, f, wantFlags[i])
		}
	}
}

func TestExtractSubcommand_DoctorWithOutputEqualsFlag(t *testing.T) {
	// "doctor --output=json" → subcmd="doctor", path="", flags=["--output=json"]
	subcmd, repoPath, flags, err := extractSubcommand([]string{"doctor", "--output=json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "doctor" {
		t.Errorf("subcmd = %q, want %q", subcmd, "doctor")
	}
	if repoPath != "" {
		t.Errorf("repoPath = %q, want empty", repoPath)
	}
	wantFlags := []string{"--output=json"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
}

func TestParseOutputFlag_Default(t *testing.T) {
	// given: no --output flag
	// when
	format := parseOutputFlag([]string{})
	// then
	if format != "text" {
		t.Errorf("format = %q, want %q", format, "text")
	}
}

func TestParseOutputFlag_Json(t *testing.T) {
	// given
	flagArgs := []string{"--output", "json"}
	// when
	format := parseOutputFlag(flagArgs)
	// then
	if format != "json" {
		t.Errorf("format = %q, want %q", format, "json")
	}
}

func TestParseOutputFlag_JsonEquals(t *testing.T) {
	// given
	flagArgs := []string{"--output=json"}
	// when
	format := parseOutputFlag(flagArgs)
	// then
	if format != "json" {
		t.Errorf("format = %q, want %q", format, "json")
	}
}

func TestParseOutputFlag_Text(t *testing.T) {
	// given
	flagArgs := []string{"--output", "text"}
	// when
	format := parseOutputFlag(flagArgs)
	// then
	if format != "text" {
		t.Errorf("format = %q, want %q", format, "text")
	}
}

func TestParseStateFlag_Default(t *testing.T) {
	// given: no --state flag
	// when
	states := parseStateFlag([]string{})
	// then — nil means no filter
	if states != nil {
		t.Errorf("states = %v, want nil", states)
	}
}

func TestParseStateFlag_Single(t *testing.T) {
	// given
	flagArgs := []string{"--state", "todo"}
	// when
	states := parseStateFlag(flagArgs)
	// then
	if len(states) != 1 || states[0] != "todo" {
		t.Errorf("states = %v, want [todo]", states)
	}
}

func TestParseStateFlag_CommaSeparated(t *testing.T) {
	// given
	flagArgs := []string{"--state", "todo,in-progress"}
	// when
	states := parseStateFlag(flagArgs)
	// then — hyphens normalized to spaces for Linear state name matching
	if len(states) != 2 {
		t.Fatalf("states = %v, want 2 elements", states)
	}
	if states[0] != "todo" || states[1] != "in progress" {
		t.Errorf("states = %v, want [todo, in progress]", states)
	}
}

func TestParseStateFlag_EqualsForm(t *testing.T) {
	// given
	flagArgs := []string{"--state=todo,done"}
	// when
	states := parseStateFlag(flagArgs)
	// then
	if len(states) != 2 {
		t.Fatalf("states = %v, want 2 elements", states)
	}
	if states[0] != "todo" || states[1] != "done" {
		t.Errorf("states = %v, want [todo, done]", states)
	}
}

func TestExtractSubcommand_IssuesWithStateFlag(t *testing.T) {
	// "issues ./repo --state todo" → subcmd="issues", path="./repo", flags=["--state", "todo"]
	subcmd, repoPath, flags, err := extractSubcommand([]string{"issues", "./repo", "--state", "todo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "issues" {
		t.Errorf("subcmd = %q, want %q", subcmd, "issues")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	wantFlags := []string{"--state", "todo"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
	for i, f := range flags {
		if f != wantFlags[i] {
			t.Errorf("flags[%d] = %q, want %q", i, f, wantFlags[i])
		}
	}
}

func TestExtractSubcommand_ArchivePrune(t *testing.T) {
	// "archive-prune ./repo --days 7 --execute" → subcmd="archive-prune", path="./repo", flags
	subcmd, repoPath, flags, err := extractSubcommand([]string{"archive-prune", "./repo", "--days", "7", "--execute"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "archive-prune" {
		t.Errorf("subcmd = %q, want %q", subcmd, "archive-prune")
	}
	if repoPath != "./repo" {
		t.Errorf("repoPath = %q, want %q", repoPath, "./repo")
	}
	wantFlags := []string{"--days", "7", "--execute"}
	if len(flags) != len(wantFlags) {
		t.Fatalf("flags = %v, want %v", flags, wantFlags)
	}
	for i, f := range flags {
		if f != wantFlags[i] {
			t.Errorf("flags[%d] = %q, want %q", i, f, wantFlags[i])
		}
	}
}

func TestParseDaysFlag_Default(t *testing.T) {
	days := parseDaysFlag([]string{})
	if days != 30 {
		t.Errorf("days = %d, want 30", days)
	}
}

func TestParseDaysFlag_Explicit(t *testing.T) {
	days := parseDaysFlag([]string{"--days", "7"})
	if days != 7 {
		t.Errorf("days = %d, want 7", days)
	}
}

func TestParseDaysFlag_Equals(t *testing.T) {
	days := parseDaysFlag([]string{"--days=14"})
	if days != 14 {
		t.Errorf("days = %d, want 14", days)
	}
}

func TestParseExecuteFlag_Absent(t *testing.T) {
	if parseExecuteFlag([]string{}) {
		t.Error("execute should be false when flag absent")
	}
}

func TestParseExecuteFlag_Present(t *testing.T) {
	if !parseExecuteFlag([]string{"--execute"}) {
		t.Error("execute should be true when flag present")
	}
}

func TestParseExecuteFlag_ExplicitFalse(t *testing.T) {
	if parseExecuteFlag([]string{"--execute", "false"}) {
		t.Error("execute should be false when --execute false")
	}
}

func TestParseExecuteFlag_EqualsFalse(t *testing.T) {
	if parseExecuteFlag([]string{"--execute=false"}) {
		t.Error("execute should be false when --execute=false")
	}
}

func TestParseExecuteFlag_EqualsTrue(t *testing.T) {
	if !parseExecuteFlag([]string{"--execute=true"}) {
		t.Error("execute should be true when --execute=true")
	}
}

func TestExtractSubcommand_Empty(t *testing.T) {
	// No args → subcmd="run", path=""
	subcmd, repoPath, flags, err := extractSubcommand([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if subcmd != "run" {
		t.Errorf("subcmd = %q, want %q", subcmd, "run")
	}
	if repoPath != "" {
		t.Errorf("repoPath = %q, want empty", repoPath)
	}
	if len(flags) != 0 {
		t.Errorf("flags = %v, want empty", flags)
	}
}
