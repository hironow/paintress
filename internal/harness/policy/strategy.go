package policy

// FixStrategy identifies the approach for a review-fix cycle.
type FixStrategy string

const (
	// StrategyDirect applies review comments directly without additional guidance.
	StrategyDirect FixStrategy = "direct"
	// StrategyDecompose breaks review comments into smaller sub-tasks before fixing.
	StrategyDecompose FixStrategy = "decompose"
	// StrategyRewrite rewrites the affected section from scratch to resolve comments.
	StrategyRewrite FixStrategy = "rewrite"
)

// strategies is the ordered rotation of fix strategies.
var strategies = []FixStrategy{
	StrategyDirect,
	StrategyDecompose,
	StrategyRewrite,
}

// StrategyForCycle returns the fix strategy for the given cycle number.
// Cycles rotate through Direct → Decompose → Rewrite and repeat.
// Cycle numbering starts at 1.
func StrategyForCycle(cycle int) FixStrategy {
	if cycle < 1 {
		cycle = 1
	}
	idx := (cycle - 1) % len(strategies)
	return strategies[idx]
}

