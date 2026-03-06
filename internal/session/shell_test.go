package session_test

import (
	"runtime"
	"testing"

	"github.com/hironow/paintress/internal/session"
)

func TestShellName_ReturnsNonEmpty(t *testing.T) {
	name := session.ExportShellName()
	if name == "" {
		t.Error("shellName() returned empty string")
	}
}

func TestShellFlag_ReturnsNonEmpty(t *testing.T) {
	flag := session.ExportShellFlag()
	if flag == "" {
		t.Error("shellFlag() returned empty string")
	}
}

func TestShellName_MatchesPlatform(t *testing.T) {
	name := session.ExportShellName()
	switch runtime.GOOS {
	case "windows":
		if name != "cmd" {
			t.Errorf("on windows, shellName() = %q, want %q", name, "cmd")
		}
	default:
		if name != "sh" {
			t.Errorf("on %s, shellName() = %q, want %q", runtime.GOOS, name, "sh")
		}
	}
}

func TestShellFlag_MatchesPlatform(t *testing.T) {
	flag := session.ExportShellFlag()
	switch runtime.GOOS {
	case "windows":
		if flag != "/c" {
			t.Errorf("on windows, shellFlag() = %q, want %q", flag, "/c")
		}
	default:
		if flag != "-c" {
			t.Errorf("on %s, shellFlag() = %q, want %q", runtime.GOOS, flag, "-c")
		}
	}
}
