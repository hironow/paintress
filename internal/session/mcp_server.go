package session

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

// MCPServer is a minimal stdio-based Model Context Protocol server
// scaffolded for the refs/issues/0027 jun15 MCP pivot (Phase 1 MVP).
//
// This is a SKELETON: only the paintress.ping health-check tool is
// exposed. Real tools (paintress.next_issue, paintress.update_gradient,
// paintress.append_journal, ...) ship in subsequent commits on the
// feat/jun15-mcp-pivot branch.
//
// Wire it into a claude code interactive session via --mcp-config so
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
// returning an "uninitialized" payload so the claude code session can
// surface a clear error to the operator.
type MCPServer struct {
	in        io.Reader
	out       io.Writer
	logger    domain.Logger
	continent string
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

// jsonrpcMessage is the minimum JSON-RPC 2.0 envelope this skeleton
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
	case "tools/list":
		return s.respond(msg.ID, map[string]any{"tools": toolDescriptors()})
	case "tools/call":
		return s.handleToolsCall(ctx, msg)
	default:
		return s.respondError(msg.ID, -32601, fmt.Sprintf("method not implemented: %s", msg.Method))
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
		result = stubUpdateGradient(call.Arguments)
		status = "deprecated"
	case "paintress.append_journal":
		result = stubAppendJournal(call.Arguments)
		status = "deprecated"
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

// toolDescriptors returns the Phase 1 MVP tool set. Each entry pins the
// interface (name, description, inputSchema) so claude code clients see
// a stable contract. The handler bodies (stubNextIssue / stubUpdateGradient
// / stubAppendJournal) are placeholders that ship in subsequent commits
// with real domain wiring.
func toolDescriptors() []map[string]any {
	return []map[string]any{
		{
			"name":        "paintress.ping",
			"description": "Health check. Returns 'pong'.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "paintress.next_issue",
			"description": "Return the next expedition target issue (Phase 1: stub returns a placeholder Issue payload until the domain wiring lands).",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "paintress.update_gradient",
			"description": "Update the gradient gauge level by delta (Phase 1: stub returns the requested delta and a placeholder level).",
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
			"description": "Append a journal entry (Phase 1: stub echoes the entry without persisting).",
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
// issue ids + next expedition number + last PR) so the claude code
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

// stubUpdateGradient echoes the requested delta with a placeholder new
// level so claude code clients can exercise the contract end-to-end.
func stubUpdateGradient(args json.RawMessage) map[string]any {
	var payload struct {
		Delta int `json:"delta"`
	}
	if len(args) > 0 {
		_ = json.Unmarshal(args, &payload)
	}
	return jsonResult(map[string]any{
		"stub":      true,
		"delta":     payload.Delta,
		"new_level": payload.Delta,
		"reason":    "phase-1-mvp: real gradient gauge wiring lands when the harness/policy package is exposed",
	})
}

// stubAppendJournal echoes the entry payload without persisting.
func stubAppendJournal(args json.RawMessage) map[string]any {
	var entry map[string]any
	if len(args) > 0 {
		_ = json.Unmarshal(args, &entry)
	}
	if entry == nil {
		entry = map[string]any{}
	}
	return jsonResult(map[string]any{
		"stub":   true,
		"entry":  entry,
		"reason": "phase-1-mvp: real journal append lands when domain.JournalEntry persistence is wired",
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
