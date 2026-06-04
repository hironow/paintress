package session

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/usecase/port"
)

// MCPServer is a stdio-based Model Context Protocol server for the
// refs/issues/0027 jun15 MCP pivot.
//
// All four tools are real implementations: paintress.ping (health
// check), paintress.next_issue (reads journal + pr-index state), and
// paintress.update_gradient + paintress.append_journal (persist
// EventGradientChanged / EventExpeditionCompleted via the event store
// when an emitter is wired; cmd wires one by default).
//
// Wire it into a Claude Code interactive session via --mcp-config so
// inference stays on the human-initiated session's subscription quota
// rather than crossing into the Agent SDK credit pool that gates
// `claude -p` from 2026-06-15.
//
// Protocol: JSON-RPC 2.0 over stdio, one envelope per line. Stderr
// carries human-readable diagnostics (per the project stdout/stderr
// separation invariant).
//
// continent is the project root directory (= paintress's "continent"
// abstraction) used to resolve journal / pr-index paths for the
// real-impl MCP tools. When empty, real-impl tools fall back to
// returning an "uninitialized" payload so the Claude Code session can
// surface a clear error to the operator.
type MCPServer struct {
	in        io.Reader
	out       io.Writer
	logger    domain.Logger
	continent string
	emitter   port.ExpeditionEventEmitter
}

// NewMCPServer wires explicit I/O so tests can drive the server
// without subprocess overhead. Passing nil for logger uses NopLogger.
// continent defaults to empty string; use WithContinent to configure
// it, typically to os.Getwd() in the cobra subcommand RunE.
func NewMCPServer(in io.Reader, out io.Writer, logger domain.Logger) *MCPServer {
	if logger == nil {
		logger = &domain.NopLogger{}
	}
	return &MCPServer{in: in, out: out, logger: logger}
}

// WithContinent sets the project root used by real-impl MCP tools to
// resolve journal / pr-index paths. Returns s for chaining.
func (s *MCPServer) WithContinent(continent string) *MCPServer {
	s.continent = continent
	return s
}

// WithEmitter wires the usecase ExpeditionEventEmitter used to emit
// EventGradientChanged / EventExpeditionCompleted from
// paintress.update_gradient / paintress.append_journal
// (refs/issues/0027 Phase 4 follow-up #4). The cmd composition root
// always wires a real emitter, so production calls persist events.
// Passing nil falls back to preview-only (update_gradient) /
// filesystem-only (append_journal), which tests use to keep emission
// opt-in.
//
// LLM firing remains human-initiated: emission happens only when the
// claude-code session calls the MCP tool.
func (s *MCPServer) WithEmitter(emitter port.ExpeditionEventEmitter) *MCPServer {
	s.emitter = emitter
	return s
}

// jsonrpcMessage is the minimum JSON-RPC 2.0 envelope this server
// understands. Method-specific params decode on demand from
// Params (json.RawMessage).
type jsonrpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Serve reads messages from in line-by-line and writes responses to
// out until ctx cancels or stdin closes. Per-message decode errors
// surface as JSON-RPC error responses; only stream-level read errors
// abort Serve.
func (s *MCPServer) Serve(ctx context.Context) error {
	scanner := bufio.NewScanner(s.in)
	// 4 MiB buffer to comfortably cover D-Mail bodies in later commits.
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if err := s.handle(ctx, line); err != nil {
			s.logger.Warn("mcp server: handle: %v", err)
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("mcp server: read stdin: %w", err)
	}
	return nil
}

func (s *MCPServer) handle(ctx context.Context, line []byte) error {
	var msg jsonrpcMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	switch msg.Method {
	case "initialize":
		return s.respond(msg.ID, initializeResult())
	case "notifications/initialized":
		// JSON-RPC notification (no id): the client signals it finished
		// the handshake. No response is sent.
		return nil
	case "tools/list":
		return s.respond(msg.ID, map[string]any{"tools": toolDescriptors()})
	case "tools/call":
		return s.handleToolsCall(ctx, msg)
	default:
		// Unknown notifications (no id) are ignored per JSON-RPC; only
		// id-bearing requests get a method-not-found error.
		if len(msg.ID) == 0 {
			return nil
		}
		return s.respondError(msg.ID, -32601, fmt.Sprintf("method not implemented: %s", msg.Method))
	}
}

