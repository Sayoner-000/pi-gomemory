package installer

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixed(v string) func(context.Context) (string, error) {
	return func(context.Context) (string, error) { return v, nil }
}

func failing(msg string) func(context.Context) (string, error) {
	return func(context.Context) (string, error) { return "", errors.New(msg) }
}

func TestRefreshDeps(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "deps.json")
	probes := []DepProbe{
		{Name: "a", Current: fixed("1.0.0"), Latest: fixed("v1.1.0"), Update: []string{"a", "up"}},
		{Name: "b", Current: fixed("2.0.0"), Latest: failing("sin red")},
		{Name: "c", Current: failing("c no está en PATH"), Latest: fixed("9.9.9")},
	}
	states := RefreshDeps(probes, cache, time.Now())
	if len(states) != 3 || states[0].Latest != "v1.1.0" || states[1].Error != "sin red" || states[2].Error == "" {
		t.Fatalf("states = %+v", states)
	}
	if !strings.Contains(readFile(t, cache), `"name": "a"`) {
		t.Fatal("no se guardó la caché")
	}
}

func TestDepsStatus(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "deps.json")
	RefreshDeps([]DepProbe{
		{Name: "a", Current: fixed("1.0.0"), Latest: fixed("1.0.0")},
		{Name: "b", Current: fixed("1.0.0"), Latest: fixed("1.2.0"), Update: []string{"b", "update"}},
		{Name: "pi-gomemory", Current: fixed("dev"), Latest: fixed("2.0.0")},
		{Name: "c", Current: fixed("1.0.0"), Latest: failing("x")},
	}, cache, time.Now())
	out := DepsStatus(cache)
	if len(out) != 1 || out[0].Name != "b" {
		t.Fatalf("outdated = %+v", out)
	}
	if n := FormatDepsNotice(out); !strings.Contains(n, "b update") || !strings.Contains(n, "1.0.0 → 1.2.0") {
		t.Fatalf("aviso = %q", n)
	}
	if DepsStatus(filepath.Join(t.TempDir(), "no")) != nil || FormatDepsNotice(nil) != "" {
		t.Fatal("sin caché no debe haber aviso")
	}
}

func TestHarnessDepsCheck(t *testing.T) {
	s := GenerateHarnessScript()
	for _, want := range []string{
		"pi-gomemory-setup deps-refresh",
		"pi-gomemory-setup deps-status; then",
		"pi-gomemory-setup deps-update",
		"PI_GOMEMORY_NO_UPDATE_CHECK",
		"read -r -t 10",
		"[ -t 0 ]",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("el launcher no contiene %q", want)
		}
	}
	if strings.Contains(s, "check-update") {
		t.Error("el launcher conserva la comprobación antigua de solo pi-gomemory")
	}
}
