package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMissionText_English(t *testing.T) {
	orig := Lang
	defer func() { Lang = orig }()
	Lang = "en"

	text := MissionText()
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
	orig := Lang
	defer func() { Lang = orig }()
	Lang = "ja"

	text := MissionText()
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
	orig := Lang
	defer func() { Lang = orig }()
	Lang = "fr"

	text := MissionText()
	if !containsStr(text, "engagement") {
		t.Error("French mission should contain 'engagement'")
	}
	if containsStr(text, "行動規範") {
		t.Error("French mission should not contain Japanese")
	}
}

func TestMissionText_FallbackToEnglish(t *testing.T) {
	orig := Lang
	defer func() { Lang = orig }()
	Lang = "de"

	text := MissionText()
	if !containsStr(text, "Rules of Engagement") {
		t.Error("unsupported lang should fall back to English")
	}
}

func TestWriteMission(t *testing.T) {
	dir := t.TempDir()
	expDir := filepath.Join(dir, ".expedition")
	os.MkdirAll(expDir, 0755)

	orig := Lang
	defer func() { Lang = orig }()
	Lang = "en"

	err := WriteMission(dir)
	if err != nil {
		t.Fatalf("WriteMission() error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(expDir, "mission.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(content), "Rules of Engagement") {
		t.Error("written mission.md should contain English content")
	}
}

func TestWriteMission_Japanese(t *testing.T) {
	dir := t.TempDir()
	expDir := filepath.Join(dir, ".expedition")
	os.MkdirAll(expDir, 0755)

	orig := Lang
	defer func() { Lang = orig }()
	Lang = "ja"

	err := WriteMission(dir)
	if err != nil {
		t.Fatalf("WriteMission() error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(expDir, "mission.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(string(content), "行動規範") {
		t.Error("written mission.md should contain Japanese content")
	}
}

func TestWriteMission_OverwritesPrevious(t *testing.T) {
	dir := t.TempDir()
	expDir := filepath.Join(dir, ".expedition")
	os.MkdirAll(expDir, 0755)

	orig := Lang
	defer func() { Lang = orig }()

	// Write Japanese first
	Lang = "ja"
	WriteMission(dir)

	// Overwrite with English
	Lang = "en"
	WriteMission(dir)

	content, err := os.ReadFile(filepath.Join(expDir, "mission.md"))
	if err != nil {
		t.Fatal(err)
	}
	if containsStr(string(content), "行動規範") {
		t.Error("overwritten mission.md should not contain Japanese")
	}
	if !containsStr(string(content), "Rules of Engagement") {
		t.Error("overwritten mission.md should contain English")
	}
}

func TestMissionPath(t *testing.T) {
	got := MissionPath("/some/repo")
	want := filepath.Join("/some/repo", ".expedition", "mission.md")
	if got != want {
		t.Errorf("MissionPath = %q, want %q", got, want)
	}
}
