package session

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// realDMail emits a D-Mail through the transactional outbox (refs
// issue 0031). Arguments are a typed subset of the D-Mail v1 schema;
// direct outbox/ writes from the session remain forbidden because they
// would bypass the SQLite stage -> atomic flush contract phonewave's
// watcher depends on. SendDMail also emits dmail.staged /
// dmail.flushed events when the expedition emitter is wired.
func realDMail(ctx context.Context, continent string, emitter port.ExpeditionEventEmitter, args json.RawMessage) map[string]any {
	var payload struct {
		Kind        string            `json:"kind"`
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Body        string            `json:"body"`
		Issues      []string          `json:"issues"`
		Severity    string            `json:"severity"`
		Priority    int               `json:"priority"`
		Metadata    map[string]string `json:"metadata"`
	}
	if len(args) > 0 {
		_ = json.Unmarshal(args, &payload)
	}
	if continent == "" {
		return jsonResult(map[string]any{
			"initialized": false,
			"sent":        false,
			"reason":      "paintress mcp continent not configured (start `paintress mcp` from the project root)",
		})
	}
	mail, err := domain.NewProducedDMail(
		domain.DMailKind(payload.Kind),
		payload.Name,
		payload.Description,
		payload.Body,
		payload.Issues,
		payload.Severity,
		payload.Priority,
		payload.Metadata,
	)
	if err != nil {
		return jsonResult(map[string]any{
			"initialized": true,
			"sent":        false,
			"reason":      err.Error(),
		})
	}
	store, err := NewOutboxStoreForDir(continent)
	if err != nil {
		return jsonResult(map[string]any{
			"initialized": true,
			"sent":        false,
			"reason":      fmt.Sprintf("outbox store open failed: %v", err),
		})
	}
	defer func() { _ = store.Close() }()
	if err := SendDMail(ctx, store, mail, emitter); err != nil {
		return jsonResult(map[string]any{
			"initialized": true,
			"sent":        false,
			"reason":      fmt.Sprintf("dmail send failed (re-run dmail to retry): %v", err),
		})
	}
	return jsonResult(map[string]any{
		"initialized": true,
		"sent":        true,
		"name":        mail.Name,
		"filename":    mail.Name + ".md",
		"kind":        string(mail.Kind),
		"persistence": "transactional-outbox",
	})
}
