//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
)

// TestE2E_PaintressInit verifies that `paintress init` creates the expected
// directory structure in a clean working directory.
func TestE2E_PaintressInit(t *testing.T) {
	ctx := context.Background()
	c := buildTestContainer(t, ctx)
	dir := "/workspace/t_paintress_init"

	// given: a git repository inside container
	execInContainer(t, ctx, c, []string{"mkdir", "-p", dir})
	execInContainer(t, ctx, c, []string{"sh", "-c", fmt.Sprintf("cd %s && git init --initial-branch=main", dir)})

	// when: run paintress init (requires repo path as positional arg)
	out, _, err := runCmd(t, ctx, c, dir, "init", "--lang", "en", dir)
	if err != nil {
		t.Fatalf("paintress init: %v\n%s", err, out)
	}

	// then: expedition directory structure exists
	for _, sub := range []string{
		".expedition",
		".expedition/inbox",
		".expedition/outbox",
		".expedition/archive",
		".expedition/.run",
	} {
		path := fmt.Sprintf("%s/%s", dir, sub)
		if !dirExistsInContainer(t, ctx, c, path) {
			t.Errorf("expected directory %s to exist in container", path)
		}
	}
}
