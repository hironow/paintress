package session

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hironow/paintress"
)

// === Lumina: Parallel Journal Scan ===

func TestRace_Lumina_ConcurrentScan_ManyJournals(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	// Create 50 journals with mixed statuses — exercises goroutine pool + mutex
	for i := 1; i <= 50; i++ {
		var content string
		if i%3 == 0 {
			content = fmt.Sprintf(`# Expedition #%d

- **Status**: failed
- **Reason**: timeout error
- **Mission**: implement
`, i)
		} else {
			content = fmt.Sprintf(`# Expedition #%d

- **Status**: success
- **Mission**: implement
`, i)
		}
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", i)), []byte(content), 0644)
	}

	// Run scan — internally spawns goroutines per file
	luminas := ScanJournalsForLumina(dir)

	// Should find patterns given enough repetitions
	hasFailure := false
	hasSuccess := false
	for _, l := range luminas {
		if containsStr(l.Pattern, "timeout error") {
			hasFailure = true
		}
		if containsStr(l.Pattern, "implement") && containsStr(l.Pattern, "Proven approach") {
			hasSuccess = true
		}
	}
	if !hasFailure {
		t.Error("expected failure lumina from 16+ 'timeout error' failures")
	}
	if !hasSuccess {
		t.Error("expected success lumina from 34+ 'implement' successes")
	}
}

func TestRace_Lumina_ConcurrentScan(t *testing.T) {
	dir := t.TempDir()
	jDir := filepath.Join(dir, ".expedition", "journal")
	os.MkdirAll(jDir, 0755)

	for i := 1; i <= 5; i++ {
		content := `# Expedition

- **Status**: failed
- **Reason**: compile error
- **Mission**: fix
`
		os.WriteFile(filepath.Join(jDir, fmt.Sprintf("%03d.md", i)), []byte(content), 0644)
	}

	// Multiple goroutines call ScanJournalsForLumina concurrently on same dir
	var wg sync.WaitGroup
	results := make([][]paintress.Lumina, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = ScanJournalsForLumina(dir)
		}(i)
	}
	wg.Wait()

	// All results should be consistent (same journals -> same luminas)
	for i, r := range results {
		if len(r) == 0 {
			t.Errorf("goroutine %d returned 0 luminas", i)
		}
	}
}

// === DevServer: Start/Stop Race ===

func TestRace_DevServer_ConcurrentStopCalls(t *testing.T) {
	ds := NewDevServer("echo hello", "http://localhost:19999", t.TempDir(), filepath.Join(t.TempDir(), "dev.log"), paintress.NewLogger(io.Discard, false))

	// Multiple concurrent Stop calls should not panic
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ds.Stop()
		}()
	}
	wg.Wait()
}

func TestRace_DevServer_ConcurrentFieldAccess(t *testing.T) {
	ds := NewDevServer("echo hello", "http://localhost:19999", t.TempDir(), filepath.Join(t.TempDir(), "dev.log"), paintress.NewLogger(io.Discard, false))

	var wg sync.WaitGroup

	// Concurrent reads and Stop calls — exercises the mutex
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			ds.Stop()
		}()
		go func() {
			defer wg.Done()
			// Access fields that are protected by mutex
			ds.mu.Lock()
			_ = ds.running
			ds.mu.Unlock()
		}()
	}
	wg.Wait()
}

// === Expedition: Streaming + Reserve integration ===

func TestRace_Expedition_ConcurrentReserveCheck(t *testing.T) {
	dir := t.TempDir()
	rp := paintress.NewReserveParty("opus", []string{"sonnet"}, paintress.NewLogger(io.Discard, false))
	g := paintress.NewGradientGauge(5)

	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    paintress.Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
		Logger:    paintress.NewLogger(io.Discard, false),
		Gradient:  g,
		Reserve:   rp,
	}

	var wg sync.WaitGroup

	// Simulate concurrent access: BuildPrompt + Reserve ops + Gradient ops
	for i := 0; i < 20; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			_ = e.BuildPrompt()
		}()
		go func() {
			defer wg.Done()
			rp.CheckOutput("normal output")
		}()
		go func() {
			defer wg.Done()
			_ = rp.ActiveModel()
		}()
		go func() {
			defer wg.Done()
			g.Charge()
		}()
	}
	wg.Wait()
}
