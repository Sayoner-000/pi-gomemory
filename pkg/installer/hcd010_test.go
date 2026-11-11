package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSidebarExtension(t *testing.T) {
	s := readAsset(t, "extensions/gomemory-sidebar.ts")
	for _, want := range []string{
		"gomemory-sidebar", `registerCommand("sidebar"`,
		"Context", "tokens", "% used", "$", "spent",
		"MCP", "Configured",
		"top-right", "visible: (w)",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("gomemory-sidebar.ts no contiene %q", want)
		}
	}
}

func TestSidebarExtensionInstalled(t *testing.T) {
	opts := sandbox(t)
	if err := InstallTUIExtension(repoAssets(t), opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(opts.PiAgentDir, "extensions", "gomemory-sidebar.ts")); err != nil {
		t.Fatal("gomemory-sidebar.ts no se instala")
	}
}
