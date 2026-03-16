package session

import (
	"fmt"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
)

func (p *Paintress) printBanner() {
	domain.LogSection(p.Logger, "paintress run")
}

// reconcileFlags scans the continent's own flag.md and, when workers > 0,
// all worktree flag.md files, returning the one with the highest LastExpedition.
func reconcileFlags(continent string, workers int) domain.ExpeditionFlag {
	flags := []domain.ExpeditionFlag{ReadFlag(continent)}
	if workers == 0 {
		return flags[0]
	}
	pattern := filepath.Join(continent, domain.StateDir, ".run", "worktrees", "*",
		domain.StateDir, ".run", "flag.md")
	matches, _ := filepath.Glob(pattern)
	for _, match := range matches {
		base := filepath.Dir(filepath.Dir(filepath.Dir(match)))
		flags = append(flags, ReadFlag(base))
	}
	return domain.BestFlag(flags)
}

func (p *Paintress) writeFlag(dir string, expNum int, issueID, status, remaining string, midHighSeverity int) {
	current := ReadFlag(dir)
	if expNum <= current.LastExpedition {
		return
	}
	WriteFlag(dir, expNum, issueID, status, remaining, midHighSeverity)
}

// HandoverSummary returns the current expedition counts for handover documents.
func (p *Paintress) HandoverSummary() domain.RunSummary {
	return domain.RunSummary{
		Total:   p.totalAttempted.Load(),
		Success: p.totalSuccess.Load(),
		Skipped: p.totalSkipped.Load(),
		Failed:  p.totalFailed.Load(),
	}
}

func (p *Paintress) printSummary() {
	total := p.totalAttempted.Load()

	if p.config.OutputFormat == "json" {
		summary := domain.RunSummary{
			Total:           total,
			Success:         p.totalSuccess.Load(),
			Skipped:         p.totalSkipped.Load(),
			Failed:          p.totalFailed.Load(),
			Bugs:            p.totalBugs.Load(),
			MidHighSeverity: p.totalMidHighSeverity.Load(),
			Gradient:        p.gradient.FormatLog(),
		}
		out, err := domain.FormatSummaryJSON(summary)
		if err != nil {
			p.Logger.Error("json marshal: %v", err)
			return
		}
		fmt.Fprintln(p.DataOut, out)
		return
	}

	domain.LogSection(p.Logger, "paintress summary")
	p.Logger.Info("%s", fmt.Sprintf(domain.Msg("expeditions_sent"), total))
	p.Logger.OK("%s", fmt.Sprintf(domain.Msg("success_count"), p.totalSuccess.Load()))
	p.Logger.Warn("%s", fmt.Sprintf(domain.Msg("skipped_count"), p.totalSkipped.Load()))
	p.Logger.Error("%s", fmt.Sprintf(domain.Msg("failed_count"), p.totalFailed.Load()))
	if p.totalMidHighSeverity.Load() > 0 {
		p.Logger.Warn("Mid-expedition HIGH severity D-Mail: %d", p.totalMidHighSeverity.Load())
	}
	if p.totalBugs.Load() > 0 {
		p.Logger.Info("%s", fmt.Sprintf(domain.Msg("bugs_count"), p.totalBugs.Load()))
	}
	fmt.Fprintln(p.ErrOut)
	p.Logger.Info("%s", fmt.Sprintf(domain.Msg("gradient_info"), p.gradient.FormatLog()))
	p.Logger.Info("%s", fmt.Sprintf(domain.Msg("party_info"), p.reserve.Status()))
	fmt.Fprintln(p.ErrOut)
	p.Logger.Info("Flag:     %s", domain.FlagPath(p.config.Continent))
	p.Logger.Info("Journals: %s", domain.JournalDir(p.config.Continent))
	p.Logger.Info("Logs:     %s", p.logDir)
}
