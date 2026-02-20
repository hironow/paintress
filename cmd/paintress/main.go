package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hironow/paintress"
)

var version = "dev"

// knownSubcommands lists all recognized subcommands.
var knownSubcommands = map[string]bool{
	"init":   true,
	"doctor": true,
	"issues": true,
}

// extractSubcommand separates args (os.Args[1:]) into a subcommand, a repo
// path, and remaining flag arguments. This allows flexible ordering:
//
//	paintress ./my-repo                       # run (default)
//	paintress --model opus ./my-repo          # flags before path
//	paintress ./my-repo --model opus          # flags after path
//	paintress init ./my-repo                  # subcommand
//	paintress doctor                          # subcommand (no path)
//	paintress --version                       # version flag
func extractSubcommand(args []string) (subcmd, repoPath string, flagArgs []string, err error) {
	subcmd = "run" // default

	if len(args) == 0 {
		return subcmd, "", nil, nil
	}

	// Check if first arg is a known subcommand
	if knownSubcommands[args[0]] {
		subcmd = args[0]
		args = args[1:]
	}

	// Separate remaining args into flags and positional (repoPath)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		// "--" terminates flag parsing; everything after is positional
		if arg == "--" {
			for _, rest := range args[i+1:] {
				if repoPath == "" {
					repoPath = rest
				}
			}
			break
		}
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			// --flag=value is self-contained; don't consume next arg
			if strings.Contains(arg, "=") {
				continue
			}
			// If this flag takes a value (next arg is not a flag), consume it
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				if isBoolFlag(arg) {
					// Bool flags optionally accept any strconv.ParseBool value
					// (true/false/1/0/t/f/T/F/TRUE/FALSE)
					if _, parseErr := strconv.ParseBool(args[i+1]); parseErr == nil {
						i++
						flagArgs = append(flagArgs, args[i])
					}
					continue
				}
				i++
				flagArgs = append(flagArgs, args[i])
			}
		} else {
			// First non-flag positional arg is the repo path
			if repoPath == "" {
				repoPath = arg
			}
		}
	}

	return subcmd, repoPath, flagArgs, nil
}

// parseOutputFlag extracts the --output value from flagArgs.
// Returns "text" when unspecified.
func parseOutputFlag(flagArgs []string) string {
	for i, arg := range flagArgs {
		if arg == "--output" && i+1 < len(flagArgs) {
			return flagArgs[i+1]
		}
		if strings.HasPrefix(arg, "--output=") {
			return strings.TrimPrefix(arg, "--output=")
		}
	}
	return "text"
}

// isBoolFlag checks if a flag argument is a known boolean flag.
func isBoolFlag(arg string) bool {
	name := strings.TrimLeft(arg, "-")
	switch name {
	case "version", "dry-run", "no-dev":
		return true
	}
	return false
}

func main() {
	os.Exit(run())
}

func run() int {
	subcmd, repoPath, flagArgs, err := extractSubcommand(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	switch subcmd {
	case "init":
		if repoPath == "" {
			fmt.Fprintf(os.Stderr, "Usage: paintress init <repo-path>\n")
			return 1
		}
		runInit(repoPath)
		return 0
	case "doctor":
		outputFmt := parseOutputFlag(flagArgs)
		runDoctor(outputFmt)
		return 0
	case "issues":
		if repoPath == "" {
			fmt.Fprintf(os.Stderr, "Usage: paintress issues <repo-path> [--output json|text]\n")
			return 1
		}
		outputFmt := parseOutputFlag(flagArgs)
		return runIssues(repoPath, outputFmt)
	}

	// Default: "run" subcommand
	cfg := parseFlags(repoPath, flagArgs)

	shutdownTracer := paintress.InitTracer("paintress", version)
	defer func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		shutdownTracer(shutdownCtx)
	}()

	if err := paintress.ValidateContinent(cfg.Continent); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		paintress.LogWarn("%s", fmt.Sprintf(paintress.Msg("signal_received"), sig))
		cancel()
	}()

	p := paintress.NewPaintress(cfg)
	return p.Run(ctx)
}

