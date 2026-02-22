package paintress

import (
	"context"
	"os"
	"path/filepath"
	"time"

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

	// Track last-processed mod time per file to deduplicate: fsnotify may
	// fire CREATE + WRITE for the same atomic write (same mod time), but
	// legitimate updates (different mod time) must still be delivered.
	lastMod := make(map[string]time.Time)

	// Initial scan: catch files that already exist before the event loop starts.
	entries, _ := os.ReadDir(inboxDir)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		fullPath := filepath.Join(inboxDir, entry.Name())
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		dm, err := ParseDMail(data)
		if err != nil {
			continue
		}
		lastMod[fullPath] = info.ModTime()
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
			info, err := os.Stat(event.Name)
			if err != nil {
				continue
			}
			if prev, ok := lastMod[event.Name]; ok && !info.ModTime().After(prev) {
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
			lastMod[event.Name] = info.ModTime()
			onNewDMail(dm)
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}
