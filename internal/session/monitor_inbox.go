package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/hironow/paintress/internal/domain"
)

// MonitorInbox starts monitoring the inbox directory for D-Mail files.
// Two-phase: 1) synchronous drain of existing files, 2) async fsnotify watch.
// Returns a channel that delivers parsed D-Mails. Channel closes on ctx cancel.
func MonitorInbox(ctx context.Context, continent string, logger domain.Logger) (<-chan domain.DMail, error) {
	inboxDir := domain.InboxDir(continent)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := watcher.Add(inboxDir); err != nil {
		watcher.Close()
		return nil, err
	}

	// Phase 1: synchronous drain of existing files.
	entries, err := os.ReadDir(inboxDir)
	if err != nil {
		watcher.Close()
		return nil, err
	}

	var initial []domain.DMail
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(inboxDir, e.Name()))
		if readErr != nil {
			logger.Warn("inbox drain: skip %s: %v", e.Name(), readErr)
			continue
		}
		dm, parseErr := domain.ParseDMail(data)
		if parseErr != nil {
			logger.Warn("inbox drain: skip %s: %v", e.Name(), parseErr)
			continue
		}
		logger.Info("inbox drain: received %s", dm.Name)
		initial = append(initial, dm)
	}

	ch := make(chan domain.DMail, len(initial)+8)
	for _, dm := range initial {
		ch <- dm
	}

	// Phase 2: async fsnotify watch goroutine.
	go func() {
		defer close(ch)
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
					continue
				}
				if !strings.HasSuffix(event.Name, ".md") {
					continue
				}
				data, readErr := os.ReadFile(event.Name)
				if readErr != nil {
					logger.Warn("inbox watch: skip %s: %v", filepath.Base(event.Name), readErr)
					continue
				}
				dm, parseErr := domain.ParseDMail(data)
				if parseErr != nil {
					logger.Warn("inbox watch: skip %s: %v", filepath.Base(event.Name), parseErr)
					continue
				}
				logger.Info("inbox watch: received %s", dm.Name)
				select {
				case ch <- dm:
				case <-ctx.Done():
					return
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return ch, nil
}
