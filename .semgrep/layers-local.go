package testdata

import "path/filepath"

// ==========================================================================
// layers-local.yaml test fixture
// Covers paintress-specific rules:
//   - no-state-dir-literal-in-path-join
// ==========================================================================

// --- Rule: no-state-dir-literal-in-path-join ---

func badStateDirLiteral() {
	// ruleid: no-state-dir-literal-in-path-join
	filepath.Join("/home", ".expedition")
}

func badStateDirLiteralWithSuffix() {
	// ruleid: no-state-dir-literal-in-path-join
	filepath.Join("/home", ".expedition", "events")
}

func goodStateDirConst(stateDir string) {
	// ok: no-state-dir-literal-in-path-join
	filepath.Join("/home", stateDir)
}

var _ = filepath.Join
