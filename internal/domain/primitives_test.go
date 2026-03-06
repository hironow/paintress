package domain_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestNewRepoPath_Valid(t *testing.T) {
	// given
	raw := "/tmp/repo"

	// when
	rp, err := domain.NewRepoPath(raw)

	// then
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rp.String() != raw {
		t.Errorf("expected %q, got %q", raw, rp.String())
	}
}

func TestNewRepoPath_RejectsEmpty(t *testing.T) {
	// when
	_, err := domain.NewRepoPath("")

	// then
	if err == nil {
		t.Fatal("expected error for empty RepoPath")
	}
}

func TestNewTeam_NonEmpty(t *testing.T) {
	// when
	team := domain.NewTeam("MY")

	// then
	if team.String() != "MY" {
		t.Errorf("expected MY, got %s", team.String())
	}
	if team.IsEmpty() {
		t.Error("expected non-empty team")
	}
}

func TestNewTeam_Empty(t *testing.T) {
	// when
	team := domain.NewTeam("")

	// then
	if team.String() != "" {
		t.Errorf("expected empty string, got %q", team.String())
	}
	if !team.IsEmpty() {
		t.Error("expected empty team")
	}
}

func TestNewProject_NonEmpty(t *testing.T) {
	// when
	proj := domain.NewProject("Hades")

	// then
	if proj.String() != "Hades" {
		t.Errorf("expected Hades, got %s", proj.String())
	}
	if proj.IsEmpty() {
		t.Error("expected non-empty project")
	}
}

func TestNewProject_Empty(t *testing.T) {
	// when
	proj := domain.NewProject("")

	// then
	if proj.String() != "" {
		t.Errorf("expected empty string, got %q", proj.String())
	}
	if !proj.IsEmpty() {
		t.Error("expected empty project")
	}
}

func TestNewDays_Valid(t *testing.T) {
	// given
	raw := 30

	// when
	d, err := domain.NewDays(raw)

	// then
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if d.Int() != raw {
		t.Errorf("expected %d, got %d", raw, d.Int())
	}
}

func TestNewDays_RejectsZero(t *testing.T) {
	// when
	_, err := domain.NewDays(0)

	// then
	if err == nil {
		t.Fatal("expected error for zero Days")
	}
}

func TestNewDays_RejectsNegative(t *testing.T) {
	// when
	_, err := domain.NewDays(-1)

	// then
	if err == nil {
		t.Fatal("expected error for negative Days")
	}
}
