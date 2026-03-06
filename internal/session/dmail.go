package session

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// SendDMail writes a d-mail via the transactional outbox (Stage → Flush to
// archive/ + outbox/). Archive-first ordering is guaranteed by the OutboxStore.
func SendDMail(ctx context.Context, store port.OutboxStore, d domain.DMail, emitter port.ExpeditionEventEmitter) error {
	if d.SchemaVersion == "" {
		d.SchemaVersion = domain.DMailSchemaVersion
	}
	if err := domain.ValidateDMail(d); err != nil {
		return fmt.Errorf("dmail: validate: %w", err)
	}
	data, err := d.Marshal()
	if err != nil {
		return fmt.Errorf("dmail: marshal: %w", err)
	}

	filename := d.Name + ".md"
	if err := store.Stage(ctx, filename, data); err != nil {
		return fmt.Errorf("dmail: stage: %w", err)
	}
	if emitter != nil {
		if emitErr := emitter.EmitDMailStaged(d.Name, time.Now()); emitErr != nil {
			return fmt.Errorf("dmail: event staged: %w", emitErr)
		}
	}
	n, err := store.Flush(ctx)
	if err != nil {
		return fmt.Errorf("dmail: flush: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("dmail: flush: item not delivered (write failure, will retry)")
	}
	if emitter != nil {
		if emitErr := emitter.EmitDMailFlushed(n, time.Now()); emitErr != nil {
			return fmt.Errorf("dmail: event flushed: %w", emitErr)
		}
	}
	return nil
}

// ScanInbox reads all .md files in inbox/, parses each as DMail.
// Returns parsed d-mails sorted by filename. Returns empty slice for empty
// or non-existent directory.
func ScanInbox(continent string) ([]domain.DMail, error) {
	dir := domain.InboxDir(continent)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []domain.DMail{}, nil
		}
		return nil, fmt.Errorf("dmail: read inbox: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var dmails []domain.DMail
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("dmail: read %s: %w", e.Name(), err)
		}
		dm, err := domain.ParseDMail(data)
		if err != nil {
			return nil, fmt.Errorf("dmail: parse %s: %w", e.Name(), err)
		}
		dmails = append(dmails, dm)
	}

	if dmails == nil {
		return []domain.DMail{}, nil
	}
	return dmails, nil
}

// ArchiveInboxDMail moves a d-mail from inbox/ to archive/.
// Uses os.Rename for atomic move.
func ArchiveInboxDMail(continent, name string, emitter port.ExpeditionEventEmitter) error {
	filename := name + ".md"
	src := filepath.Join(domain.InboxDir(continent), filename)
	arcDir := domain.ArchiveDir(continent)
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

	if emitter != nil {
		if emitErr := emitter.EmitDMailArchived(name, time.Now()); emitErr != nil {
			return fmt.Errorf("dmail: event archived: %w", emitErr)
		}
	}
	return nil
}

