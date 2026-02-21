package cmd

import "testing"

func TestNeedsDefaultRun(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"empty args", []string{}, false},
		{"known subcommand run", []string{"run", "./repo"}, false},
		{"known subcommand version", []string{"version"}, false},
		{"known subcommand doctor", []string{"doctor"}, false},
		{"known subcommand with flag", []string{"version", "--json"}, false},
		{"bare repo path", []string{"./repo"}, true},
		{"unknown flag (run-specific)", []string{"--model", "opus", "./repo"}, true},
		{"root bool flag then path", []string{"--verbose", "./repo"}, true},
		{"root bool flag then subcommand", []string{"--verbose", "version"}, false},
		{"special flag --version", []string{"--version"}, false},
		{"special flag --help", []string{"--help"}, false},
		{"special flag -h", []string{"-h"}, false},
		{"root string flag then path", []string{"-o", "json", "./repo"}, true},
		{"root string flag then subcommand", []string{"-o", "json", "version"}, false},
		{"root string flag=value then path", []string{"--output=json", "./repo"}, true},
		{"root string flag=value then subcommand", []string{"--output=json", "doctor"}, false},
		{"--version with extra path", []string{"--version", "/path/to/repo"}, false},
		{"--help with extra path", []string{"--help", "/path/to/repo"}, false},
		{"-h with extra path", []string{"-h", "/path/to/repo"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := NewRootCommand()
			got := NeedsDefaultRun(rootCmd, tt.args)
			if got != tt.want {
				t.Errorf("NeedsDefaultRun(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
