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
	writeFile(t, settings, `{"theme":"x","packages":["npm:pi-antigravity","npm:pi-cc-extensions@0.9.0","npm:otro",{"source":"git:x"}]}`)
	for i := 0; i < 2; i++ {
		if err := EnsurePinnedPackages(opts.PiAgentDir); err != nil {
			t.Fatal(err)
		}
	}
	got := readFile(t, settings)
	for _, want := range []string{`"npm:pi-antigravity@` + AntigravityVersion + `"`, `"npm:pi-cc-extensions@` + CCExtensionsVersion + `"`, `"npm:otro"`, `"git:x"`, `"theme": "x"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("falta %s en %s", want, got)
		}
	}
	if strings.Count(got, "pi-antigravity") != 1 || strings.Count(got, "pi-cc-extensions") != 1 {
		t.Fatalf("entradas duplicadas: %s", got)
	}
	if !strings.Contains(AntigravityPackageSource, "@"+AntigravityVersion) || !strings.Contains(CCExtensionsSource, "@"+CCExtensionsVersion) {
		t.Fatal("las fuentes deben llevar versión exacta")
	}
}

func TestPinnedDepProbes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "npm", "node_modules", "pi-antigravity", "package.json"), `{"version":"0.11.0"}`)
	for _, p := range DefaultDepProbes("v1.0.0", dir) {
		if p.Name != "pi-antigravity" && p.Name != "pi-cc-extensions" {
			continue
		}
		latest, err := p.Latest(context.Background())
		want := AntigravityVersion
		if p.Name == "pi-cc-extensions" {
			want = CCExtensionsVersion
		}
		if err != nil || latest != want {
			t.Fatalf("%s latest = %q %v, want versión fijada %s", p.Name, latest, err, want)
		}
		if strings.Join(p.Update, " ") != "pi install npm:"+p.Name+"@"+want {
			t.Fatalf("%s update = %v", p.Name, p.Update)
		}
	}
}
