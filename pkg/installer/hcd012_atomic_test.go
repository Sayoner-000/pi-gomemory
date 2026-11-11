package installer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// C-05: el instalador no usa os.WriteFile fuera de tests y uninstall.go.
func TestHCD012AtomicNoWriteFile(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || f == "uninstall.go" {
			continue
		}
		af, err := parser.ParseFile(token.NewFileSet(), f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(af, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "os" && sel.Sel.Name == "WriteFile" {
					t.Errorf("%s usa os.WriteFile", f)
				}
			}
			return true
		})
	}
}

func noTemps(t *testing.T, root string) {
	t.Helper()
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && strings.Contains(d.Name(), ".tmp-") {
			t.Errorf("temporal huérfano: %s", p)
		}
		return nil
	})
}

// C-05: las instalaciones dejan el contenido esperado, el launcher 0755 y sin temporales.
func TestHCD012AtomicInstall(t *testing.T) {
	assets := repoAssets(t)
	opts := sandbox(t)
	if err := InstallHarness(opts.LocalBinDir); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(opts.LocalBinDir, "pi-gomemory")
	if st, err := os.Stat(launcher); err != nil || st.Mode().Perm() != 0755 {
		t.Fatalf("launcher: %v %v", st, err)
	}
	if readFile(t, launcher) != GenerateHarnessScript() {
		t.Error("launcher distinto")
	}
	if err := InstallTheme(assets, opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	if err := InstallTUIExtension(assets, opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	if readFile(t, filepath.Join(opts.PiAgentDir, "extensions", "gomemory-splash.ts")) != readAsset(t, "extensions/gomemory-splash.ts") {
		t.Error("extensión distinta")
	}
	if err := InstallInstructions(opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readFile(t, filepath.Join(opts.PiAgentDir, "AGENTS.md")), "gomemory") {
		t.Error("AGENTS.md sin bloque")
	}
	if _, err := InstallSkills(assets, opts); err != nil {
		t.Fatal(err)
	}
	noTemps(t, opts.HomeDir)
	noTemps(t, opts.LocalBinDir)
	noTemps(t, opts.PiAgentDir)
}
