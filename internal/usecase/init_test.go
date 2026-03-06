package usecase_test

import (
	"fmt"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase"
)

type stubInitRunner struct {
	called   bool
	repoPath string
	team     string
	project  string
	err      error
}

func (s *stubInitRunner) InitProject(repoPath, team, project string) error {
	s.called = true
	s.repoPath = repoPath
	s.team = team
	s.project = project
	return s.err
}

func TestRunInit_ValidCommand(t *testing.T) {
	runner := &stubInitRunner{}
	rp, _ := domain.NewRepoPath("/tmp/repo")
	cmd := domain.NewInitCommand(rp, domain.NewTeam("MY"), domain.NewProject("Hades"))

	err := usecase.RunInit(cmd, runner)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !runner.called {
		t.Fatal("expected InitProject to be called")
	}
	if runner.repoPath != "/tmp/repo" {
		t.Errorf("expected repoPath /tmp/repo, got %q", runner.repoPath)
	}
	if runner.team != "MY" {
		t.Errorf("expected team MY, got %q", runner.team)
	}
	if runner.project != "Hades" {
		t.Errorf("expected project Hades, got %q", runner.project)
	}
}

func TestRunInit_RunnerError(t *testing.T) {
	runner := &stubInitRunner{err: fmt.Errorf("continent invalid")}
	rp, _ := domain.NewRepoPath("/tmp/repo")
	cmd := domain.NewInitCommand(rp, domain.NewTeam(""), domain.NewProject(""))

	err := usecase.RunInit(cmd, runner)

	if err == nil {
		t.Fatal("expected error from runner")
	}
}
