package main

import (
	"testing"
)

func TestRunDoctorChecks_AllSet(t *testing.T) {
	// given
	env := map[string]string{
		"PAINTRESS_TG_TOKEN":   "123456:ABC-DEF",
		"PAINTRESS_TG_CHAT_ID": "987654321",
	}

	// when
	checks := runDoctorChecks(func(key string) string { return env[key] })

	// then
	for _, c := range checks {
		if !c.OK {
			t.Errorf("check %q should be OK", c.Name)
		}
	}
}

func TestRunDoctorChecks_MissingToken(t *testing.T) {
	// given
	env := map[string]string{
		"PAINTRESS_TG_CHAT_ID": "123",
	}

	// when
	checks := runDoctorChecks(func(key string) string { return env[key] })

	// then
	var tokenCheck *doctorCheck
	for i := range checks {
		if checks[i].Name == "PAINTRESS_TG_TOKEN" {
			tokenCheck = &checks[i]
			break
		}
	}
	if tokenCheck == nil {
		t.Fatal("expected PAINTRESS_TG_TOKEN check")
	}
	if tokenCheck.OK {
		t.Error("missing token should not be OK")
	}
}

func TestRunDoctorChecks_InvalidChatID(t *testing.T) {
	// given
	env := map[string]string{
		"PAINTRESS_TG_TOKEN":   "token",
		"PAINTRESS_TG_CHAT_ID": "not-a-number",
	}

	// when
	checks := runDoctorChecks(func(key string) string { return env[key] })

	// then
	var chatCheck *doctorCheck
	for i := range checks {
		if checks[i].Name == "PAINTRESS_TG_CHAT_ID" {
			chatCheck = &checks[i]
			break
		}
	}
	if chatCheck == nil {
		t.Fatal("expected PAINTRESS_TG_CHAT_ID check")
	}
	if chatCheck.OK {
		t.Error("non-integer chat ID should not be OK")
	}
	if chatCheck.Hint == "" {
		t.Error("expected hint for invalid chat ID")
	}
}

func TestRunDoctorChecks_MasksToken(t *testing.T) {
	// given
	env := map[string]string{
		"PAINTRESS_TG_TOKEN":   "123456:ABC-DEF-GHI-JKL",
		"PAINTRESS_TG_CHAT_ID": "123",
	}

	// when
	checks := runDoctorChecks(func(key string) string { return env[key] })

	// then
	var tokenCheck *doctorCheck
	for i := range checks {
		if checks[i].Name == "PAINTRESS_TG_TOKEN" {
			tokenCheck = &checks[i]
			break
		}
	}
	if tokenCheck == nil {
		t.Fatal("expected PAINTRESS_TG_TOKEN check")
	}
	if tokenCheck.Value == "123456:ABC-DEF-GHI-JKL" {
		t.Error("token should be masked, not shown in full")
	}
	if tokenCheck.Value == "" {
		t.Error("masked token should not be empty")
	}
}