// mcpProtocolVersion is the single MCP protocol version this server
// implements. Per the MCP lifecycle spec, the server returns the
// version it actually supports (not an echo of the client's request):
// echoing an unsupported client version would falsely claim support
// and break future / draft clients. The client decides compatibility
// from this value.
const mcpProtocolVersion = "2024-11-05"

// initializeResult builds the MCP initialize handshake response. The
// Claude Code session sends `initialize` first; without a valid reply
// it never proceeds to tools/list. The server advertises its supported
// protocol version + the tools capability.
func initializeResult() map[string]any {
	return map[string]any{
		"protocolVersion": mcpProtocolVersion,
		"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
		"serverInfo":      map[string]any{"name": "paintress", "version": "0.1.0"},
	}
}

// handleToolsCall dispatches a single tools/call request and records
// MCP invocation metrics (mcp.tool.invocations counter +
// mcp.tool.duration histogram) for cost-monitoring verification post
// 2026-06-15 (refs/issues/0027 Phase 3 cost monitoring (a)).
func (s *MCPServer) handleToolsCall(ctx context.Context, msg jsonrpcMessage) error {
	start := time.Now()
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(msg.Params, &call); err != nil {
		platform.RecordMCPInvocation(ctx, "", "error", time.Since(start))
		return s.respondError(msg.ID, -32602, "invalid tools/call params")
	}

	status := "ok"
	var result map[string]any
	switch call.Name {
	case "paintress.ping":
		result = textResult("pong")
	case "paintress.next_issue":
		result = realNextIssue(s.continent)
	case "paintress.update_gradient":
		result = realUpdateGradient(ctx, s.continent, s.emitter, call.Arguments, s.logger)
	case "paintress.append_journal":
		result = realAppendJournal(s.continent, s.emitter, call.Arguments)
	default:
		platform.RecordMCPInvocation(ctx, call.Name, "error", time.Since(start))
		return s.respondError(msg.ID, -32601, fmt.Sprintf("unknown tool: %s", call.Name))
	}

	err := s.respond(msg.ID, result)
	if err != nil {
		status = "error"
	}
	platform.RecordMCPInvocation(ctx, call.Name, status, time.Since(start))
	return err
}

// toolDescriptors returns the tool set. Each entry pins the interface
// (name, description, inputSchema) so Claude Code clients see a stable
// contract. next_issue / update_gradient / append_journal are real
// impl: they read pr-index / event store and write journal/ + pr-index;
// update_gradient / append_journal also emit EventGradientChanged /
// EventExpeditionCompleted when an emitter is wired (cmd wires one).
func toolDescriptors() []map[string]any {
	return []map[string]any{
		{
			"name":        "paintress.ping",
			"description": "Health check. Returns 'pong'.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "paintress.next_issue",
			"description": "Return paintress's local journal state (completed_issue_ids + next_expedition_number + last_pr). The Claude Code session queries linear-mcp separately and uses completed_issue_ids to exclude already-done work.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "paintress.update_gradient",
			"description": "Read current gradient_level from the event store, apply delta, and persist an EventGradientChanged event (persistence='event-store'). Returns current_level + new_level. Falls back to a preview without persisting when no emitter is wired.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"delta": map[string]any{"type": "integer", "description": "signed level change"},
				},
				"required": []any{"delta"},
			},
		},
		{
			"name":        "paintress.append_journal",
			"description": "Persist an ExpeditionReport to journal/<NNN>.md + pr-index and emit an EventExpeditionCompleted event (persistence='event-store+filesystem'). Falls back to filesystem-only when no emitter is wired.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"expedition":   map[string]any{"type": "integer"},
					"date":         map[string]any{"type": "string"},
					"issue_id":     map[string]any{"type": "string"},
					"issue_title":  map[string]any{"type": "string"},
					"mission_type": map[string]any{"type": "string"},
					"status":       map[string]any{"type": "string"},
					"reason":       map[string]any{"type": "string"},
					"pr_url":       map[string]any{"type": "string"},
				},
				"required": []any{"expedition", "issue_id", "status"},
			},
		},
	}
}

