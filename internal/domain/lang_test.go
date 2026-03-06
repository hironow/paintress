package domain
// white-box-reason: internal state: tests unexported language global variable switching

import (
	"sync"
	"testing"
)

func TestMsg_English_Default(t *testing.T) {
	orig := Lang
	defer func() { Lang = orig }()
	Lang = "en"

	got := Msg("grad_attack")
	if !containsStr(got, "GRADIENT ATTACK") {
		t.Errorf("English grad_attack should contain GRADIENT ATTACK, got %q", got)
	}
}

func TestMsg_Japanese(t *testing.T) {
	orig := Lang
	defer func() { Lang = orig }()
	Lang = "ja"

	got := Msg("grad_empty")
	if !containsStr(got, "ゲージ空") {
		t.Errorf("Japanese grad_empty should contain ゲージ空, got %q", got)
	}
}

func TestMsg_MissingKey(t *testing.T) {
	got := Msg("nonexistent_key_xyz")
	if got != "[missing: nonexistent_key_xyz]" {
		t.Errorf("missing key should return [missing: ...], got %q", got)
	}
}

func TestMsg_French(t *testing.T) {
	orig := Lang
	defer func() { Lang = orig }()
	Lang = "fr"

	got := Msg("grad_empty")
	if !containsStr(got, "Gradient vide") {
		t.Errorf("French grad_empty should contain 'Gradient vide', got %q", got)
	}
}

func TestMsg_FallbackToEnglish(t *testing.T) {
	orig := Lang
	defer func() { Lang = orig }()
	Lang = "de" // unsupported language

	got := Msg("grad_attack")
	if !containsStr(got, "GRADIENT ATTACK") {
		t.Errorf("unsupported lang should fall back to English, got %q", got)
	}
}

func TestMsg_AllKeysHaveEnglish(t *testing.T) {
	for key, variants := range messages {
		if _, ok := variants["en"]; !ok {
			t.Errorf("key %q is missing English translation", key)
		}
	}
}

func TestMsg_AllKeysHaveJapanese(t *testing.T) {
	for key, variants := range messages {
		if _, ok := variants["ja"]; !ok {
			t.Errorf("key %q is missing Japanese translation", key)
		}
	}
}

func TestMsg_AllKeysHaveFrench(t *testing.T) {
	for key, variants := range messages {
		if _, ok := variants["fr"]; !ok {
			t.Errorf("key %q is missing French translation", key)
		}
	}
}

// --- from race_test.go ---

func TestLang_ConcurrentMsgReads(t *testing.T) {
	orig := Lang
	defer func() { Lang = orig }()

	var wg sync.WaitGroup

	// Concurrent reads of Msg() — all reading the same global Lang
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg := Msg("grad_attack")
			if msg == "" {
				t.Error("Msg should never return empty")
			}
		}()
	}
	wg.Wait()
}
