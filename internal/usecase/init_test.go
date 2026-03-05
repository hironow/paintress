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
	cmd := domain.InitCommand{RepoPath: "/tmp/repo", Team: "MY", Project: "Hades"}

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

func TestRunInit_EmptyRepoPath(t *testing.T) {
	runner := &stubInitRunner{}
	cmd := domain.InitCommand{RepoPath: ""}

	err := usecase.RunInit(cmd, runner)

	if err == nil {
		t.Fatal("expected error for empty RepoPath")
	}
	if runner.called {
		t.Fatal("expected InitProject not to be called")
	}
}

func TestRunInit_RunnerError(t *testing.T) {
	runner := &stubInitRunner{err: fmt.Errorf("continent invalid")}
	cmd := domain.InitCommand{RepoPath: "/tmp/repo"}

	err := usecase.RunInit(cmd, runner)

	if err == nil {
		t.Fatal("expected error from runner")
	}
}
