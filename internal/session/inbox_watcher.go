package session

import (
	"context"
	"os"
	"path/filepath"

	"go.opentelemetry.io/otel/attribute"

	"github.com/fsnotify/fsnotify"
	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

// watchInbox watches the inbox/ directory for new or updated D-Mail files.
func watchInbox(ctx context.Context, continent string, onNewDMail func(dm domain.DMail), ready chan<- struct{}) {
	_, span := platform.Tracer.Start(ctx, "paintress.inbox")
	defer span.End()

	inboxDir := domain.InboxDir(continent)

	if _, err := os.Stat(inboxDir); err != nil {
		span.SetAttributes(attribute.Int("inbox.watch.event.count", 0))
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.stage", "paintress.inbox"))
		span.SetAttributes(attribute.Int("inbox.watch.event.count", 0))
		return
	}
	defer watcher.Close()

	if err := watcher.Add(inboxDir); err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.stage", "paintress.inbox"))
		span.SetAttributes(attribute.Int("inbox.watch.event.count", 0))
		return
	}

	eventCount := 0

	// Initial scan: catch files that already exist before the event loop starts.
	entries, err := os.ReadDir(inboxDir)
	if err != nil {
		span.RecordError(err)
		entries = nil
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(inboxDir, entry.Name()))
		if err != nil {
			continue
		}
		dm, err := domain.ParseDMail(data)
		if err != nil {
			continue
		}
		onNewDMail(dm)
		eventCount++
	}

	if ready != nil {
		ready <- struct{}{}
	}

	for {
		select {
		case <-ctx.Done():
			span.SetAttributes(attribute.Int("inbox.watch.event.count", eventCount))
			return
		case event, ok := <-watcher.Events:
			if !ok {
				span.SetAttributes(attribute.Int("inbox.watch.event.count", eventCount))
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
			dm, err := domain.ParseDMail(data)
			if err != nil {
				continue
			}
			onNewDMail(dm)
			eventCount++
		case _, ok := <-watcher.Errors:
			if !ok {
				span.SetAttributes(attribute.Int("inbox.watch.event.count", eventCount))
				return
			}
		}
	}
}
