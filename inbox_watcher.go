package paintress

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// watchInbox watches the inbox/ directory for new D-Mail files using filesystem
// notifications and invokes onNewDMail when a new .md file is detected.
// Returns silently if the inbox directory does not exist.
//
// If ready is non-nil, a value is sent after the watcher is fully set up,
// allowing callers to synchronize without time.Sleep.
func watchInbox(ctx context.Context, continent string, onNewDMail func(dm DMail), ready chan<- struct{}) {
	inboxDir := InboxDir(continent)

	if _, err := os.Stat(inboxDir); err != nil {
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	if err := watcher.Add(inboxDir); err != nil {
		return
	}

	if ready != nil {
		ready <- struct{}{}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if filepath.Ext(event.Name) != ".md" {
				continue
			}
			if event.Op&fsnotify.Create == 0 {
				continue
			}
			data, err := os.ReadFile(event.Name)
			if err != nil {
				continue
			}
			dm, err := ParseDMail(data)
			if err != nil {
				continue
			}
			onNewDMail(dm)
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}
