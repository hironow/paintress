package paintress

import (
	"context"
	"time"
)

// watchFlag polls flag.md for current_issue changes and invokes onIssueChange
// when a new issue is detected. It runs until ctx is cancelled.
func watchFlag(ctx context.Context, continent string, interval time.Duration, onIssueChange func(issue, title string)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastIssue string
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			flag := ReadFlag(continent)
			if flag.CurrentIssue != "" && flag.CurrentIssue != lastIssue {
				lastIssue = flag.CurrentIssue
				onIssueChange(flag.CurrentIssue, flag.CurrentTitle)
			}
		}
	}
}
