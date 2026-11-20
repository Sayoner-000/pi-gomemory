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
	t.Parallel()

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

// TestEmbeddedAntigravityPromptGuardRegression valida la extensión embebida
// con la suite offline y comprueba que InstallTUIExtension la instala y la
// registra en el manifiesto para que uninstall pueda retirarla.
func TestEmbeddedAntigravityPromptGuardRegression(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	extAbs, err := filepath.Abs(filepath.Join("..", "..", "extensions", "antigravity-prompt-guard.js"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, "node", "--test",
		filepath.Join("..", "..", "tests", "antigravity-prompt-guard", "guard.test.mjs"))
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if key != "AGY_GUARD_MODULE" && key != "ANTIGRAVITY_PROMPT_GUARD" {
			cmd.Env = append(cmd.Env, value)
		}
	}
	cmd.Env = append(cmd.Env, "AGY_GUARD_MODULE=file://"+extAbs)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("embedded prompt-guard regression failed: %v\n%s", err, out)
	}

	opts := sandbox(t)
	if err := InstallTUIExtension(repoAssets(t), opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(opts.PiAgentDir, "extensions", "antigravity-prompt-guard.js")
	if readFile(t, dst) != readAsset(t, "extensions/antigravity-prompt-guard.js") {
		t.Fatal("antigravity-prompt-guard.js no se instala o difiere del embebido")
	}
	found := false
	for _, f := range installedFiles(opts) {
		found = found || f == dst
	}
	if !found {
		t.Fatal("antigravity-prompt-guard.js no figura en el manifiesto: uninstall no lo retiraría")
	}
}
