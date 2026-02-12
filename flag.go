package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExpeditionFlag represents the checkpoint on the Continent.
type ExpeditionFlag struct {
	LastExpedition int
	LastUpdated    string
	LastIssue      string
	LastStatus     string
	Remaining      string
}

func FlagPath(continent string) string {
	return filepath.Join(continent, ".expedition", "flag.md")
}

func ReadFlag(continent string) ExpeditionFlag {
	f := ExpeditionFlag{Remaining: "?"}
	file, err := os.Open(FlagPath(continent))
	if err != nil {
		return f
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if k, v, ok := parseKV(line); ok {
			switch k {
			case "last_expedition":
				fmt.Sscanf(v, "%d", &f.LastExpedition)
			case "last_updated":
				f.LastUpdated = v
			case "last_issue":
				f.LastIssue = v
			case "last_status":
				f.LastStatus = v
			case "remaining_issues":
				f.Remaining = v
			}
		}
	}
	return f
}

func WriteFlag(continent string, expNum int, issueID, status, remaining string) error {
	content := fmt.Sprintf(`# Expedition Flag
# Last planted by Expedition #%d

last_expedition: %d
last_updated: %s
last_issue: %s
last_status: %s
remaining_issues: %s
`, expNum, expNum, time.Now().Format("2006-01-02 15:04:05"), issueID, status, remaining)

	return os.WriteFile(FlagPath(continent), []byte(content), 0644)
}

func parseKV(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "#") || line == "" {
		return "", "", false
	}
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}
