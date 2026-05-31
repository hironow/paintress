//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func paintressBin() string {
	if env := os.Getenv("PAINTRESS_BIN"); env != "" {
		return env
	}
	return "paintress"
}

func TestE2E_MCPServerToolsList(t *testing.T) {
	// given
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`

	// when
	cmd := exec.Command(paintressBin(), "mcp")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("mcp command failed: %v\nstderr: %s", err, stderr.String())
	}

	// then
	outStr := stdout.String()
	idx := strings.Index(outStr, `{"jsonrpc"`)
	if idx < 0 {
		t.Fatalf("no JSON-RPC response found in stdout: %s", outStr)
	}
	jsonStr := outStr[idx:]

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
