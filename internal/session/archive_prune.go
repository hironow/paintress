package session

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// ArchivePrune scans .expedition/archive/ for .md files older than the given
// number of days. When execute is false (dry-run), it only lists candidates.
// When execute is true, it deletes them and reports how many were removed.
func ArchivePrune(continent string, days int, execute bool) (domain.PruneResult, error) {
	if days <= 0 {
		return domain.PruneResult{}, fmt.Errorf("days must be positive, got %d", days)
	}
	dir := domain.ArchiveDir(continent)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return domain.PruneResult{}, nil
		}
		return domain.PruneResult{}, err
	}

	threshold := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	var result domain.PruneResult

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