// textResult wraps a plain string into the MCP content envelope.
func textResult(text string) map[string]any {
	return map[string]any{"content": []map[string]any{{"type": "text", "text": text}}}
}

// jsonResult marshals data as JSON and returns an MCP content envelope.
func jsonResult(data any) map[string]any {
	body, err := json.Marshal(data)
	if err != nil {
		return textResult(fmt.Sprintf(`{"error":"marshal failed: %v"}`, err))
	}
	return map[string]any{"content": []map[string]any{{"type": "text", "text": string(body)}}}
}

// realNextIssue surfaces paintress's local journal state (= completed
// issue ids + next expedition number + last PR) so the Claude Code
// session can decide which Linear issue to fetch next via linear-mcp.
//
// paintress does NOT call linear-mcp itself (= that would re-introduce
// claude-driven inference, the very thing this pivot removes). The
// session, with both paintress mcp and linear-mcp attached, reads
// completed_issue_ids from this tool and excludes them from its
// linear-mcp query.
//
// continent is the project root resolved via WithContinent (typically
// os.Getwd() in the cobra subcommand). When empty or the journal
// directory is missing, the response indicates an uninitialized
// project so the session surfaces a clear error.
func realNextIssue(continent string) map[string]any {
	if continent == "" {
		return jsonResult(map[string]any{
			"initialized":            false,
			"reason":                 "paintress mcp continent root not configured (start `paintress mcp` from the project root or pass via WithContinent)",
			"next_expedition_number": 1,
			"completed_issue_ids":    []string{},
		})
	}
	entries, err := ReadPRIndex(continent)
	if err != nil {
		return jsonResult(map[string]any{
			"initialized": false,
			"reason":      fmt.Sprintf("pr-index read failed: %v", err),
			"continent":   continent,
		})
	}

	completedIDs := make([]string, 0, len(entries))
	maxExp := 0
	var lastPR map[string]any
	for _, e := range entries {
		completedIDs = append(completedIDs, e.IssueID)
		if e.Expedition > maxExp {
			maxExp = e.Expedition
		}
	}
	if len(entries) > 0 {
		last := entries[len(entries)-1]
		lastPR = map[string]any{
			"expedition": last.Expedition,
			"issue_id":   last.IssueID,
			"pr_url":     last.PRUrl,
		}
	}

	return jsonResult(map[string]any{
		"initialized":            true,
		"continent":              continent,
		"next_expedition_number": maxExp + 1,
		"completed_issue_ids":    completedIDs,
		"last_pr":                lastPR,
		"journal_dir":            domain.JournalDir(continent),
		"instruction":            "Query linear-mcp for unstarted issues, exclude completed_issue_ids, pick highest priority. Persist completion via paintress.append_journal after the expedition.",
	})
}

