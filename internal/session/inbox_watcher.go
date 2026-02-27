package session

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/hironow/paintress"
)

// watchInbox watches the inbox/ directory for new or updated D-Mail files.
func watchInbox(ctx context.Context, continent string, onNewDMail func(dm paintress.DMail), ready chan<- struct{}) {
	inboxDir := paintress.InboxDir(continent)

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
	entries, _ := os.ReadDir(inboxDir)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(inboxDir, entry.Name()))
		if err != nil {
			continue
		}
		dm, err := paintress.ParseDMail(data)
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
			dm, err := paintress.ParseDMail(data)
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
