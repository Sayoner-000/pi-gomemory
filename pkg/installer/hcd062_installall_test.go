package installer

import (
	"os"
	"strings"
	"testing"
)

// HCD-062 T6: el paso de herramientas es el primero de InstallAll y, si una
// obligatoria falla, aborta antes de tocar nada (ni la marca de desinstalado).
func TestHCD062InstallAllRunsToolsFirst(t *testing.T) {
	opts := toolSandbox(t)
	srv, hits := scriptServer(t, map[string]string{"/pi.sh": "exit 9\n"})
	withToolHooks(t, testHooks(srv))
	withTools(t, []Tool{{Name: "pi", Required: true, Script: srv.URL + "/pi.sh", Interp: "sh", Manual: "manual-pi"}})
	writeFile(t, UninstalledMarkerPath(opts.HomeDir), "x")

	err := InstallAll(repoAssets(t), opts)
	if err == nil || !strings.Contains(err.Error(), "manual-pi") {
		t.Fatalf("InstallAll debe abortar por pi: %v", err)
	}
	if len(*hits) != 1 {
		t.Fatalf("descargas = %v", *hits)
	}
	if _, err := os.Stat(UninstalledMarkerPath(opts.HomeDir)); err != nil {
		t.Fatalf("la marca de desinstalado debe seguir si se aborta: %v", err)
	}
	if _, err := os.Stat(opts.PiAgentDir); err == nil {
		t.Fatalf("no debe configurarse Pi si falta pi: existe %s", opts.PiAgentDir)
	}
}

// Con todas presentes, InstallAll sigue y retira la marca como antes.
func TestHCD062InstallAllContinuesWhenPresent(t *testing.T) {
	opts := toolSandbox(t)
	withTools(t, []Tool{{Name: "pi", Required: true, Script: "https://invalid.example/pi.sh", Interp: "sh", Manual: "m"}})
	fakeTool(t, opts.LocalBinDir, "pi")
	writeFile(t, UninstalledMarkerPath(opts.HomeDir), "x")
	if err := InstallAll(repoAssets(t), opts); err != nil {
		t.Fatalf("InstallAll: %v", err)
	}
	if _, err := os.Stat(UninstalledMarkerPath(opts.HomeDir)); !os.IsNotExist(err) {
		t.Fatalf("la marca debe retirarse: %v", err)
	}
}
