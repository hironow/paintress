package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hironow/paintress/internal/domain"
)

// realGetInsights exposes the learning loop to the session (refs issue
// 0034 P4): persisted insight-ledger files (.expedition/insights/*.md)
// plus a live Lumina recomputation from the journals. The live scan
// revives the dormant ScanJournalsForLumina path — recomputed per call
// from the journal read models, so it is always fresh, write-free and
// idempotent. Missing files / empty journals are an empty result, not
// an error.
func realGetInsights(continent string, args json.RawMessage) map[string]any {
	var payload struct {
		Kind string `json:"kind"`
	}
	if len(args) > 0 {
		_ = json.Unmarshal(args, &payload)
	}
	if continent == "" {
		return jsonResult(map[string]any{
			"initialized": false,
			"reason":      "paintress mcp continent not configured (start `paintress mcp` from the project root)",
		})
	}

	insightsDir := filepath.Join(continent, domain.StateDir, "insights")
	runDir := filepath.Join(continent, domain.StateDir, ".run")
	writer := NewInsightWriter(insightsDir, runDir)

	files := []map[string]any{}
	if entries, err := os.ReadDir(insightsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			if payload.Kind != "" && !strings.HasPrefix(e.Name(), payload.Kind) {
				continue
			}
			file, readErr := writer.Read(e.Name())
			if readErr != nil {
				continue
			}
			entryMaps := make([]map[string]any, 0, len(file.Entries))
			for _, ie := range file.Entries {
				entryMaps = append(entryMaps, map[string]any{
					"title":       ie.Title,
					"what":        ie.What,
					"why":         ie.Why,
					"how":         ie.How,
					"when":        ie.When,
					"who":         ie.Who,
					"constraints": ie.Constraints,
					"extra":       ie.Extra,
				})
			}
			files = append(files, map[string]any{
				"file":       e.Name(),
				"kind":       file.Kind,
				"updated_at": file.UpdatedAt,
				"entries":    entryMaps,
			})
		}
	}

	luminas := ScanJournalsForLumina(continent)
	liveLumina := make([]map[string]any, 0, len(luminas))
	for _, l := range luminas {
		liveLumina = append(liveLumina, map[string]any{
			"pattern": l.Pattern,
			"source":  l.Source,
			"uses":    l.Uses,
		})
	}

	return jsonResult(map[string]any{
		"initialized": true,
		"continent":   continent,
		"insights":    files,
		"live_lumina": liveLumina,
		"instruction": fmt.Sprintf("Review defensive patterns (failure-pattern / high-severity-alert) before implementing; offensive patterns (success-pattern) are proven approaches. %d persisted file(s), %d live lumina(s).", len(files), len(liveLumina)),
	})
}
