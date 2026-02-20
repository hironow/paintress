package paintress

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// watchFlag watches flag.md for current_issue changes using filesystem
// notifications and invokes onIssueChange when a new issue is detected.
// Performs an initial read to detect pre-existing flags, then watches
// the parent directory (.expedition/.run/) for write events.
// Returns silently if the directory does not exist.
//
// If ready is non-nil, a value is sent after the watcher is fully set up,
// allowing callers to synchronize without time.Sleep.
func watchFlag(ctx context.Context, continent string, onIssueChange func(issue, title string), ready chan<- struct{}) {
	runDir := filepath.Join(continent, ".expedition", ".run")

	if _, err := os.Stat(runDir); err != nil {
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	if err := watcher.Add(runDir); err != nil {
		return
	}

	// Signal that the watcher is fully set up
	if ready != nil {
		ready <- struct{}{}
	}

	// Initial read: detect pre-existing flag
	var lastIssue string
	flag := ReadFlag(continent)
	if flag.CurrentIssue != "" {
		lastIssue = flag.CurrentIssue
		onIssueChange(flag.CurrentIssue, flag.CurrentTitle)
	}

	flagName := filepath.Base(FlagPath(continent))
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if filepath.Base(event.Name) != flagName {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			flag := ReadFlag(continent)
			if flag.CurrentIssue != "" && flag.CurrentIssue != lastIssue {
				lastIssue = flag.CurrentIssue
				onIssueChange(flag.CurrentIssue, flag.CurrentTitle)
			}
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}
