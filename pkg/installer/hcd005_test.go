package installer

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestPinnedPackages(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // sin pi: se usa el registro declarativo
	opts := sandbox(t)
	settings := filepath.Join(opts.PiAgentDir, "settings.json")
	writeFile(t, settings, `{"theme":"x","packages":["npm:pi-cc-extensions@0.9.0","npm:otro",{"source":"git:x"}]}`)
	for i := 0; i < 2; i++ {
		if err := EnsurePinnedPackages(opts.PiAgentDir); err != nil {
			t.Fatal(err)
		}
	}
	got := readFile(t, settings)
	for _, want := range []string{`"npm:pi-cc-extensions@` + CCExtensionsVersion + `"`, `"npm:otro"`, `"git:x"`, `"theme": "x"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("falta %s en %s", want, got)
		}
	}
	if strings.Count(got, "pi-cc-extensions") != 1 {
		t.Fatalf("entradas duplicadas: %s", got)
	}
	// HCD-085: pi-antigravity ya no se fija ni se añade.
	if strings.Contains(got, "pi-antigravity") {
		t.Fatalf("EnsurePinnedPackages no debe añadir pi-antigravity: %s", got)
	}
	if !strings.Contains(CCExtensionsSource, "@"+CCExtensionsVersion) {
		t.Fatal("las fuentes deben llevar versión exacta")
	}
}

func TestPinnedDepProbes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	seen := false
	for _, p := range DefaultDepProbes("v1.0.0", dir) {
		if p.Name == "pi-antigravity" {
			t.Fatal("HCD-085: pi-antigravity ya no es un componente del meta-harness")
		}
		if p.Name != "pi-cc-extensions" {
			continue
		}
		seen = true
		latest, err := p.Latest(context.Background())
		if err != nil || latest != CCExtensionsVersion {
			t.Fatalf("%s latest = %q %v, want versión fijada %s", p.Name, latest, err, CCExtensionsVersion)
		}
		if strings.Join(p.Update, " ") != "pi install npm:"+p.Name+"@"+CCExtensionsVersion {
			t.Fatalf("%s update = %v", p.Name, p.Update)
		}
	}
	if !seen {
		t.Fatal("falta la sonda de pi-cc-extensions")
	}
}
