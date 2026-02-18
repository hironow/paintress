package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ContextDir returns the path to the context injection directory.
func ContextDir(continent string) string {
	return filepath.Join(continent, ".expedition", "context")
}

// ReadContextFiles reads all .md files from .expedition/context/ and
// concatenates them into a single string for prompt injection.
// Returns ("", nil) if the directory does not exist.
// Returns a non-nil error for other filesystem failures (e.g. permission denied).
func ReadContextFiles(continent string) (string, error) {
	dir := ContextDir(continent)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("reading context directory: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var buf strings.Builder
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return "", fmt.Errorf("reading context file %s: %w", e.Name(), err)
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		buf.WriteString(fmt.Sprintf("### %s\n\n", name))
		buf.Write(content)
		buf.WriteString("\n\n")
	}
	return buf.String(), nil
}
