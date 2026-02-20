package paintress

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// watchInbox watches the inbox/ directory for new or updated D-Mail files using
// filesystem notifications and invokes onNewDMail when a valid .md file is detected.
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

	// Initial scan: catch files that already exist before the event loop starts.
	// The watcher is already registered, so any file created during this scan
	// will also produce an event — duplicates are handled by the caller's dedup.
	entries, _ := os.ReadDir(inboxDir)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(inboxDir, entry.Name()))
		if err != nil {
			continue
		}
		dm, err := ParseDMail(data)
		if err != nil {
			continue
		}
		onNewDMail(dm)
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
			if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
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
