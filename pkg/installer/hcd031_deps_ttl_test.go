package installer

import (
	"context"
	"testing"
	"time"
)

// HCD-031: una release publicada después de la última consulta se descubre
// como mucho una hora más tarde. Con menos de una hora se reutiliza la caché.
func TestHCD031DepsCacheTTL(t *testing.T) {
	checked := time.Date(2026, 11, 13, 6, 5, 0, 0, time.UTC)
	calls := 0
	probes := []DepProbe{{
		Name:    "pi-gomemory",
		Current: func(context.Context) (string, error) { return "v1.12.1", nil },
		Latest:  func(context.Context) (string, error) { calls++; return "v1.13.0", nil },
	}}
	seed := func(t *testing.T) string {
		return writeDepsCache(t, `{"checked_at":"`+checked.Format(time.RFC3339)+`","deps":[{"name":"pi-gomemory","current":"v1.12.1","latest":"v1.12.1"}]}`)
	}

	cache := seed(t)
	if st := RefreshDeps(probes, cache, checked.Add(30*time.Minute)); calls != 0 || st[0].Latest != "v1.12.1" {
		t.Fatalf("a los 30 min debe reutilizar la caché: llamadas=%d, latest=%s", calls, st[0].Latest)
	}

	cache = seed(t)
	st := RefreshDeps(probes, cache, checked.Add(61*time.Minute))
	if calls != 1 || st[0].Latest != "v1.13.0" {
		t.Fatalf("pasada una hora debe volver a la red: llamadas=%d, latest=%s", calls, st[0].Latest)
	}
	if got := DepsStatusFor(cache, probes); len(got) != 1 || got[0].Latest != "v1.13.0" {
		t.Fatalf("deps-status debe anunciar v1.13.0: %+v", got)
	}
}
