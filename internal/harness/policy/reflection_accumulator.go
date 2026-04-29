package policy

import (
	"fmt"
	"strings"
)

// ReflectionCycle holds review data from one review-fix cycle.
type ReflectionCycle struct { // nosemgrep: structure.multiple-exported-structs-go -- reflection family (ReflectionCycle/ReflectionAccumulator) are cohesive types for the review-fix cycle stagnation detector; splitting would fragment the accumulator API [permanent]
	Number   int
	Comments string
	TagCount int
}

// ReflectionAccumulator accumulates review comment history across review-fix cycles
// to enable stagnation detection and reflection-enriched fix prompts.
type ReflectionAccumulator struct {
	cycles []ReflectionCycle
}

// NewReflectionAccumulator creates an empty ReflectionAccumulator.
func NewReflectionAccumulator() *ReflectionAccumulator {
	return &ReflectionAccumulator{}
}

// AddCycle records a review cycle's comments and automatically counts priority tags.
func (a *ReflectionAccumulator) AddCycle(cycleNumber int, comments string) {
	a.cycles = append(a.cycles, ReflectionCycle{
		Number:   cycleNumber,
		Comments: comments,
		TagCount: CountPriorityTags(comments),
	})
}

// IsStagnant reports whether the last two recorded cycles show no improvement.
// Returns false when fewer than two cycles have been recorded (no baseline).
func (a *ReflectionAccumulator) IsStagnant() bool {
	if len(a.cycles) < 2 {
		return false
	}
	last := a.cycles[len(a.cycles)-1]
	prev := a.cycles[len(a.cycles)-2]
	return IsStagnant(last.TagCount, prev.TagCount)
}

// FormatForPrompt renders accumulated cycle history as a prompt section.
// Returns an empty string when no cycles have been recorded.
func (a *ReflectionAccumulator) FormatForPrompt() string {
	if len(a.cycles) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Previous review cycles\n\n")
	for _, c := range a.cycles {
		sb.WriteString(fmt.Sprintf("### Cycle %d (%d tag(s))\n\n", c.Number, c.TagCount))
		sb.WriteString(c.Comments)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

