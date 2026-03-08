package session

// white-box-reason: session internals: tests unexported triagePreFlightDMails method

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// countingEmitter wraps NopExpeditionEventEmitter and counts EmitEscalated
// and EmitRetryAttempted calls for assertion in triage tests.
type countingEmitter struct {
	port.NopExpeditionEventEmitter
	escalatedCount     int
	retryAttemptedCount int
}

func (c *countingEmitter) EmitEscalated(_ string, _ []string, _ time.Time) error {
	c.escalatedCount++
	return nil
}

func (c *countingEmitter) EmitRetryAttempted(_ string, _ int, _ time.Time) error {
	c.retryAttemptedCount++
	return nil
}

func TestTriagePreFlightDMails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		dmails             []domain.DMail
		maxRetries         int
		preloadRetries     int // how many times to pre-call Track before triage
		preloadIssues      []string
		wantRemaining      int
		wantEscalated      int
		wantRetryAttempted int
	}{
		{
			name: "no action passes through",
			dmails: []domain.DMail{
				{Name: "info-1", Kind: "report", Description: "status update"},
			},
			maxRetries:    3,
			wantRemaining: 1,
		},
		{
			name:          "empty inbox returns zero",
			dmails:        []domain.DMail{},
			maxRetries:    3,
			wantRemaining: 0,
		},
		{
			name: "escalate removes dmail and emits event",
			dmails: []domain.DMail{
				{Name: "esc-1", Kind: "feedback", Description: "critical", Action: "escalate", Issues: []string{"MY-10"}},
			},
			maxRetries:    3,
			wantRemaining: 0,
			wantEscalated: 1,
		},
		{
			name: "resolve removes dmail",
			dmails: []domain.DMail{
				{Name: "res-1", Kind: "feedback", Description: "fixed", Action: "resolve", Issues: []string{"MY-20"}},
			},
			maxRetries:    3,
			wantRemaining: 0,
		},
		{
			name: "retry with issues under limit keeps dmail",
			dmails: []domain.DMail{
				{Name: "retry-1", Kind: "feedback", Description: "flaky", Action: "retry", Issues: []string{"MY-30"}},
			},
			maxRetries:         3,
			wantRemaining:      1,
			wantRetryAttempted: 1,
		},
		{
			name: "retry with issues at max promotes to escalation",
			dmails: []domain.DMail{
				{Name: "retry-max", Kind: "feedback", Description: "stuck", Action: "retry", Issues: []string{"MY-40"}},
			},
			maxRetries:     3,
			preloadRetries: 3, // Track called 3 times before; triage call will be the 4th > maxRetries
			preloadIssues:  []string{"MY-40"},
			wantRemaining:  0,
			wantEscalated:  1,
		},
		{
			name: "retry without issues passes through",
			dmails: []domain.DMail{
				{Name: "retry-noissue", Kind: "feedback", Description: "generic retry", Action: "retry"},
			},
			maxRetries:    3,
			wantRemaining: 1,
		},
		{
			name: "mixed actions filter correctly",
			dmails: []domain.DMail{
				{Name: "esc-mix", Kind: "feedback", Description: "escalate me", Action: "escalate", Issues: []string{"MY-50"}},
				{Name: "pass-mix", Kind: "report", Description: "just info"},
				{Name: "res-mix", Kind: "feedback", Description: "resolved", Action: "resolve", Issues: []string{"MY-51"}},
				{Name: "retry-mix", Kind: "feedback", Description: "try again", Action: "retry", Issues: []string{"MY-52"}},
			},
			maxRetries:         3,
			wantRemaining:      2, // pass-mix + retry-mix
			wantEscalated:      1, // esc-mix
			wantRetryAttempted: 1, // retry-mix
		},
		{
			name: "unknown action passes through",
			dmails: []domain.DMail{
				{Name: "unknown-1", Kind: "report", Description: "future action", Action: "future-action"},
			},
			maxRetries:    3,
			wantRemaining: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			emitter := &countingEmitter{}
			tracker := domain.NewRetryTracker()

			// preload retries to simulate previous attempts
			for range tt.preloadRetries {
				tracker.Track(tt.preloadIssues)
			}

			p := &Paintress{
				Emitter:      emitter,
				config:       domain.Config{MaxRetries: tt.maxRetries},
				Logger:       &domain.NopLogger{},
				retryTracker: tracker,
			}

			// when
			result := p.triagePreFlightDMails(context.Background(), tt.dmails)

			// then
			if got := len(result); got != tt.wantRemaining {
				t.Errorf("remaining D-Mails = %d, want %d", got, tt.wantRemaining)
			}
			if emitter.escalatedCount != tt.wantEscalated {
				t.Errorf("escalated count = %d, want %d", emitter.escalatedCount, tt.wantEscalated)
			}
			if emitter.retryAttemptedCount != tt.wantRetryAttempted {
				t.Errorf("retryAttempted count = %d, want %d", emitter.retryAttemptedCount, tt.wantRetryAttempted)
			}
		})
	}
}

// writeDMailFile marshals a DMail and writes it to the inbox directory.
func writeDMailFile(t *testing.T, inboxDir string, dm domain.DMail) {
	t.Helper()
	data, err := dm.Marshal()
	if err != nil {
		t.Fatalf("marshal dmail %s: %v", dm.Name, err)
	}
	if err := os.WriteFile(filepath.Join(inboxDir, dm.Name+".md"), data, 0644); err != nil {
		t.Fatalf("write dmail %s: %v", dm.Name, err)
	}
}

