package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// HCD-062 T7 · C-08: la sonda de codebase-memory-mcp se actualiza con
// deps-install y el aviso muestra ese comando, no una URL.
func TestHCD062DepsInstallProbeAndNotice(t *testing.T) {
	var cbm DepProbe
	for _, p := range DefaultDepProbes("v1.0.0", t.TempDir()) {
		if p.Name == "codebase-memory-mcp" {
			cbm = p
		}
	}
	if strings.Join(cbm.Update, " ") != "pi-gomemory-setup deps-install codebase-memory-mcp" || cbm.Manual != "" {
		t.Fatalf("sonda = %+v", cbm)
	}
	notice := FormatDepsNotice([]DepState{{Name: cbm.Name, Current: "0.11.0", Latest: "0.12.1", Update: cbm.Update}})
	if !strings.Contains(notice, "pi-gomemory-setup deps-install codebase-memory-mcp") || strings.Contains(notice, "http") {
		t.Fatalf("aviso = %q", notice)
	}
}

// UpdateDeps resuelve pi-gomemory-setup por su propia ruta, no por el PATH.
func TestHCD062DepsInstallUpdateUsesExecutable(t *testing.T) {
	dir := t.TempDir()
	self := filepath.Join(dir, "self-setup")
	marker := filepath.Join(dir, "ran")
	writeFile(t, self, "#!/bin/sh\necho \"$@\" > "+marker+"\n")
	if err := os.Chmod(self, 0o755); err != nil {
		t.Fatal(err)
	}
	old := selfExecutable
	selfExecutable = func() (string, error) { return self, nil }
	t.Cleanup(func() { selfExecutable = old })
	t.Setenv("PATH", "/usr/bin:/bin")
	failed := UpdateDeps([]DepState{{Name: "codebase-memory-mcp", Current: "1.0.0", Latest: "2.0.0", Update: []string{"pi-gomemory-setup", "deps-install", "codebase-memory-mcp"}}}, &bytes.Buffer{}, &bytes.Buffer{})
	if len(failed) != 0 {
		t.Fatalf("falló: %v", failed)
	}
	data, err := os.ReadFile(marker)
	if err != nil || strings.TrimSpace(string(data)) != "deps-install codebase-memory-mcp" {
		t.Fatalf("args = %q %v", data, err)
	}
}

// C-13: solo admite codebase-memory-mcp.
func TestHCD062DepsInstallRejectsOtherNames(t *testing.T) {
	opts := toolSandbox(t)
	srv, hits := scriptServer(t, map[string]string{})
	withToolHooks(t, testHooks(srv))
	for _, args := range [][]string{nil, {"pi"}, {"codegraph"}, {"codebase-memory-mcp", "extra"}} {
		var stderr bytes.Buffer
		if code := RunDepsInstall(opts, args, &bytes.Buffer{}, &stderr); code != 2 || !strings.Contains(stderr.String(), "Uso:") {
			t.Fatalf("args %v: code %d, stderr %q", args, code, stderr.String())
		}
	}
	if len(*hits) != 0 {
		t.Fatalf("no debe descargar nada: %v", *hits)
	}
}

// M-2 (cuarta revisión): el destino es el directorio de binarios del
// manifiesto, aunque PI_GOMEMORY_BIN_DIR no esté puesta.
func TestHCD062DepsInstallUsesManifestBinDir(t *testing.T) {
	opts := toolSandbox(t)
	custom := filepath.Join(opts.HomeDir, "mis-bin")
	installOpts := *opts
	installOpts.LocalBinDir = custom
	if err := WriteManifest(&installOpts, "v1.0.0"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PI_GOMEMORY_BIN_DIR", "")
	asset, tgz, sums := cbmRelease(t, "v0.12.1", cbmFakeBin, nil)
	srv, _ := cbmServer(t, "v0.12.1", asset, tgz, sums)
	h := testHooks(srv)
	h.CBMReleases = srv.URL + "/releases"
	withToolHooks(t, h)
	var stdout, stderr bytes.Buffer
	if code := RunDepsInstall(opts, []string{"codebase-memory-mcp"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code %d: %s %s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(custom, "codebase-memory-mcp")); err != nil {
		t.Fatalf("debe instalarse en el bin_dir del manifiesto: %v", err)
	}
}
