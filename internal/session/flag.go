package session

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// ReadFlag reads the expedition flag from the Continent's flag.md file.
func ReadFlag(continent string) domain.ExpeditionFlag {
	f := domain.ExpeditionFlag{Remaining: "?"}
	path := domain.FlagPath(continent)
	file, err := os.Open(path)
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
			case "current_issue":
				f.CurrentIssue = v
			case "current_title":
				f.CurrentTitle = v
			case "mid_high_severity":
				fmt.Sscanf(v, "%d", &f.MidHighSeverity)
			}
		}
	}
	return f
}

// WriteFlag writes an expedition flag checkpoint file.
func WriteFlag(continent string, expNum int, issueID, status, remaining string, midHighSeverity int) error { // nosemgrep: domain-primitives.multiple-string-params-go -- issueID/status/remaining are semantically distinct [permanent]
	issueID = sanitizeFlagValue(issueID)
	status = sanitizeFlagValue(status)
	remaining = sanitizeFlagValue(remaining)
	content := fmt.Sprintf(`# Expedition Flag
# This checkpoint summarizes the previous Expedition's outcome.
# Use it as context when starting your mission.
# Last planted by Expedition #%d

last_expedition: %d
last_updated: %s
# The issue attempted in the previous Expedition (check last_status for outcome)
last_issue: %s
last_status: %s
remaining_issues: %s
mid_high_severity: %d
`, expNum, expNum, time.Now().Format("2006-01-02 15:04:05"), issueID, status, remaining, midHighSeverity)

	return os.WriteFile(domain.FlagPath(continent), []byte(content), 0644)
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

func sanitizeFlagValue(value string) string {
	replacer := strings.NewReplacer("\r", " ", "\n", " ")
	return strings.TrimSpace(replacer.Replace(value))
}
