package paintress

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PruneResult holds the outcome of an archive prune operation.
type PruneResult struct {
	Candidates []string // basenames of files older than threshold
	Deleted    int      // number of files actually removed (0 in dry-run)
}

// ArchivePrune scans .expedition/archive/ for .md files older than the given
// number of days. When execute is false (dry-run), it only lists candidates.
// When execute is true, it deletes them and reports how many were removed.
func ArchivePrune(continent string, days int, execute bool) (PruneResult, error) {
	if days <= 0 {
		return PruneResult{}, fmt.Errorf("days must be positive, got %d", days)
	}
	dir := ArchiveDir(continent)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return PruneResult{}, nil
		}
		return PruneResult{}, err
	}

	threshold := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	var result PruneResult

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(threshold) {
			result.Candidates = append(result.Candidates, e.Name())
			if execute {
				if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
					result.Deleted++
				}
			}
		}
	}

	return result, nil
}
