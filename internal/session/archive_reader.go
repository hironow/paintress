package session

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// archiveReaderAdapter reads D-Mails from the archive directory.
type archiveReaderAdapter struct {
	archiveDir string
}

// NewArchiveReader creates an ArchiveReader for the given archive directory.
func NewArchiveReader(archiveDir string) port.ArchiveReader {
	return &archiveReaderAdapter{archiveDir: archiveDir}
}

func (a *archiveReaderAdapter) ReadArchiveDMails(_ context.Context) ([]domain.DMail, error) {
	entries, err := os.ReadDir(a.archiveDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names) // chronological by naming convention

	var dmails []domain.DMail
	for _, name := range names {
		data, readErr := os.ReadFile(filepath.Join(a.archiveDir, name))
		if readErr != nil {
			continue // skip unreadable files
		}
		dm, parseErr := domain.ParseDMail(data)
		if parseErr != nil {
			continue // skip unparseable files
		}
		dmails = append(dmails, dm)
	}
	return dmails, nil
}
