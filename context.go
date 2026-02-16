package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContextDir returns the path to the context injection directory.
func ContextDir(continent string) string {
	return filepath.Join(continent, ".expedition", "context")
}

// ReadContextFiles reads all .md files from .expedition/context/ and
// concatenates them into a single string for prompt injection.
// Returns empty string if the directory does not exist.
func ReadContextFiles(continent string) string {
	dir := ContextDir(continent)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var buf strings.Builder
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		buf.WriteString(fmt.Sprintf("### %s\n\n", name))
		buf.Write(content)
		buf.WriteString("\n\n")
	}
	return buf.String()
}
