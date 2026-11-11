package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTasksExtension(t *testing.T) {
	s := readAsset(t, "extensions/gomemory-tasks.ts")
	for _, want := range []string{
		`"hcd", "show", "--json"`, "timeout: 5000", "setWidget", `"gomemory-tasks"`,
		"✓", "○", "✗", `"session_start"`, `"tool_result"`, `"agent_end"`,
		"mem hcd", `registerCommand("tasks"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("gomemory-tasks.ts no contiene %q", want)
		}
	}
}

func TestTasksExtensionInstalled(t *testing.T) {
	opts := sandbox(t)
	if err := InstallTUIExtension(repoAssets(t), opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(opts.PiAgentDir, "extensions", "gomemory-tasks.ts")); err != nil {
		t.Fatal("gomemory-tasks.ts no se instala")
	}
}