// realUpdateGradient reads the current GradientLevel via the event
// store + projection, applies the delta, and emits an
// EventGradientChanged via the injected emitter (Phase 4 follow-up #4,
// persistence='event-store'). When no emitter is wired (tests /
// opt-out), it returns a preview of (current + delta) without
// persisting. The session can re-read the new level via the next
// projection.
//
// continent is the project root from MCPServer.WithContinent. When
// empty the response signals uninitialized so the session aborts.
func realUpdateGradient(ctx context.Context, continent string, emitter port.ExpeditionEventEmitter, args json.RawMessage, logger domain.Logger) map[string]any {
	var payload struct {
		Delta int `json:"delta"`
	}
	if len(args) > 0 {
		_ = json.Unmarshal(args, &payload)
	}
	if continent == "" {
		return jsonResult(map[string]any{
			"initialized":   false,
			"reason":        "paintress mcp continent root not configured",
			"delta":         payload.Delta,
			"current_level": 0,
			"preview_level": payload.Delta,
		})
	}
	stateDir := filepath.Join(continent, domain.StateDir)
	store := NewEventStore(stateDir, logger)
	events, _, err := store.LoadAll(ctx)
	if err != nil {
		return jsonResult(map[string]any{
			"initialized":   false,
			"reason":        fmt.Sprintf("event store load failed: %v", err),
			"delta":         payload.Delta,
			"current_level": 0,
			"preview_level": payload.Delta,
		})
	}
	state := ProjectState(events)
	newLevel := state.GradientLevel + payload.Delta
	if emitter == nil {
		return jsonResult(map[string]any{
			"initialized":   true,
			"continent":     continent,
			"current_level": state.GradientLevel,
			"delta":         payload.Delta,
			"preview_level": newLevel,
			"persistence":   "preview-only",
			"note":          "Preview only. Emitter not wired; cmd composition root injects one via MCPServer.WithEmitter to persist EventGradientChanged.",
		})
	}
	if err := emitter.EmitGradientChange(newLevel, "mcp.update_gradient", time.Now().UTC()); err != nil {
		return jsonResult(map[string]any{
			"initialized":   true,
			"continent":     continent,
			"current_level": state.GradientLevel,
			"delta":         payload.Delta,
			"preview_level": newLevel,
			"persistence":   "preview-only",
			"reason":        fmt.Sprintf("emit gradient change: %v", err),
		})
	}
	return jsonResult(map[string]any{
		"initialized":   true,
		"continent":     continent,
		"current_level": state.GradientLevel,
		"delta":         payload.Delta,
		"new_level":     newLevel,
		"persistence":   "event-store",
	})
}

