package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// WriteClaudeLog persists raw stream-json events to .run/claude-logs/.
// Each invocation creates a new timestamped .jsonl file.  nosemgrep: layer-session-no-event-persistence [permanent]
func WriteClaudeLog(continent string, rawEvents []string) error {
	if len(rawEvents) == 0 {
		return nil
	}

	logDir := filepath.Join(domain.RunDir(continent), "claude-logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("mkdir claude-logs: %w", err)
	}

	// nosemgrep: layer-session-no-event-persistence — log file, not event store [permanent]
	filename := fmt.Sprintf("%s.jsonl", time.Now().UTC().Format("20060102-150405"))
	path := filepath.Join(logDir, filename)

	var buf strings.Builder
	for _, event := range rawEvents {
		buf.WriteString(event)
		buf.WriteByte('\n')
	}

	if err := os.WriteFile(path, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write claude-log: %w", err)
	}
	return nil
}
