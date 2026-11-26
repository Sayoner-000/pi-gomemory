package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ADM 2026-11-20, H-01 cubría la ingesta de Bash de /tdd y se retiró con él en HCD-085.

// ADM 2026-11-20, H-02: si cp, chmod o mv fallan al colocar el binario
// descargado, install.sh no debe dejar ${SETUP_BIN}.tmp.* en INSTALL_DIR.
func TestADM20261120InstallShStagingCleanup(t *testing.T) {
	t.Parallel()
	for _, tool := range []string{"curl", "tar"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatalf("%s es obligatorio para esta prueba", tool)
		}
	}
	root := t.TempDir()

	// Release local sintética servida por file://.
	release := filepath.Join(root, "release")
	pkgDir := filepath.Join(root, "pkg")
	writeExec(t, filepath.Join(pkgDir, "pi-gomemory-setup"), "exit 0")
	osName, arch := hostReleaseTarget(t)
	tarball := "pi-gomemory-setup_vSYNTH_" + osName + "_" + arch + ".tar.gz"
	if err := os.MkdirAll(release, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("tar", "-czf", filepath.Join(release, tarball), "-C", pkgDir, "pi-gomemory-setup").CombinedOutput(); err != nil {
		t.Fatalf("tar: %v\n%s", err, out)
	}
	data, err := os.ReadFile(filepath.Join(release, tarball))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	writeFile(t, filepath.Join(release, "SHA256SUMS"), hex.EncodeToString(sum[:])+"  "+tarball+"\n")

	// chmod falla siempre: el temporal ya se copió a INSTALL_DIR y set -e aborta.
	shim := filepath.Join(root, "shim")
	writeExec(t, filepath.Join(shim, "chmod"), "echo 'chmod sintético: fallo' >&2\nexit 1")

	installDir := filepath.Join(root, "bin")
	cwd := filepath.Join(root, "cwd") // fuera del código fuente: se descarga
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", script)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(),
		"PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+root,
		"PI_GOMEMORY_BIN_DIR="+installDir,
		"PI_GOMEMORY_VERSION=vSYNTH",
		"PI_GOMEMORY_BASE_URL=file://"+release,
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("install.sh debía fallar con chmod roto:\n%s", out)
	}
	if !strings.Contains(string(out), "chmod sintético") {
		t.Fatalf("el fallo debía venir de chmod (la descarga no llegó a colocar el binario):\n%s", out)
	}
	entries, err := os.ReadDir(installDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp.") {
			t.Fatalf("install.sh dejó el temporal %s en INSTALL_DIR", e.Name())
		}
	}
}

// hostReleaseTarget devuelve el OS y la arquitectura con la misma
// normalización que install.sh.
func hostReleaseTarget(t *testing.T) (string, string) {
	t.Helper()
	osOut, err := exec.Command("uname", "-s").Output()
	if err != nil {
		t.Fatal(err)
	}
	archOut, err := exec.Command("uname", "-m").Output()
	if err != nil {
		t.Fatal(err)
	}
	arch := strings.TrimSpace(string(archOut))
	switch arch {
	case "x86_64", "amd64":
		arch = "amd64"
	case "arm64", "aarch64":
		arch = "arm64"
	}
	return strings.ToLower(strings.TrimSpace(string(osOut))), arch
}
