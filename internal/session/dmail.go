package session

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hironow/paintress"
)

// SendDMail writes a d-mail via the transactional outbox (Stage → Flush to
// archive/ + outbox/). Archive-first ordering is guaranteed by the OutboxStore.
func SendDMail(store paintress.OutboxStore, d paintress.DMail, eventStore paintress.EventStore) error {
	if d.SchemaVersion == "" {
		d.SchemaVersion = paintress.DMailSchemaVersion
	}
	data, err := d.Marshal()
	if err != nil {
		return fmt.Errorf("dmail: marshal: %w", err)
	}

	filename := d.Name + ".md"
	if err := store.Stage(filename, data); err != nil {
		return fmt.Errorf("dmail: stage: %w", err)
	}
	emitDMailEvent(eventStore, paintress.EventDMailStaged, paintress.DMailStagedData{Name: d.Name})
	n, err := store.Flush()
	if err != nil {
		return fmt.Errorf("dmail: flush: %w", err)
	}
	emitDMailEvent(eventStore, paintress.EventDMailFlushed, paintress.DMailFlushedData{Count: n})
	return nil
}

// ScanInbox reads all .md files in inbox/, parses each as DMail.
// Returns parsed d-mails sorted by filename. Returns empty slice for empty
// or non-existent directory.
func ScanInbox(continent string) ([]paintress.DMail, error) {
	dir := paintress.InboxDir(continent)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []paintress.DMail{}, nil
		}
		return nil, fmt.Errorf("dmail: read inbox: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var dmails []paintress.DMail
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("dmail: read %s: %w", e.Name(), err)
		}
		dm, err := paintress.ParseDMail(data)
		if err != nil {
			return nil, fmt.Errorf("dmail: parse %s: %w", e.Name(), err)
		}
		dmails = append(dmails, dm)
	}

	if dmails == nil {
		return []paintress.DMail{}, nil
	}
	return dmails, nil
}

// ArchiveInboxDMail moves a d-mail from inbox/ to archive/.
// Uses os.Rename for atomic move.
func ArchiveInboxDMail(continent, name string, eventStore paintress.EventStore) error {
	filename := name + ".md"
	src := filepath.Join(paintress.InboxDir(continent), filename)
	arcDir := paintress.ArchiveDir(continent)
	dst := filepath.Join(arcDir, filename)

	if err := os.MkdirAll(arcDir, 0755); err != nil {
		return fmt.Errorf("dmail: mkdir archive: %w", err)
	}

	if err := os.Rename(src, dst); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if _, statErr := os.Stat(dst); statErr == nil {
				return nil // already archived by another worker
			} else if errors.Is(statErr, fs.ErrNotExist) {
				return fmt.Errorf("dmail: archive %s: source not found and not in archive", name)
			} else {
				return fmt.Errorf("dmail: archive %s: stat archive dst: %w", name, statErr)
			}
		}
		return fmt.Errorf("dmail: archive %s: %w", name, err)
	}

	emitDMailEvent(eventStore, paintress.EventDMailArchived, paintress.DMailArchivedData{Name: name})
	return nil
}

func emitDMailEvent(store paintress.EventStore, eventType paintress.EventType, data any) {
	if store == nil {
		return
	}
	ev, err := paintress.NewEvent(eventType, data, time.Now())
	if err != nil {
		return
	}
	store.Append(ev)
}
