package installer

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// HCD-077 · C-004: ForceRefreshDeps consulta Latest aunque Current falle
// (herramienta ausente), para que el selector muestre la última versión y la
// caché no quede incompleta tras instalar.
func TestHCD077RefreshQueriesLatestWhenMissing(t *testing.T) {
	probes := []DepProbe{
		{
			Name:    "herdr",
			Current: func(context.Context) (string, error) { return "", errors.New("no está en PATH") },
			Latest:  func(context.Context) (string, error) { return "v0.9.5", nil },
		},
	}
	states := ForceRefreshDeps(probes, "", time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC))
	if len(states) != 1 {
		t.Fatalf("len(states) = %d, quiero 1", len(states))
	}
	st := states[0]
	if st.Latest != "v0.9.5" {
		t.Fatalf("Latest = %q, quiero v0.9.5 (state=%+v)", st.Latest, st)
	}
	if st.Current != "" {
		t.Fatalf("Current = %q, quiero vacío", st.Current)
	}
	if st.Outdated() {
		t.Fatalf("herramienta ausente no debe marcarse Outdated: %+v", st)
	}
	if st.Error == "" {
		t.Fatalf("Error debe conservar el fallo de Current: %+v", st)
	}
}

// HCD-077 · C-005: cuando falla la instalación de un tool obligatorio ausente,
// Sync incluye la entrada con Action="failed" antes de devolver el error fatal.
func TestHCD077SyncJSONIncludesFatalRequiredFailure(t *testing.T) {
	home := t.TempDir()
	opts := &Options{
		HomeDir:     home,
		PiAgentDir:  filepath.Join(home, ".pi", "agent"),
		LocalBinDir: filepath.Join(home, ".local", "bin"),
	}
	oldTools := installTools
	installTools = productionTools()
	t.Cleanup(func() { installTools = oldTools })
	withNode(t, "", false)
	withSyncProbes(t, offlineProbes)
	t.Setenv(skipDepsEnv, "")
	t.Setenv("PATH", t.TempDir())

	var out bytes.Buffer
	results, err := Sync(opts, SyncOptions{Yes: true, JSON: true}, nil, &out)
	if err == nil {
		t.Fatal("esperaba error fatal al fallar instalación de pi")
	}
	if len(results) == 0 {
		t.Fatalf("results vacío tras fallo fatal de pi; sync --json omitió el componente fallido")
	}
	var found *SyncResult
	for i := range results {
		if results[i].Name == "pi" {
			found = &results[i]
			break
		}
	}
	if found == nil || found.Action != "failed" || found.Error == "" {
		t.Fatalf("esperaba entrada failed para pi en results, obtuve: %+v", results)
	}
}

// HCD-077 · C-002: ParseSyncArgs rechaza --only= y --skip= vacíos (o solo
// comas/espacios) devolviendo SyncUsageError (código 2).
func TestHCD077ParseSyncArgsRejectsEmptyFilter(t *testing.T) {
	cases := [][]string{
		{"--only="},
		{"--only= , , "},
		{"--skip="},
		{"--skip= , "},
	}
	for _, args := range cases {
		_, err := ParseSyncArgs(args)
		var usageErr *SyncUsageError
		if !errors.As(err, &usageErr) {
			t.Errorf("ParseSyncArgs(%v): err=%v, quiero *SyncUsageError", args, err)
		}
	}
}
