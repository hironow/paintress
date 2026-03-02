// fake-claude is a test double for the claude CLI used in paintress E2E tests.
// It accepts the same flags as real claude (--model, --print, -p, etc.)
// and writes canned responses to stdout based on prompt content.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

// promptCounter tracks prompt log file numbering.
var promptCounter atomic.Int64

const defaultResponse = `I'll analyze the issues and work on them systematically.

## Issue Analysis

Looking at the assigned issues, I'll start with the highest priority item.

### Changes Made

1. Updated the configuration file
2. Fixed the validation logic
3. Added missing test coverage

All changes have been committed and pushed.
`

const failureResponse = `I encountered an error while processing the request.

The operation could not be completed due to a configuration issue.
Please check the project settings and try again.
`

func main() {
	prompt := extractPrompt(os.Args[1:])

	// Log prompt if FAKE_CLAUDE_PROMPT_LOG_DIR is set.
	if logDir := os.Getenv("FAKE_CLAUDE_PROMPT_LOG_DIR"); logDir != "" {
		n := promptCounter.Add(1)
		name := filepath.Join(logDir, fmt.Sprintf("prompt-%03d.txt", n))
		_ = os.MkdirAll(logDir, 0755)
		_ = os.WriteFile(name, []byte(prompt), 0644)
	}

	// Simulate failure if FAKE_CLAUDE_FAIL_PATTERN matches.
	if pat := os.Getenv("FAKE_CLAUDE_FAIL_PATTERN"); pat != "" {
		if strings.Contains(prompt, pat) {
			fmt.Fprint(os.Stdout, failureResponse)
			os.Exit(1)
		}
	}

	// Select fixture based on prompt content.
	response := selectResponse(prompt)
	fmt.Fprint(os.Stdout, response)
}

// extractPrompt finds the -p flag value from args.
func extractPrompt(args []string) string {
	for i, arg := range args {
		if arg == "-p" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// selectResponse picks a canned response based on prompt keywords.
func selectResponse(prompt string) string {
	switch {
	case strings.Contains(prompt, "FAKE_EMPTY"):
		return ""
	case strings.Contains(prompt, "rate limit"):
		return "Rate limit exceeded. Please try again later.\n"
	default:
		return defaultResponse
	}
}
