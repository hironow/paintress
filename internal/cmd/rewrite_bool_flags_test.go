package cmd

import (
	"slices"
	"testing"
)

func TestRewriteBoolFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			"no bool flags",
			[]string{"--model", "opus", "./repo"},
			[]string{"--model", "opus", "./repo"},
		},
		{
			"--dry-run without value (toggle)",
			[]string{"--dry-run", "./repo"},
			[]string{"--dry-run", "./repo"},
		},
		{
			"--dry-run false (space-separated)",
			[]string{"--dry-run", "false", "./repo"},
			[]string{"--dry-run=false", "./repo"},
		},
		{
			"--dry-run true (space-separated)",
			[]string{"--dry-run", "true", "./repo"},
			[]string{"--dry-run=true", "./repo"},
		},
		{
			"--execute false",
			[]string{"archive-prune", "--execute", "false", "./repo"},
			[]string{"archive-prune", "--execute=false", "./repo"},
		},
		{
			"--dry-run=false (already correct)",
			[]string{"--dry-run=false", "./repo"},
			[]string{"--dry-run=false", "./repo"},
		},
		{
			"--no-dev 0 (ParseBool compatible)",
			[]string{"--no-dev", "0", "./repo"},
			[]string{"--no-dev=0", "./repo"},
		},
		{
			"--dry-run followed by non-bool value",
			[]string{"--dry-run", "something", "./repo"},
			[]string{"--dry-run", "something", "./repo"},
		},
		{
			"--dry-run at end (no next arg)",
			[]string{"run", "--dry-run"},
			[]string{"run", "--dry-run"},
		},
		{
			"multiple bool flags",
			[]string{"--dry-run", "true", "--no-dev", "false", "./repo"},
			[]string{"--dry-run=true", "--no-dev=false", "./repo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RewriteBoolFlags(tt.args)
			if !slices.Equal(got, tt.want) {
				t.Errorf("RewriteBoolFlags(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
