package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseKV_Normal(t *testing.T) {
	k, v, ok := parseKV("last_expedition: 5")
	if !ok {
		t.Fatal("expected ok")
	}
	if k != "last_expedition" || v != "5" {
		t.Errorf("got k=%q v=%q", k, v)
	}
}

func TestParseKV_Comment(t *testing.T) {
	_, _, ok := parseKV("# this is a comment")
	if ok {
		t.Error("comments should return false")
	}
}

func TestParseKV_Empty(t *testing.T) {
	_, _, ok := parseKV("")
	if ok {
		t.Error("empty line should return false")
	}
}

func TestParseKV_NoColon(t *testing.T) {
	_, _, ok := parseKV("no colon here")
	if ok {
		t.Error("line without colon should return false")
	}
}

func TestParseKV_ValueWithColon(t *testing.T) {
	k, v, ok := parseKV("last_updated: 2024-01-01 12:00:00")
	if !ok {
		t.Fatal("expected ok")
	}
	if k != "last_updated" {
		t.Errorf("key = %q", k)
	}
	if v != "2024-01-01 12:00:00" {
		t.Errorf("value = %q, want time string with colon", v)
	}
}

func TestParseKV_WhitespaceOnly(t *testing.T) {
	_, _, ok := parseKV("   ")
	if ok {
		t.Error("whitespace-only line should return false")
	}
}

func TestReadFlag_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	f := ReadFlag(dir)
	if f.Remaining != "?" {
		t.Errorf("default Remaining should be '?', got %q", f.Remaining)
	}
	if f.LastExpedition != 0 {
		t.Errorf("default LastExpedition should be 0, got %d", f.LastExpedition)
	}
}

func TestFlagPath(t *testing.T) {
	p := FlagPath("/some/repo")
	want := filepath.Join("/some/repo", ".expedition", "flag.md")
	if p != want {
		t.Errorf("FlagPath = %q, want %q", p, want)
	}
}

func TestWriteFlag_AllFields(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	WriteFlag(dir, 10, "AWE-99", "success", "0")
	f := ReadFlag(dir)

	if f.LastExpedition != 10 {
		t.Errorf("LastExpedition = %d, want 10", f.LastExpedition)
	}
	if f.LastIssue != "AWE-99" {
		t.Errorf("LastIssue = %q, want AWE-99", f.LastIssue)
	}
	if f.LastStatus != "success" {
		t.Errorf("LastStatus = %q, want success", f.LastStatus)
	}
	if f.Remaining != "0" {
		t.Errorf("Remaining = %q, want 0", f.Remaining)
	}
	if f.LastUpdated == "" {
		t.Error("LastUpdated should not be empty")
	}
}

func TestWriteFlag_Overwrite(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)

	WriteFlag(dir, 1, "AWE-1", "success", "10")
	WriteFlag(dir, 2, "AWE-2", "failed", "9")

	f := ReadFlag(dir)
	if f.LastExpedition != 2 {
		t.Errorf("should reflect latest write, got %d", f.LastExpedition)
	}
	if f.LastIssue != "AWE-2" {
		t.Errorf("LastIssue = %q, want AWE-2", f.LastIssue)
	}
}
