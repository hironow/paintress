package filter

import (
	"strings"
	"testing"
)

func TestNewRegistry_LoadsAllPrompts(t *testing.T) {
	// given/when
	reg, err := NewRegistry()

	// then
	if err != nil {
		t.Fatalf("NewRegistry() error: %v", err)
	}

	expected := []string{
		"expedition_en",
		"expedition_fr",
		"expedition_ja",
		"fetch_issues",
		"follow_up",
		"mission_en_linear",
		"mission_en_wave",
		"mission_fr_linear",
		"mission_fr_wave",
		"mission_ja_linear",
		"mission_ja_wave",
		"review_fix",
		"review_fix_with_reflection",
		"review_fix_with_strategy",
	}
	names := reg.Names()
	if len(names) != len(expected) {
		t.Fatalf("got %d prompts %v, want %d %v", len(names), names, len(expected), expected)
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("names[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestRegistry_Get_Found(t *testing.T) {
	// given
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error: %v", err)
	}

	// when
	entry, err := reg.Get("review_fix")

	// then
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if entry.Name != "review_fix" {
		t.Errorf("Name = %q, want %q", entry.Name, "review_fix")
	}
	if entry.Version != "1.0" {
		t.Errorf("Version = %q, want %q", entry.Version, "1.0")
	}
	if !strings.Contains(entry.Template, "{branch}") {
		t.Error("template should contain {branch} placeholder")
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	// given
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error: %v", err)
	}

	// when
	_, err = reg.Get("nonexistent")

	// then
	if err == nil {
		t.Fatal("expected error for nonexistent prompt")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to contain 'not found'", err.Error())
	}
}

func TestRegistry_Expand(t *testing.T) {
	// given
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error: %v", err)
	}

	// when
	result, err := reg.Expand("review_fix", map[string]string{
		"branch":   "feat/my-branch",
		"comments": "Fix the typo on line 42.",
	})

	// then
	if err != nil {
		t.Fatalf("Expand() error: %v", err)
	}
	if !strings.Contains(result, "feat/my-branch") {
		t.Error("expanded result should contain the branch name")
	}
	if !strings.Contains(result, "Fix the typo on line 42.") {
		t.Error("expanded result should contain the comments")
	}
	if strings.Contains(result, "{branch}") {
		t.Error("expanded result should not contain unexpanded {branch}")
	}
}

func TestRegistry_Expand_NotFound(t *testing.T) {
	// given
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error: %v", err)
	}

	// when
	_, err = reg.Expand("nonexistent", nil)

	// then
	if err == nil {
		t.Fatal("expected error for nonexistent prompt")
	}
}

// TestRegistry_ReviewFixMatchesLegacy verifies that the YAML template
// produces output equivalent to the original BuildReviewFixPrompt.
func TestRegistry_ReviewFixMatchesLegacy(t *testing.T) {
	// given
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error: %v", err)
	}
	branch := "feat/test-branch"
	comments := "[P1] Fix null check\n[P2] Add test"

	// when — expand from YAML
	yamlResult, err := reg.Expand("review_fix", map[string]string{
		"branch":   branch,
		"comments": comments,
	})
	if err != nil {
		t.Fatalf("Expand() error: %v", err)
	}

	// then — must contain all critical parts of the legacy output
	for _, want := range []string{
		"You are on branch feat/test-branch with an open PR",
		"[P1] Fix null check",
		"[P2] Add test",
		"Fix all review comments above",
		"Do not create a new branch or PR",
		"Do not change the Linear issue status",
	} {
		if !strings.Contains(yamlResult, want) {
			t.Errorf("YAML output missing expected fragment: %q", want)
		}
	}
}

func TestRegistry_FollowUpMatchesLegacy(t *testing.T) {
	// given
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error: %v", err)
	}

	// when
	result, err := reg.Expand("follow_up", map[string]string{
		"dmail_section": "### test-dmail (report)\n\n**Description:** test\n",
	})
	if err != nil {
		t.Fatalf("Expand() error: %v", err)
	}

	// then
	for _, want := range []string{
		"D-Mail(s) arrived during this expedition",
		"Review them and take any additional action",
		"### test-dmail (report)",
	} {
		if !strings.Contains(result, want) {
			t.Errorf("follow_up output missing expected fragment: %q", want)
		}
	}
}

func TestRegistry_FetchIssuesMatchesLegacy(t *testing.T) {
	// given
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error: %v", err)
	}

	// when
	result, err := reg.Expand("fetch_issues", map[string]string{
		"team":           `"TEAM"`,
		"project_clause": ` for project "MyProject"`,
		"output_path":    "/tmp/issues.json",
	})
	if err != nil {
		t.Fatalf("Expand() error: %v", err)
	}

	// then
	for _, want := range []string{
		"mcp__linear__list_issues",
		`"TEAM"`,
		`for project "MyProject"`,
		"/tmp/issues.json",
		"JSON array",
	} {
		if !strings.Contains(result, want) {
			t.Errorf("fetch_issues output missing expected fragment: %q", want)
		}
	}
}

func TestExpandTemplate(t *testing.T) {
	// given
	tmpl := "Hello {name}, welcome to {place}."
	vars := map[string]string{
		"name":  "Alice",
		"place": "Wonderland",
	}

	// when
	result := expandTemplate(tmpl, vars)

	// then
	want := "Hello Alice, welcome to Wonderland."
	if result != want {
		t.Errorf("expandTemplate() = %q, want %q", result, want)
	}
}

func TestExpandTemplate_NoVars(t *testing.T) {
	// given
	tmpl := "No placeholders here."

	// when
	result := expandTemplate(tmpl, nil)

	// then
	if result != tmpl {
		t.Errorf("expandTemplate() = %q, want %q", result, tmpl)
	}
}

func TestRegistry_Names_Sorted(t *testing.T) {
	// given
	reg, err := NewRegistry()
	if err != nil {
		t.Fatalf("NewRegistry() error: %v", err)
	}

	// when
	names := reg.Names()

	// then
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("names not sorted: %q > %q", names[i-1], names[i])
		}
	}
}
