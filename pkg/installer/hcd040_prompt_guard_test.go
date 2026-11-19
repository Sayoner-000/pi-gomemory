package installer

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestHCD040PromptGuardRegression runs the archived reference, never the
// installed extension. It performs no network or credential access.
func TestHCD040PromptGuardRegression(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "--test",
		filepath.Join("..", "..", "tests", "antigravity-prompt-guard", "guard.test.mjs"))
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if key != "AGY_GUARD_MODULE" && key != "ANTIGRAVITY_PROMPT_GUARD" {
			cmd.Env = append(cmd.Env, value)
		}
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("offline prompt-guard regression failed: %v\n%s", err, out)
	}
	t.Logf("%s", out)
}
