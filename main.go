package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

// version is set at build time via ldflags.
var version = "dev"

const defaultClaudeCmd = "claude"

type Config struct {
	Continent      string
	MaxExpeditions int
	TimeoutSec     int
	Model          string // "opus" or "opus,sonnet,haiku" for reserve party
	BaseBranch     string
	ClaudeCmd      string // CLI command name for Claude Code (e.g. "claude", "cc-p")
	DevCmd         string
	DevDir         string // working directory for dev server (defaults to Continent)
	DevURL         string
	ReviewCmd      string // Code review command (e.g. "codex review --base main")
	DryRun         bool
}

func main() {
	cfg := parseFlags()

	if err := validateContinent(cfg.Continent); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		LogWarn("%s", fmt.Sprintf(Msg("signal_received"), sig))
		cancel()
	}()

	p := NewPaintress(cfg)
	os.Exit(p.Run(ctx))
}

func parseFlags() Config {
	cfg := Config{}
	var lang string
	var showVersion bool

	flag.BoolVar(&showVersion, "version", false, "Show version and exit")
	flag.IntVar(&cfg.MaxExpeditions, "max-expeditions", 50, "Maximum number of expeditions")
	flag.IntVar(&cfg.TimeoutSec, "timeout", 1980, "Timeout per expedition in seconds (default: 33min)")
	flag.StringVar(&cfg.Model, "model", "opus", "Model(s) comma-separated for reserve: opus,sonnet,haiku")
	flag.StringVar(&cfg.BaseBranch, "base-branch", "main", "Base branch")
	flag.StringVar(&cfg.ClaudeCmd, "claude-cmd", defaultClaudeCmd, "Claude Code CLI command name")
	flag.StringVar(&cfg.DevCmd, "dev-cmd", "npm run dev", "Dev server command")
	flag.StringVar(&cfg.DevDir, "dev-dir", "", "Dev server working directory (defaults to repo path)")
	flag.StringVar(&cfg.DevURL, "dev-url", "http://localhost:3000", "Dev server URL")
	flag.StringVar(&cfg.ReviewCmd, "review-cmd", "codex review --base main", "Code review command after PR creation")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "Generate prompts only")
	flag.StringVar(&lang, "lang", "en", "Output language: en, ja, fr")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: paintress <repo-path> [options]\n\n")
		fmt.Fprintf(os.Stderr, "The Paintress — drives the Expedition loop.\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  <repo-path>    Target repository (The Continent)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  paintress ./my-repo\n")
		fmt.Fprintf(os.Stderr, "  paintress ./my-repo --model opus,sonnet --lang ja\n")
		fmt.Fprintf(os.Stderr, "  paintress ./my-repo --dry-run\n")
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("paintress %s\n", version)
		os.Exit(0)
	}

	if lang == "ja" || lang == "en" || lang == "fr" {
		Lang = lang
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	continent, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
		os.Exit(1)
	}
	cfg.Continent = continent

	return cfg
}

func validateContinent(continent string) error {
	journalDir := filepath.Join(continent, ".expedition", "journal")
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return err
	}

	// Ensure .logs/ is gitignored
	gitignore := filepath.Join(continent, ".expedition", ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		os.WriteFile(gitignore, []byte(".logs/\n"), 0644)
	}
	return nil
}