// realAppendJournal writes the expedition report to the journal
// directory and pr-index file via the existing WriteJournal /
// WritePRIndex helpers, then emits an EventExpeditionCompleted via the
// injected emitter (Phase 4 follow-up #4, persistence=
// 'event-store+filesystem'). When no emitter is wired (tests /
// opt-out), it persists filesystem-only. The session can re-read the
// new state via paintress.next_issue.
//
// continent is the project root from MCPServer.WithContinent. When
// empty the response signals uninitialized so the session aborts.
//
//nolint:staticcheck // intentional: documents the existing journal/pr-index files maintained by session/journal.go
func realAppendJournal(continent string, emitter port.ExpeditionEventEmitter, args json.RawMessage) map[string]any {
	var payload struct {
		Expedition         int    `json:"expedition"`
		IssueID            string `json:"issue_id"`
		IssueTitle         string `json:"issue_title"`
		MissionType        string `json:"mission_type"`
		Branch             string `json:"branch"`
		PRUrl              string `json:"pr_url"`
		Status             string `json:"status"`
		Reason             string `json:"reason"`
		Remaining          string `json:"remaining"`
		BugsFound          int    `json:"bugs_found"`
		BugIssues          string `json:"bug_issues"`
		Insight            string `json:"insight"`
		FailureType        string `json:"failure_type"`
		HighSeverityDMails string `json:"high_severity_dmails"`
		WaveID             string `json:"wave_id"`
		StepID             string `json:"step_id"`
	}
	if len(args) > 0 {
		_ = json.Unmarshal(args, &payload)
	}
	if continent == "" {
		return jsonResult(map[string]any{
			"initialized": false,
			"reason":      "paintress mcp continent root not configured",
		})
	}
	if payload.Expedition <= 0 || payload.IssueID == "" || payload.Status == "" {
		return jsonResult(map[string]any{
			"initialized": true,
			"persisted":   false,
			"reason":      "missing required fields: expedition (>0), issue_id, status",
			"received":    payload,
		})
	}
	report := &domain.ExpeditionReport{
		Expedition:         payload.Expedition,
		IssueID:            payload.IssueID,
		IssueTitle:         payload.IssueTitle,
		MissionType:        payload.MissionType,
		Branch:             payload.Branch,
		PRUrl:              payload.PRUrl,
		Status:             payload.Status,
		Reason:             payload.Reason,
		Remaining:          payload.Remaining,
		BugsFound:          payload.BugsFound,
		BugIssues:          payload.BugIssues,
		Insight:            payload.Insight,
		FailureType:        payload.FailureType,
		HighSeverityDMails: payload.HighSeverityDMails,
		WaveID:             payload.WaveID,
		StepID:             payload.StepID,
	}
	if err := WriteJournal(continent, report); err != nil {
		return jsonResult(map[string]any{
			"initialized": true,
			"persisted":   false,
			"reason":      fmt.Sprintf("write journal: %v", err),
		})
	}
	if err := WritePRIndex(continent, report); err != nil {
		return jsonResult(map[string]any{
			"initialized":  true,
			"persisted":    true,
			"journal_file": filepath.Join(domain.JournalDir(continent), fmt.Sprintf("%03d.md", report.Expedition)),
			"pr_index":     false,
			"reason":       fmt.Sprintf("journal written but pr-index append failed: %v", err),
		})
	}
	if emitter == nil {
		return jsonResult(map[string]any{
			"initialized":      true,
			"persisted":        true,
			"expedition":       report.Expedition,
			"issue_id":         report.IssueID,
			"journal_file":     filepath.Join(domain.JournalDir(continent), fmt.Sprintf("%03d.md", report.Expedition)),
			"pr_index_updated": report.PRUrl != "" && report.PRUrl != "none",
			"persistence":      "filesystem-only",
			"note":             "Filesystem persistence complete (journal/<NNN>.md + pr index). Emitter not wired; cmd composition root injects one via MCPServer.WithEmitter to also emit EventExpeditionCompleted.", // nosemgrep: layer-session-no-event-persistence -- comment text only, persistence is via session/journal.go::WriteJournal+WritePRIndex helpers that the rule allows [permanent]
		})
	}
	bugsFoundStr := strconv.Itoa(report.BugsFound)
	if err := emitter.EmitCompleteExpedition(report.Expedition, report.Status, report.IssueID, bugsFoundStr, report.WaveID, report.StepID, time.Now().UTC()); err != nil {
		return jsonResult(map[string]any{
			"initialized":      true,
			"persisted":        true,
			"expedition":       report.Expedition,
			"issue_id":         report.IssueID,
			"journal_file":     filepath.Join(domain.JournalDir(continent), fmt.Sprintf("%03d.md", report.Expedition)),
			"pr_index_updated": report.PRUrl != "" && report.PRUrl != "none",
			"persistence":      "filesystem-only",
			"reason":           fmt.Sprintf("emit expedition completed: %v", err),
		})
	}
	return jsonResult(map[string]any{
		"initialized":      true,
		"persisted":        true,
		"expedition":       report.Expedition,
		"issue_id":         report.IssueID,
		"journal_file":     filepath.Join(domain.JournalDir(continent), fmt.Sprintf("%03d.md", report.Expedition)),
		"pr_index_updated": report.PRUrl != "" && report.PRUrl != "none",
		"persistence":      "event-store+filesystem",
	})
}

func (s *MCPServer) respond(id json.RawMessage, result any) error {
	return s.writeMessage(jsonrpcMessage{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *MCPServer) respondError(id json.RawMessage, code int, message string) error {
	return s.writeMessage(jsonrpcMessage{JSONRPC: "2.0", ID: id, Error: &jsonrpcError{Code: code, Message: message}})
}

func (s *MCPServer) writeMessage(msg jsonrpcMessage) error {
	out, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	if _, err := s.out.Write(append(out, '\n')); err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	return nil
}
