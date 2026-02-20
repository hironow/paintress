package paintress

import (
	"fmt"
	"strings"
	"sync"
)

// GradientGauge tracks consecutive successes to scale difficulty.
// In the game, Gradient Charges build up through skillful play,
// enabling devastating Gradient Attacks.
// Here, consecutive successes raise the gauge, unlocking harder issues.
//
// Gauge levels:
//
//	0 (empty)  -- only pick low-priority / small issues
//	1-2        -- normal priority issues
//	3-4        -- high priority issues
//	5+ (full)  -- Gradient Attack: attempt the highest priority / most complex issue
//
// A failure resets the gauge to 0 (like getting hit resets your combo).
// A skip decrements by 1 (partial damage).
type GradientGauge struct {
	mu    sync.RWMutex
	level int
	max   int
	log   []string // history of charges/discharges
}

func NewGradientGauge(max int) *GradientGauge {
	return &GradientGauge{
		max: max,
	}
}

// Charge increments the gauge on success.
func (g *GradientGauge) Charge() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.level < g.max {
		g.level++
	}
	g.log = append(g.log, fmt.Sprintf("+1 (now %d/%d)", g.level, g.max))
}

// Discharge resets the gauge on failure.
func (g *GradientGauge) Discharge() {
	g.mu.Lock()
	defer g.mu.Unlock()
	prev := g.level
	g.level = 0
	g.log = append(g.log, fmt.Sprintf("RESET %d->0", prev))
}

// Decay decrements by 1 on skip (partial setback).
func (g *GradientGauge) Decay() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.level > 0 {
		g.level--
	}
	g.log = append(g.log, fmt.Sprintf("-1 (now %d/%d)", g.level, g.max))
}

// Level returns the current gauge level.
func (g *GradientGauge) Level() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.level
}

// IsGradientAttack returns true if the gauge is full — time for a big challenge.
func (g *GradientGauge) IsGradientAttack() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.level >= g.max
}

// PriorityHint returns a priority guidance string for the expedition prompt.
// This tells the Expeditioner which difficulty of issue to pick.
func (g *GradientGauge) PriorityHint() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.priorityHint()
}

// priorityHint is the lock-free internal version. Caller must hold at least RLock.
func (g *GradientGauge) priorityHint() string {
	switch {
	case g.level >= g.max:
		return Msg("grad_attack")
	case g.level >= 3:
		return Msg("grad_high")
	case g.level >= 1:
		return Msg("grad_normal")
	default:
		return Msg("grad_empty")
	}
}

// FormatForPrompt returns the gauge state for prompt injection.
func (g *GradientGauge) FormatForPrompt() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	bar := strings.Repeat("█", g.level) + strings.Repeat("░", g.max-g.level)
	return fmt.Sprintf("Gradient Gauge: [%s] %d/%d\n%s", bar, g.level, g.max, g.priorityHint())
}

// FormatLog returns the gauge history.
func (g *GradientGauge) FormatLog() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if len(g.log) == 0 {
		return "(empty)"
	}
	return strings.Join(g.log, "\n")
}
