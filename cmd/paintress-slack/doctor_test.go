package main

import (
	"testing"
)

func TestRunDoctorChecks_AllSet(t *testing.T) {
	// given
	env := map[string]string{
		"PAINTRESS_SLACK_TOKEN":      "xoxb-123-456-abc",
		"PAINTRESS_SLACK_CHANNEL_ID": "C01234567",
		"PAINTRESS_SLACK_APP_TOKEN":  "xapp-1-A0123-def",
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
		"PAINTRESS_SLACK_CHANNEL_ID": "C01234567",
		"PAINTRESS_SLACK_APP_TOKEN":  "xapp-1-A0123-def",
	}

	// when
	checks := runDoctorChecks(func(key string) string { return env[key] })

	// then
	var tokenCheck *doctorCheck
	for i := range checks {
		if checks[i].Name == "PAINTRESS_SLACK_TOKEN" {
			tokenCheck = &checks[i]
			break
		}
	}
	if tokenCheck == nil {
		t.Fatal("expected PAINTRESS_SLACK_TOKEN check")
	}
	if tokenCheck.OK {
		t.Error("missing token should not be OK")
	}
}

func TestRunDoctorChecks_MissingChannelID(t *testing.T) {
	// given
	env := map[string]string{
		"PAINTRESS_SLACK_TOKEN":     "xoxb-123-456-abc",
		"PAINTRESS_SLACK_APP_TOKEN": "xapp-1-A0123-def",
	}

	// when
	checks := runDoctorChecks(func(key string) string { return env[key] })

	// then
	var chCheck *doctorCheck
	for i := range checks {
		if checks[i].Name == "PAINTRESS_SLACK_CHANNEL_ID" {
			chCheck = &checks[i]
			break
		}
	}
	if chCheck == nil {
		t.Fatal("expected PAINTRESS_SLACK_CHANNEL_ID check")
	}
	if chCheck.OK {
		t.Error("missing channel ID should not be OK")
	}
}

func TestRunDoctorChecks_MissingAppToken(t *testing.T) {
	// given
	env := map[string]string{
		"PAINTRESS_SLACK_TOKEN":      "xoxb-123-456-abc",
		"PAINTRESS_SLACK_CHANNEL_ID": "C01234567",
	}

	// when
	checks := runDoctorChecks(func(key string) string { return env[key] })

	// then
	var appCheck *doctorCheck
	for i := range checks {
		if checks[i].Name == "PAINTRESS_SLACK_APP_TOKEN" {
			appCheck = &checks[i]
			break
		}
	}
	if appCheck == nil {
		t.Fatal("expected PAINTRESS_SLACK_APP_TOKEN check")
	}
	if appCheck.OK {
		t.Error("missing app token should not be OK")
	}
}

func TestRunDoctorChecks_MasksSecrets(t *testing.T) {
	// given
	env := map[string]string{
		"PAINTRESS_SLACK_TOKEN":      "xoxb-123-456-abcdef-long-token",
		"PAINTRESS_SLACK_CHANNEL_ID": "C01234567",
		"PAINTRESS_SLACK_APP_TOKEN":  "xapp-1-A0123-very-long-token",
	}

	// when
	checks := runDoctorChecks(func(key string) string { return env[key] })

	// then
	for _, c := range checks {
		switch c.Name {
		case "PAINTRESS_SLACK_TOKEN":
			if c.Value == env["PAINTRESS_SLACK_TOKEN"] {
				t.Error("token should be masked, not shown in full")
			}
			if c.Value == "" {
				t.Error("masked token should not be empty")
			}
		case "PAINTRESS_SLACK_CHANNEL_ID":
			if c.Value != env["PAINTRESS_SLACK_CHANNEL_ID"] {
				t.Errorf("channel ID should not be masked, got %q", c.Value)
			}
		case "PAINTRESS_SLACK_APP_TOKEN":
			if c.Value == env["PAINTRESS_SLACK_APP_TOKEN"] {
				t.Error("app token should be masked, not shown in full")
			}
			if c.Value == "" {
				t.Error("masked app token should not be empty")
			}
		}
	}
}
