package session
// white-box-reason: SQLite internals: tests unexported db field for PRAGMA verification

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestSQLiteOutboxStore_PragmaSynchronousNormal(t *testing.T) {
	// given
	continent := t.TempDir()
	for _, dir := range []string{
		domain.ArchiveDir(continent),
		domain.OutboxDir(continent),
		domain.InboxDir(continent),
		filepath.Join(continent, ".expedition", ".run"),
	} {
		os.MkdirAll(dir, 0o755)
	}
	store, err := NewOutboxStoreForDir(continent)
	if err != nil {
		t.Fatalf("create outbox store: %v", err)
	}
	defer store.Close()

	// when: query PRAGMA on the store's own connection (package-internal access)
	var synchronous string
	if err := store.db.QueryRow("PRAGMA synchronous").Scan(&synchronous); err != nil {
		t.Fatalf("query PRAGMA synchronous: %v", err)
	}

	// then: synchronous = 1 (NORMAL)
	if synchronous != "1" {
		t.Errorf("PRAGMA synchronous: got %q, want %q (NORMAL)", synchronous, "1")
	}
}
