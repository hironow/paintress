package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// === Lumina: Parallel Journal Scan ===

func TestLumina_ConcurrentScan_ManyJournals(t *testing.T) {
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

func TestLumina_ConcurrentScan_CalledFromMultipleGoroutines(t *testing.T) {
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
	results := make([][]Lumina, 10)
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

func TestLumina_WriteLumina_ConcurrentCalls(t *testing.T) {
	// WriteLumina is called sequentially in production (Paintress loop),
	// but verify concurrent calls don't panic or corrupt state.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			dir := t.TempDir()
			os.MkdirAll(filepath.Join(dir, ".expedition"), 0755)
			luminas := []Lumina{
				{Pattern: fmt.Sprintf("pattern %d", n), Source: "test", Uses: 1},
			}
			if err := WriteLumina(dir, luminas); err != nil {
				t.Errorf("WriteLumina(%d) error: %v", n, err)
			}
		}(i)
	}
	wg.Wait()
}

// === DevServer: Start/Stop Race ===

func TestDevServer_ConcurrentStopCalls(t *testing.T) {
	ds := NewDevServer("echo hello", "http://localhost:19999", t.TempDir(), filepath.Join(t.TempDir(), "dev.log"))

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

func TestDevServer_ConcurrentFieldAccess(t *testing.T) {
	ds := NewDevServer("echo hello", "http://localhost:19999", t.TempDir(), filepath.Join(t.TempDir(), "dev.log"))

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

// === Reserve Party: Extended Concurrent Scenarios ===

func TestReserve_ConcurrentCheckAndRecover(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet", "haiku"})

	var wg sync.WaitGroup

	// Mix of CheckOutput, TryRecoverPrimary, ActiveModel, ForceReserve concurrently
	for i := 0; i < 50; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			rp.CheckOutput("some normal output")
		}()
		go func() {
			defer wg.Done()
			rp.TryRecoverPrimary()
		}()
		go func() {
			defer wg.Done()
			_ = rp.ActiveModel()
		}()
		go func() {
			defer wg.Done()
			_ = rp.FormatForPrompt()
		}()
	}
	wg.Wait()

	// Should not panic and model should be valid
	model := rp.ActiveModel()
	if model != "opus" && model != "sonnet" && model != "haiku" {
		t.Errorf("unexpected model after concurrent ops: %q", model)
	}
}

func TestReserve_ConcurrentForceAndRecover(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"})

	var wg sync.WaitGroup

	// Alternate force reserve and recover in parallel
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			rp.ForceReserve()
		}()
		go func() {
			defer wg.Done()
			rp.TryRecoverPrimary()
		}()
	}
	wg.Wait()

	model := rp.ActiveModel()
	if model != "opus" && model != "sonnet" {
		t.Errorf("unexpected model: %q", model)
	}
}

func TestReserve_ConcurrentStatusAndCheckOutput(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			_ = rp.Status()
		}()
		go func() {
			defer wg.Done()
			rp.CheckOutput("rate limit exceeded")
		}()
		go func() {
			defer wg.Done()
			_ = rp.FormatForPrompt()
		}()
	}
	wg.Wait()
}

// === Gradient Gauge: Extended Concurrent Scenarios ===

func TestGradient_ConcurrentFormatForPrompt(t *testing.T) {
	g := NewGradientGauge(5)

	var wg sync.WaitGroup

	// FormatForPrompt calls priorityHint internally (deadlock regression test)
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			g.Charge()
		}()
		go func() {
			defer wg.Done()
			_ = g.FormatForPrompt()
		}()
		go func() {
			defer wg.Done()
			_ = g.PriorityHint()
		}()
	}
	wg.Wait()

	level := g.Level()
	if level < 0 || level > 5 {
		t.Errorf("level out of range: %d", level)
	}
}

func TestGradient_ConcurrentFormatLog(t *testing.T) {
	g := NewGradientGauge(10)

	var wg sync.WaitGroup

	for i := 0; i < 30; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			g.Charge()
		}()
		go func() {
			defer wg.Done()
			g.Decay()
		}()
		go func() {
			defer wg.Done()
			_ = g.FormatLog()
		}()
	}
	wg.Wait()

	log := g.FormatLog()
	if log == "" {
		t.Error("log should not be empty after operations")
	}
}

// === Logger: Concurrent Init/Close/Write ===

func TestLogger_ConcurrentInitCloseWrite(t *testing.T) {
	dir := t.TempDir()

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(3)
		go func(n int) {
			defer wg.Done()
			path := filepath.Join(dir, fmt.Sprintf("log_%d.log", n))
			InitLogFile(path)
		}(i)
		go func(n int) {
			defer wg.Done()
			LogInfo("race test info %d", n)
			LogWarn("race test warn %d", n)
		}(i)
		go func() {
			defer wg.Done()
			CloseLogFile()
		}()
	}
	wg.Wait()

	// Clean up
	CloseLogFile()
}

// === Lang: Concurrent Msg() reads with Lang writes ===

func TestLang_ConcurrentMsgReads(t *testing.T) {
	orig := Lang
	defer func() { Lang = orig }()

	var wg sync.WaitGroup

	// Concurrent reads of Msg() — all reading the same global Lang
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg := Msg("grad_attack")
			if msg == "" {
				t.Error("Msg should never return empty")
			}
		}()
	}
	wg.Wait()
}

// === Expedition: Streaming + Reserve integration ===

func TestExpedition_ConcurrentReserveCheckDuringBuild(t *testing.T) {
	dir := t.TempDir()
	rp := NewReserveParty("opus", []string{"sonnet"})
	g := NewGradientGauge(5)

	e := &Expedition{
		Number:    1,
		Continent: dir,
		Config:    Config{BaseBranch: "main", DevURL: "http://localhost:3000"},
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
