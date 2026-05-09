package session_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform/actortype"
	"github.com/hironow/paintress/internal/session"
)

func TestSendDMail_EmitsActorType_Env(t *testing.T) {
	// given
	t.Setenv("RUNOPS_ACTOR_TYPE", "ai-agent")
	t.Setenv("RUNOPS_INITIATING_ACTOR_TYPE", "")
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	dm := domain.DMail{
		Name:        "pt-report-actortype-env_00000000",
		Kind:        "report",
		Description: "with actor type",
		Body:        "# Hello\n",
	}

	// when
	if err := session.SendDMail(context.Background(), store, dm, nil); err != nil {
		t.Fatalf("SendDMail: %v", err)
	}

	// then
	outboxPath := filepath.Join(domain.OutboxDir(continent), dm.Name+".md")
	data, err := os.ReadFile(outboxPath)
	if err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("parse outbox: %v", err)
	}
	if got := parsed.Metadata["requester_actor_type"]; got != "ai-agent" {
		t.Errorf("requester_actor_type: got %q, want ai-agent", got)
	}
	if got := parsed.Metadata["requester_actor_source"]; got != "env" {
		t.Errorf("requester_actor_source: got %q, want env", got)
	}
	if _, ok := parsed.Metadata["initiating_actor_type"]; ok {
		t.Errorf("non-daemon actor must not carry initiating_actor_type")
	}
}

func TestSendDMail_EmitsActorType_DaemonWithInitiating(t *testing.T) {
	// given
	t.Setenv("RUNOPS_ACTOR_TYPE", "workspace-daemon")
	t.Setenv("RUNOPS_INITIATING_ACTOR_TYPE", "human-operator")
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	dm := domain.DMail{
		Name:        "pt-report-actortype-daemon_00000000",
		Kind:        "report",
		Description: "daemon-driven",
		Body:        "# Hello\n",
	}

	// when
	if err := session.SendDMail(context.Background(), store, dm, nil); err != nil {
		t.Fatalf("SendDMail: %v", err)
	}

	// then
	outboxPath := filepath.Join(domain.OutboxDir(continent), dm.Name+".md")
	data, err := os.ReadFile(outboxPath)
	if err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("parse outbox: %v", err)
	}
	if got := parsed.Metadata["requester_actor_type"]; got != "workspace-daemon" {
		t.Errorf("requester_actor_type: got %q, want workspace-daemon", got)
	}
	if got := parsed.Metadata["initiating_actor_type"]; got != "human-operator" {
		t.Errorf("initiating_actor_type: got %q, want human-operator", got)
	}
}

func TestSendDMail_NoActorType_LegacyCompat(t *testing.T) {
	// given — env unset (legacy compat path)
	t.Setenv("RUNOPS_ACTOR_TYPE", "")
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	dm := domain.DMail{
		Name:        "pt-report-actortype-legacy_00000000",
		Kind:        "report",
		Description: "legacy compat",
		Body:        "# Hello\n",
	}

	// when
	if err := session.SendDMail(context.Background(), store, dm, nil); err != nil {
		t.Fatalf("SendDMail: %v", err)
	}

	// then — frontmatter must not carry actor type lines (byte-identical legacy)
	outboxPath := filepath.Join(domain.OutboxDir(continent), dm.Name+".md")
	data, err := os.ReadFile(outboxPath)
	if err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	parsed, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("parse outbox: %v", err)
	}
	if v, ok := parsed.Metadata["requester_actor_type"]; ok {
		t.Errorf("requester_actor_type must be absent in legacy compat path, got %q", v)
	}
	if v, ok := parsed.Metadata["requester_actor_source"]; ok {
		t.Errorf("requester_actor_source must be absent in legacy compat path, got %q", v)
	}
}

func TestSendDMail_InvalidEnv_FailsEmit(t *testing.T) {
	// given
	t.Setenv("RUNOPS_ACTOR_TYPE", "robot")
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	dm := domain.DMail{
		Name:        "pt-report-actortype-invalid_00000000",
		Kind:        "report",
		Description: "invalid env",
		Body:        "# Hello\n",
	}

	// when
	err := session.SendDMail(context.Background(), store, dm, nil)

	// then
	if err == nil {
		t.Fatal("expected error for invalid RUNOPS_ACTOR_TYPE env, got nil")
	}
	if !errors.Is(err, actortype.ErrInvalidActorType) {
		t.Errorf("expected ErrInvalidActorType wrapped, got %v", err)
	}
	// outbox MUST NOT have the file (silent escalation prevention)
	outboxPath := filepath.Join(domain.OutboxDir(continent), dm.Name+".md")
	if _, statErr := os.Stat(outboxPath); statErr == nil {
		t.Errorf("outbox file must not exist after emit fail, but %s does", outboxPath)
	}
}

func TestSendDMail_DaemonInvalidInitiating_FailsEmit(t *testing.T) {
	// given
	t.Setenv("RUNOPS_ACTOR_TYPE", "workspace-daemon")
	t.Setenv("RUNOPS_INITIATING_ACTOR_TYPE", "robot")
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	dm := domain.DMail{
		Name:        "pt-report-actortype-daemoninvalid_00000000",
		Kind:        "report",
		Description: "daemon with invalid initiating",
		Body:        "# Hello\n",
	}

	// when
	err := session.SendDMail(context.Background(), store, dm, nil)

	// then
	if err == nil {
		t.Fatal("expected error for invalid RUNOPS_INITIATING_ACTOR_TYPE env, got nil")
	}
	if !errors.Is(err, actortype.ErrInvalidInitiatingActorType) {
		t.Errorf("expected ErrInvalidInitiatingActorType wrapped, got %v", err)
	}
	outboxPath := filepath.Join(domain.OutboxDir(continent), dm.Name+".md")
	if _, statErr := os.Stat(outboxPath); statErr == nil {
		t.Errorf("outbox file must not exist after emit fail, but %s does", outboxPath)
	}
}
