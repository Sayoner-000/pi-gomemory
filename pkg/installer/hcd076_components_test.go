package installer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeProbes sustituye la red y los binarios de las sondas: versions[name]
// es «actual→última»; un nombre sin entrada falla en Current.
func fakeProbes(versions map[string]string) []DepProbe {
	var out []DepProbe
	for _, p := range DefaultDepProbes("v1.0.0", "") {
		v, ok := versions[p.Name]
		cur, latest, _ := strings.Cut(v, "→")
		p.Published = nil
		p.Current = func(context.Context) (string, error) {
			if !ok {
				return "", errors.New("no está en PATH")
			}
			return cur, nil
		}
		p.Latest = func(context.Context) (string, error) { return latest, nil }
		out = append(out, p)
	}
	return out
}

// T7 · C-16: components --json refresca aunque la caché sea reciente y la
// comprobación esté desactivada, y devuelve todo el catálogo en su orden.
func TestHCD076ComponentsJSON(t *testing.T) {
	opts := toolSandbox(t)
	bin := filepath.Join(opts.HomeDir, "bin")
	for _, n := range []string{"pi", "mem", "codegraph", "codebase-memory-mcp"} {
		fakeTool(t, bin, n)
	}
	t.Setenv("PATH", bin+":/usr/bin:/bin")
	t.Setenv(updateCheckEnv, "1")
	cache := filepath.Join(opts.PiAgentDir, depsCacheFile)
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	stale, _ := json.Marshal(depsCache{CheckedAt: now.Add(-time.Minute), Deps: []DepState{{Name: "pi", Current: "0.0.1", Latest: "0.0.1"}}})
	writeFile(t, cache, string(stale))

	probes := fakeProbes(map[string]string{
		"pi-gomemory":         "v1.0.0→v1.0.0",
		"pi":                  "1.0.0→1.2.0",
		"mem":                 "1.0.0→1.0.0",
		"codegraph":           "1.0.0→1.0.0",
		"codebase-memory-mcp": "1.0.0→1.0.0",
		"pi-antigravity":      "0.15.0→0.15.0",
	})
	var buf strings.Builder
	if err := WriteComponentsJSON(&buf, probes, cache, now); err != nil {
		t.Fatal(err)
	}
	var got []ComponentStatus
	if err := json.Unmarshal([]byte(buf.String()), &got); err != nil {
		t.Fatalf("JSON inválido: %v\n%s", err, buf.String())
	}
	var names []string
	byName := map[string]ComponentStatus{}
	for _, c := range got {
		names = append(names, c.Name)
		byName[c.Name] = c
	}
	if strings.Join(names, ",") != strings.Join(catalogNames(KindSelf, KindTool, KindPackage), ",") {
		t.Fatalf("orden = %v", names)
	}
	pi := byName["pi"]
	if pi.Kind != "tool" || !pi.Required || !pi.Installed || pi.Current != "1.0.0" || pi.Latest != "1.2.0" || !pi.Outdated {
		t.Fatalf("pi = %+v", pi)
	}
	h := byName["herdr"]
	if h.Kind != "tool" || h.Required || h.Installed || h.Current != "" || h.Outdated || h.Error != "" {
		t.Fatalf("herdr ausente = %+v", h)
	}
	if c := byName["pi-cc-extensions"]; c.Kind != "package" || c.Installed {
		t.Fatalf("pi-cc-extensions ausente = %+v", c)
	}
	if s := byName["pi-gomemory"]; s.Kind != "self" || !s.Installed {
		t.Fatalf("pi-gomemory = %+v", s)
	}
	data, _ := os.ReadFile(cache)
	var c depsCache
	_ = json.Unmarshal(data, &c)
	if !c.CheckedAt.Equal(now) || len(c.Deps) != len(Catalog()) {
		t.Fatalf("la caché debe reescribirse completa: %s", data)
	}
}
