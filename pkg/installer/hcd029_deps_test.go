package installer

import (
	"context"
	"errors"
	"testing"
)

// HCD-029: el estado no anuncia una actualización que ya está instalada. La
// versión instalada se relee del probe; la caché solo aporta la última publicada.
func TestHCD029DepsStatusUsesLiveVersion(t *testing.T) {
	cache := writeDepsCache(t, `{"deps":[
 {"name":"pi-gomemory","current":"v1.6.2","latest":"v1.9.0"},
 {"name":"mem","current":"2.54.0","latest":"2.54.1"},
 {"name":"roto","current":"1.0.0","latest":"2.0.0"},
 {"name":"sin-probe-current","current":"1.0.0","latest":"2.0.0"}]}`)
	live := func(v string) func(context.Context) (string, error) {
		return func(context.Context) (string, error) { return v, nil }
	}
	probes := []DepProbe{
		{Name: "pi-gomemory", Current: live("v1.12.1"), Update: []string{"true"}},
		{Name: "mem", Current: live("2.54.0"), Update: []string{"true"}},
		{Name: "roto", Current: func(context.Context) (string, error) { return "", errors.New("no instalado") }, Update: []string{"true"}},
		{Name: "sin-probe-current", Update: []string{"true"}},
	}

	got := map[string]DepState{}
	for _, d := range DepsStatusFor(cache, probes) {
		got[d.Name] = d
	}

	if d, ok := got["pi-gomemory"]; ok {
		t.Errorf("pi-gomemory ya está en v1.12.1 y se anuncia %s → %s", d.Current, d.Latest)
	}
	if d, ok := got["mem"]; !ok || d.Current != "2.54.0" {
		t.Errorf("mem sigue desactualizado y debe listarse con la versión real: %+v", d)
	}
	// Si la versión real no se puede leer, se conserva lo que dice la caché.
	if _, ok := got["roto"]; !ok {
		t.Error("roto: sin lectura real se conserva el aviso de la caché")
	}
	if _, ok := got["sin-probe-current"]; !ok {
		t.Error("sin-probe-current: sin Current se conserva el aviso de la caché")
	}
}
