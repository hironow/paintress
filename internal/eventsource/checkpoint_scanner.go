package eventsource

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hironow/paintress/internal/domain"
)

// CheckpointScanner scans JSONL event files for incomplete expedition checkpoints.
type CheckpointScanner struct {
	eventsDir string
}

// NewCheckpointScanner creates a scanner for the given events directory.
func NewCheckpointScanner(continent string) *CheckpointScanner {
	return &CheckpointScanner{eventsDir: domain.EventsDir(continent)}
}

// FindIncompleteCheckpoints returns checkpoint data for expeditions that have
// a checkpoint event but no subsequent expedition.completed event.
func (s *CheckpointScanner) FindIncompleteCheckpoints() []domain.ExpeditionCheckpointData {
	files, err := os.ReadDir(s.eventsDir)
	if err != nil {
		return nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	checkpoints := make(map[int]domain.ExpeditionCheckpointData)
	completed := make(map[int]bool)

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".jsonl") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(s.eventsDir, f.Name()))
		if readErr != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var ev domain.Event
			if jsonErr := json.Unmarshal([]byte(line), &ev); jsonErr != nil {
				continue
			}
			switch ev.Type {
			case domain.EventExpeditionCheckpoint:
				var cpData domain.ExpeditionCheckpointData
				if jsonErr := json.Unmarshal(ev.Data, &cpData); jsonErr == nil {
					checkpoints[cpData.Expedition] = cpData
				}
			case domain.EventExpeditionCompleted:
				var compData domain.ExpeditionCompletedData
				if jsonErr := json.Unmarshal(ev.Data, &compData); jsonErr == nil {
					completed[compData.Expedition] = true
				}
			}
		}
	}

	var result []domain.ExpeditionCheckpointData
	for expNum, cp := range checkpoints {
		if !completed[expNum] {
			result = append(result, cp)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Expedition < result[j].Expedition
	})

	return result
}
