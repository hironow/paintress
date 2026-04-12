package usecase_test

import (
	"fmt"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase"
	"github.com/hironow/paintress/internal/usecase/port"
)

type stubInitRunner struct {
	called  bool
	baseDir string
	config  port.InitConfig
	err     error
}

func (s *stubInitRunner) InitProject(baseDir string, opts ...port.InitOption) ([]string, error) {
	s.called = true
	s.baseDir = baseDir
	s.config = port.ApplyInitOptions(opts...)
	return nil, s.err
}

func TestRunInit_ValidCommand(t *testing.T) {
	runner := &stubInitRunner{}
	rp, _ := domain.NewRepoPath("/tmp/repo")
	cmd := domain.NewInitCommand(rp, domain.NewTeam("MY"), domain.NewProject("Hades"))

	_, err := usecase.RunInit(cmd, runner)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !runner.called {
		t.Fatal("expected InitProject to be called")
	}
	if runner.baseDir != "/tmp/repo" {
		t.Errorf("expected baseDir /tmp/repo, got %q", runner.baseDir)
	}
	if runner.config.Team != "MY" {
		t.Errorf("expected team MY, got %q", runner.config.Team)
	}
	if runner.config.Project != "Hades" {
		t.Errorf("expected project Hades, got %q", runner.config.Project)
	}
}

func TestRunInit_RunnerError(t *testing.T) {
	runner := &stubInitRunner{err: fmt.Errorf("continent invalid")}
	rp, _ := domain.NewRepoPath("/tmp/repo")
	cmd := domain.NewInitCommand(rp, domain.NewTeam(""), domain.NewProject(""))

	_, err := usecase.RunInit(cmd, runner)

	if err == nil {
		t.Fatal("expected error from runner")
	}
}
