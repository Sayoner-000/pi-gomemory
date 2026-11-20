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

// ADM 2026-11-20, H-01: el texto final de Bash en Pi 1.5.2 añade, tras la
// salida, "\n\n[Showing lines a-b of N. Full output: …]" si se truncó y
// "\n\nCommand exited with code N" si falló (core/tools/bash.js, formatOutput y
// appendStatus). Los updates en vivo no los traen: si end() cuenta esas líneas
// al calcular la primera línea absoluta, la reconciliación se corre y salta las
// líneas reales posteriores al último update.
func TestADM20261120BashFinalSuffix(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { createTDDStore, createBashLiveIngest } from "./gomemory-tdd.ts";
const fails = [];
const upd = (lines, tr) => ({ content: [{ type: "text", text: lines.join("\n") }], details: tr ? { truncation: tr } : undefined });
const suite = (s) => s.getState().suites.find((x) => x.id === "backend");
const test = (s, n) => suite(s)?.tests.find((x) => x.name === n);
const bashError = (s) => Boolean(suite(s)?.tests.some((x) => x.name.includes("error de ejecución")));
// Formato real del resultado final de Bash en Pi 1.5.2 (datos sintéticos).
const notice = (full, n) => "\n\n[Showing lines " + (full.length - n + 1) + "-" + full.length + " of " + full.length + ". Full output: /tmp/pi-bash-SYNTH-0001.log]";
const exited = (code) => "\n\nCommand exited with code " + code;
const pad = Array.from({ length: 3000 }, (_, i) => "noise " + i);
const N = 2000;

// 1) Truncado y con error: aviso + estado. TestB llegó tras el último update.
{
	const live = [...pad, "=== RUN   TestA", "--- PASS: TestA (0.00s)", "=== RUN   TestB"];
	const full = [...live, "    b_test.go:7: boom", "--- FAIL: TestB (0.01s)", "FAIL", "FAIL\tdemo\t0.02s", ""];
	const store = createTDDStore();
	const ing = createBashLiveIngest(store, process.cwd());
	ing.start("a", "go test -v ./...");
	ing.update("a", upd(live.slice(-N), { truncated: true, totalLines: live.length, outputLines: N }));
	ing.end("a", true, full.slice(-N).join("\n") + notice(full, N) + exited(1), { truncated: true, totalLines: full.length, outputLines: N });
	if (test(store, "TestB")?.status !== "failed") fails.push("1: TestB debió fallar con su salida real: " + test(store, "TestB")?.status);
	if (bashError(store)) fails.push("1: no debe sustituirse por bash-error");
}

// 2) Truncado y sin error: solo el aviso. TestC pasó tras el último update.
{
	const live = [...pad, "=== RUN   TestA", "--- PASS: TestA (0.00s)", "=== RUN   TestC"];
	const full = [...live, "--- PASS: TestC (0.01s)", "PASS", "ok  \tdemo\t0.02s", ""];
	const store = createTDDStore();
	const ing = createBashLiveIngest(store, process.cwd());
	ing.start("b", "go test -v ./...");
	ing.update("b", upd(live.slice(-N), { truncated: true, totalLines: live.length, outputLines: N }));
	ing.end("b", false, full.slice(-N).join("\n") + notice(full, N), { truncated: true, totalLines: full.length, outputLines: N });
	if (test(store, "TestC")?.status !== "passed") fails.push("2: TestC debió pasar: " + test(store, "TestC")?.status);
	if (suite(store)?.status === "failed") fails.push("2: la suite no debe fallar por el aviso de truncado");
}

// 3) Sin truncar y con error: el estado de salida no es una línea de test.
{
	const live = ["=== RUN   TestA", "--- PASS: TestA (0.00s)", "=== RUN   TestB"];
	const full = [...live, "    b_test.go:7: boom", "--- FAIL: TestB (0.01s)", "FAIL", "FAIL\tdemo\t0.02s", ""];
	const store = createTDDStore();
	const ing = createBashLiveIngest(store, process.cwd());
	ing.start("c", "go test -v ./...");
	ing.update("c", upd(live));
	ing.end("c", true, full.join("\n") + exited(1));
	if (test(store, "TestB")?.status !== "failed") fails.push("3: TestB debió fallar: " + test(store, "TestB")?.status);
	if (bashError(store)) fails.push("3: no debe sustituirse por bash-error");
	if (suite(store)?.partial) fails.push("3: sin truncado la suite no es parcial");
}
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") || strings.Contains(out, "FAIL ") {
		t.Fatalf("TestADM20261120BashFinalSuffix: %s", out)
	}
}

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
