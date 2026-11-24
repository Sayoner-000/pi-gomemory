package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func hcd081Read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// legacyOK indica si una línea que nombra codebase-memory habla de su retirada
// o de la entrada heredada, que es comportamiento vigente (HCD-081).
func legacyOK(line string) bool {
	l := strings.ToLower(line)
	for _, w := range []string{"v1.28.0", "retir", "heredad", "ya no"} {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

// HCD-081 tarea 1: la ayuda del CLI y /componentes ya no ofrecen codebase-memory.
func TestHCD081HelpText(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	printHelp()
	os.Stdout = stdout
	_ = w.Close()
	help, _ := io.ReadAll(r)
	for name, text := range map[string]string{
		"printHelp":    string(help),
		"codemodeNote": codemodeNote(),
		"/componentes": hcd081Read(t, "extensions/gomemory-status.ts"),
	} {
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(strings.ToLower(line), "codebase-memory") && !legacyOK(line) {
				t.Errorf("%s sigue ofreciendo codebase-memory: %s", name, strings.TrimSpace(line))
			}
		}
	}
	if !strings.Contains(string(help), "skills-sync --check") || !strings.Contains(string(help), "skills-sync --restore") {
		t.Error("printHelp debe documentar skills-sync --check y --restore")
	}
}

// HCD-081 tarea 2: las guías vigentes describen el catálogo actual y las
// novedades de v1.29.0.
func TestHCD081DocsCurrent(t *testing.T) {
	for _, path := range []string{"docs/installation.md", "docs/usage.md", "docs/architecture.md", "docs/known-issues.md"} {
		for i, line := range strings.Split(hcd081Read(t, path), "\n") {
			if strings.Contains(strings.ToLower(line), "codebase-memory") && !legacyOK(line) {
				t.Errorf("%s:%d describe codebase-memory como vigente: %.120s", path, i+1, strings.TrimSpace(line))
			}
			if strings.Contains(line, "21 skills") {
				t.Errorf("%s:%d cuenta 21 skills (son 20)", path, i+1)
			}
		}
	}
	index := hcd081Read(t, "docs/index.md")
	if !strings.Contains(index, "releases/v1.29.0.md") || !strings.Contains(index, "releases/v1.28.0.md") {
		t.Error("docs/index.md debe enlazar las notas de v1.28.0 y v1.29.0")
	}
	if strings.Contains(index, "* [codebase-memory-mcp]") {
		t.Error("docs/index.md sigue listando codebase-memory-mcp entre las herramientas")
	}
	for _, want := range []string{"skills-sync --check", "skills-sync --restore", "stale"} {
		if !strings.Contains(hcd081Read(t, "docs/usage.md"), want) {
			t.Errorf("docs/usage.md no menciona %q", want)
		}
	}
}

// HCD-081 tarea 3: la entrada v1.29.0 del CHANGELOG y sus notas no llevan las
// marcas de texto generado que evita la skill humanizer (rayas, negritas de
// etiqueta) y cubren lo publicado.
func TestHCD081ReleaseNotes(t *testing.T) {
	changelog := hcd081Read(t, "CHANGELOG.md")
	start := strings.Index(changelog, "## [v1.29.0]")
	end := strings.Index(changelog, "## [v1.28.0]")
	if start < 0 || end < start {
		t.Fatal("falta la entrada v1.29.0 antes de v1.28.0")
	}
	notes := hcd081Read(t, "docs/releases/v1.29.0.md")
	for name, text := range map[string]string{"CHANGELOG v1.29.0": changelog[start:end], "notas v1.29.0": notes} {
		for _, bad := range []string{"—", "–", " -- ", "**"} {
			if strings.Contains(text, bad) {
				t.Errorf("%s contiene %q", name, bad)
			}
		}
		for _, want := range []string{"stale", "skills-sync --check", "--restore", "codebase-memory"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s no menciona %q", name, want)
			}
		}
	}
	if _, err := os.Stat("docs/releases/v1.28.0.md"); err != nil {
		t.Error("faltan las notas de v1.28.0")
	}
}