// fileExists reports whether the given path exists on the filesystem.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestTriagePreFlightDMails_Filesystem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		dmails         []domain.DMail
		maxRetries     int
		preloadRetries int
		preloadIssues  []string
		wantInInbox    []string // file names expected to remain in inbox
		wantInArchive  []string // file names expected to appear in archive
	}{
		{
			name: "escalate moves file from inbox to archive",
			dmails: []domain.DMail{
				{Name: "esc-fs-1", Kind: "feedback", Description: "critical", Action: "escalate", Issues: []string{"FS-10"}, SchemaVersion: domain.DMailSchemaVersion},
			},
			maxRetries:    3,
			wantInInbox:   nil,
			wantInArchive: []string{"esc-fs-1"},
		},
		{
			name: "resolve moves file from inbox to archive",
			dmails: []domain.DMail{
				{Name: "res-fs-1", Kind: "feedback", Description: "done", Action: "resolve", Issues: []string{"FS-20"}, SchemaVersion: domain.DMailSchemaVersion},
			},
			maxRetries:    3,
			wantInInbox:   nil,
			wantInArchive: []string{"res-fs-1"},
		},
		{
			name: "no action keeps file in inbox",
			dmails: []domain.DMail{
				{Name: "info-fs-1", Kind: "report", Description: "status update", SchemaVersion: domain.DMailSchemaVersion},
			},
			maxRetries:    3,
			wantInInbox:   []string{"info-fs-1"},
			wantInArchive: nil,
		},
		{
			name: "retry under limit keeps file in inbox",
			dmails: []domain.DMail{
				{Name: "retry-fs-1", Kind: "feedback", Description: "flaky", Action: "retry", Issues: []string{"FS-30"}, SchemaVersion: domain.DMailSchemaVersion},
			},
			maxRetries:    3,
			wantInInbox:   []string{"retry-fs-1"},
			wantInArchive: nil,
		},
		{
			name: "retry over limit moves file to archive",
			dmails: []domain.DMail{
				{Name: "retry-fs-max", Kind: "feedback", Description: "stuck", Action: "retry", Issues: []string{"FS-40"}, SchemaVersion: domain.DMailSchemaVersion},
			},
			maxRetries:     3,
			preloadRetries: 3,
			preloadIssues:  []string{"FS-40"},
			wantInInbox:    nil,
			wantInArchive:  []string{"retry-fs-max"},
		},
		{
			name: "mixed actions archive consumed and keep pass-through",
			dmails: []domain.DMail{
				{Name: "esc-fs-mix", Kind: "feedback", Description: "escalate me", Action: "escalate", Issues: []string{"FS-50"}, SchemaVersion: domain.DMailSchemaVersion},
				{Name: "pass-fs-mix", Kind: "report", Description: "just info", SchemaVersion: domain.DMailSchemaVersion},
				{Name: "res-fs-mix", Kind: "feedback", Description: "resolved", Action: "resolve", Issues: []string{"FS-51"}, SchemaVersion: domain.DMailSchemaVersion},
				{Name: "retry-fs-mix", Kind: "feedback", Description: "try again", Action: "retry", Issues: []string{"FS-52"}, SchemaVersion: domain.DMailSchemaVersion},
			},
			maxRetries:    3,
			wantInInbox:   []string{"pass-fs-mix", "retry-fs-mix"},
			wantInArchive: []string{"esc-fs-mix", "res-fs-mix"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given — set up real filesystem with inbox and archive dirs
			tmpDir := t.TempDir()
			inboxDir := filepath.Join(tmpDir, domain.StateDir, "inbox")
			archiveDir := filepath.Join(tmpDir, domain.StateDir, "archive")
			if err := os.MkdirAll(inboxDir, 0755); err != nil {
				t.Fatalf("mkdir inbox: %v", err)
			}

			for _, dm := range tt.dmails {
				writeDMailFile(t, inboxDir, dm)
			}

			emitter := &countingEmitter{}
			tracker := domain.NewRetryTracker()
			for range tt.preloadRetries {
				tracker.Track(tt.preloadIssues)
			}

			p := &Paintress{
				Emitter:      emitter,
				config:       domain.Config{Continent: tmpDir, MaxRetries: tt.maxRetries},
				Logger:       &domain.NopLogger{},
				retryTracker: tracker,
			}

			// when
			result := p.triagePreFlightDMails(context.Background(), tt.dmails)

			// then — verify return value length
			wantRemaining := len(tt.wantInInbox)
			if got := len(result); got != wantRemaining {
				t.Errorf("remaining D-Mails = %d, want %d", got, wantRemaining)
			}

			// then — verify files still in inbox
			for _, name := range tt.wantInInbox {
				path := filepath.Join(inboxDir, name+".md")
				if !fileExists(path) {
					t.Errorf("expected %s to remain in inbox, but file not found", name)
				}
			}

			// then — verify files moved to archive
			for _, name := range tt.wantInArchive {
				srcPath := filepath.Join(inboxDir, name+".md")
				dstPath := filepath.Join(archiveDir, name+".md")
				if fileExists(srcPath) {
					t.Errorf("expected %s to be removed from inbox, but file still exists", name)
				}
				if !fileExists(dstPath) {
					t.Errorf("expected %s to appear in archive, but file not found", name)
				}
			}
		})
	}
}
