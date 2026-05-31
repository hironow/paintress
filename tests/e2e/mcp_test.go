//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestE2E_MCPServerToolsList(t *testing.T) {
	ctx := context.Background()
	c := buildTestContainer(t, ctx)
	dir := "/workspace/t_mcp"
	initTestRepo(t, ctx, c, dir)

	// given
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`

	// when
	stdout, _, err := runCmdStdin(t, ctx, c, dir, input, "mcp")
	if err != nil {
		t.Fatalf("mcp command failed: %v", err)
	}

	// then
	idx := strings.Index(stdout, `{"jsonrpc"`)
	if idx < 0 {
		t.Fatalf("no JSON-RPC response found in stdout: %s", stdout)
	}
	jsonStr := stdout[idx:]

	var resp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON-RPC response: %v\nraw: %s", err, jsonStr)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", resp.JSONRPC)
	}

	if resp.ID != 1 {
		t.Errorf("expected id 1, got %d", resp.ID)
	}

	expectedTools := map[string]bool{
		"paintress.ping":            false,
		"paintress.next_issue":      false,
		"paintress.update_gradient": false,
		"paintress.append_journal":  false,
	}

	for _, tool := range resp.Result.Tools {
		if _, ok := expectedTools[tool.Name]; ok {
			expectedTools[tool.Name] = true
		}
	}

	for name, found := range expectedTools {
		if !found {
			t.Errorf("missing expected tool in MCP response: %s", name)
		}
	}
}
