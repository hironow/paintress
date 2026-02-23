package main

import (
	"testing"
)

func TestRunDoctorChecks_AllSet(t *testing.T) {
	// given
	env := map[string]string{
		"PAINTRESS_DISCORD_TOKEN":      "Bot MTIz.abc.xyz",
		"PAINTRESS_DISCORD_CHANNEL_ID": "123456789",
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
		"PAINTRESS_DISCORD_CHANNEL_ID": "123",
	}

	// when
	checks := runDoctorChecks(func(key string) string { return env[key] })

	// then
	var tokenCheck *doctorCheck
	for i := range checks {
		if checks[i].Name == "PAINTRESS_DISCORD_TOKEN" {
			tokenCheck = &checks[i]
			break
		}
	}
	if tokenCheck == nil {
		t.Fatal("expected PAINTRESS_DISCORD_TOKEN check")
	}
	if tokenCheck.OK {
		t.Error("missing token should not be OK")
	}
}

func TestRunDoctorChecks_MissingChannelID(t *testing.T) {
	// given
	env := map[string]string{
		"PAINTRESS_DISCORD_TOKEN": "token",
	}

	// when
	checks := runDoctorChecks(func(key string) string { return env[key] })

	// then
	var chCheck *doctorCheck
	for i := range checks {
		if checks[i].Name == "PAINTRESS_DISCORD_CHANNEL_ID" {
			chCheck = &checks[i]
			break
		}
	}
	if chCheck == nil {
		t.Fatal("expected PAINTRESS_DISCORD_CHANNEL_ID check")
	}
	if chCheck.OK {
		t.Error("missing channel ID should not be OK")
	}
}

func TestRunDoctorChecks_MasksToken(t *testing.T) {
	// given
	env := map[string]string{
		"PAINTRESS_DISCORD_TOKEN":      "Bot MTIz.abc.xyz.very-long-token",
		"PAINTRESS_DISCORD_CHANNEL_ID": "123",
	}

	// when
	checks := runDoctorChecks(func(key string) string { return env[key] })

	// then
	var tokenCheck *doctorCheck
	for i := range checks {
		if checks[i].Name == "PAINTRESS_DISCORD_TOKEN" {
			tokenCheck = &checks[i]
			break
		}
	}
	if tokenCheck == nil {
		t.Fatal("expected PAINTRESS_DISCORD_TOKEN check")
	}
	if tokenCheck.Value == env["PAINTRESS_DISCORD_TOKEN"] {
		t.Error("token should be masked, not shown in full")
	}
	if tokenCheck.Value == "" {
		t.Error("masked token should not be empty")
	}
}