func parseFlags(repoPath string, args []string) paintress.Config {
	cfg := paintress.Config{}
	var lang string
	var showVersion bool

	fs := flag.NewFlagSet("paintress", flag.ExitOnError)
	fs.BoolVar(&showVersion, "version", false, "Show version and exit")
	fs.IntVar(&cfg.MaxExpeditions, "max-expeditions", 50, "Maximum number of expeditions")
	fs.IntVar(&cfg.TimeoutSec, "timeout", 1980, "Timeout per expedition in seconds (default: 33min)")
	fs.StringVar(&cfg.Model, "model", "opus", "Model(s) comma-separated for reserve: opus,sonnet,haiku")
	fs.StringVar(&cfg.BaseBranch, "base-branch", "main", "Base branch")
	fs.StringVar(&cfg.ClaudeCmd, "claude-cmd", paintress.DefaultClaudeCmd, "Claude Code CLI command name")
	fs.StringVar(&cfg.DevCmd, "dev-cmd", "npm run dev", "Dev server command")
	fs.StringVar(&cfg.DevDir, "dev-dir", "", "Dev server working directory (defaults to repo path)")
	fs.StringVar(&cfg.DevURL, "dev-url", "http://localhost:3000", "Dev server URL")
	fs.StringVar(&cfg.ReviewCmd, "review-cmd", "codex review --base main", "Code review command after PR creation")
	fs.IntVar(&cfg.Workers, "workers", 1, "Number of worktrees in pool (0 = direct execution)")
	fs.StringVar(&cfg.SetupCmd, "setup-cmd", "", "Command to run after worktree creation (e.g. 'bun install')")
	fs.BoolVar(&cfg.NoDev, "no-dev", false, "Skip dev server startup")
	fs.BoolVar(&cfg.DryRun, "dry-run", false, "Generate prompts only")
	fs.StringVar(&cfg.OutputFormat, "output", "text", "Output format: text, json")
	fs.StringVar(&lang, "lang", "en", "Output language: en, ja, fr")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: paintress <repo-path> [options]\n\n")
		fmt.Fprintf(os.Stderr, "The Paintress — drives the Expedition loop.\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  init <repo-path>   Initialize project configuration\n")
		fmt.Fprintf(os.Stderr, "  doctor             Check external command availability\n")
		fmt.Fprintf(os.Stderr, "  issues <repo-path> List Linear issues (JSONL to stdout)\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  <repo-path>    Target repository (The Continent)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  paintress init ./my-repo\n")
		fmt.Fprintf(os.Stderr, "  paintress doctor\n")
		fmt.Fprintf(os.Stderr, "  paintress ./my-repo\n")
		fmt.Fprintf(os.Stderr, "  paintress ./my-repo --model opus,sonnet --lang ja\n")
		fmt.Fprintf(os.Stderr, "  paintress ./my-repo --dry-run\n")
	}

	fs.Parse(args)

	// Derive --review-cmd default from --base-branch when not explicitly set
	reviewCmdExplicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "review-cmd" {
			reviewCmdExplicit = true
		}
	})
	if !reviewCmdExplicit {
		cfg.ReviewCmd = fmt.Sprintf("codex review --base %s", cfg.BaseBranch)
	}

	if showVersion {
		fmt.Printf("paintress %s\n", version)
		os.Exit(0)
	}

	if lang == "ja" || lang == "en" || lang == "fr" {
		paintress.Lang = lang
	}

	if repoPath == "" {
		fs.Usage()
		os.Exit(1)
	}

	continent, err := filepath.Abs(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
		os.Exit(1)
	}
	cfg.Continent = continent

	return cfg
}

func runInit(repoPath string) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "%s╔══════════════════════════════════════════════╗%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintf(os.Stderr, "%s║          Paintress Init                      ║%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintf(os.Stderr, "%s╚══════════════════════════════════════════════╝%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintln(os.Stderr)

	if err := paintress.RunInitWithReader(repoPath, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runDoctor(outputFmt string) {
	claudeCmd := paintress.DefaultClaudeCmd
	checks := paintress.RunDoctor(claudeCmd)

	allRequired := true
	for _, c := range checks {
		if c.Required && !c.OK {
			allRequired = false
			break
		}
	}

	if outputFmt == "json" {
		out, err := paintress.FormatDoctorJSON(checks)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(out)
		if !allRequired {
			os.Exit(1)
		}
		return
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "%s╔══════════════════════════════════════════════╗%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintf(os.Stderr, "%s║          Paintress Doctor                    ║%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintf(os.Stderr, "%s╚══════════════════════════════════════════════╝%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintln(os.Stderr)

	allOK := true
	for _, c := range checks {
		if c.OK {
			fmt.Fprintf(os.Stderr, "  %s✓%s  %-12s %s (%s)\n", paintress.ColorGreen, paintress.ColorReset, c.Name, c.Version, c.Path)
		} else {
			marker := "✗"
			color := paintress.ColorRed
			label := "MISSING (required)"
			if !c.Required {
				label = "not found (optional)"
				color = paintress.ColorYellow
			} else {
				allOK = false
			}
			fmt.Fprintf(os.Stderr, "  %s%s%s  %-12s %s\n", color, marker, paintress.ColorReset, c.Name, label)
		}
	}
	fmt.Fprintln(os.Stderr)

	if !allOK {
		fmt.Fprintf(os.Stderr, "Some required commands are missing. Install them and try again.\n")
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "All checks passed.")
}

func runIssues(repoPath, outputFmt string) int {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
		return 1
	}

	cfg, err := paintress.LoadProjectConfig(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: load config: %v\n", err)
		return 1
	}
	if cfg.Linear.Team == "" {
		fmt.Fprintf(os.Stderr, "Error: linear.team not set in %s\n", paintress.ProjectConfigPath(absPath))
		fmt.Fprintf(os.Stderr, "Run 'paintress init %s' first.\n", repoPath)
		return 1
	}

	apiKey := os.Getenv("LINEAR_API_KEY")
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "Error: LINEAR_API_KEY environment variable is required\n")
		return 1
	}

	issues, err := paintress.FetchIssues(paintress.LinearAPIEndpoint, apiKey, cfg.Linear.Team, cfg.Linear.Project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	paintress.LogInfo("fetched %d issues from %s", len(issues), cfg.Linear.Team)

	switch outputFmt {
	case "json":
		out, err := paintress.FormatIssuesJSON(issues)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println(out)
	default:
		out := paintress.FormatIssuesJSONL(issues)
		if out != "" {
			fmt.Println(out)
		}
	}
	return 0
}
