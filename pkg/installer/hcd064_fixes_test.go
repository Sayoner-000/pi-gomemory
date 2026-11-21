package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// HCD-064 T1 (H-01 / C-002): al recibir SIGTERM mientras corre un instalador
// sin TTY, el manejador de señales debe matar al grupo de procesos hijo,
// borrar los temporales y salir con código 143 (128+SIGTERM).
func TestHCD064SignalKillsChildGroup(t *testing.T) {
	opts := toolSandbox(t)
	oldTTY := hooks.TTY
	hooks.TTY = func() bool { return false }
	t.Cleanup(func() { hooks.TTY = oldTTY })

	got := make(chan int, 1)
	oldExit := signalExit
	signalExit = func(code int) { got <- code }
	t.Cleanup(func() { signalExit = oldExit })

	guard := make(chan os.Signal, 1)
	signal.Notify(guard, syscall.SIGTERM)
	defer signal.Stop(guard)

	dir, cleanup, err := depsTempDir(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	pidFile := filepath.Join(opts.HomeDir, "child.pid")
	var out bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- runInstaller("sh", []string{"-c", "echo $$ > '" + pidFile + "'; sleep 30"}, nil, dir, &out, 30*time.Second)
	}()

	var childPID int
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(pidFile); err == nil {
			if p, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil && p > 0 {
				childPID = p
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if childPID == 0 {
		t.Fatal("el instalador hijo no arrancó a tiempo")
	}
	t.Cleanup(func() {
		_ = syscall.Kill(-childPID, syscall.SIGKILL)
		_ = syscall.Kill(childPID, syscall.SIGKILL)
	})

	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	select {
	case code := <-got:
		if code != 143 {
			t.Errorf("código de salida en SIGTERM = %d, se esperaba 143", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("SIGTERM no se capturó")
	}

	// El hijo (y su grupo) debe haber terminado tras la señal.
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Errorf("el instalador hijo (pid %d) siguió vivo tras SIGTERM", childPID)
	}
	if err := syscall.Kill(-childPID, 0); !errors.Is(err, syscall.ESRCH) {
		t.Errorf("el grupo de procesos -%d sigue vivo tras SIGTERM: %v", childPID, err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("el temporal %s debe borrarse: %v", dir, err)
	}
}

// HCD-064 T2 (H-02 / C-003): cancelar Uninstall o fallar por falta de TTY no
// debe borrar los temporales huérfanos antes de obtener confirmación.
func TestHCD064UninstallCancelPreservesTemps(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		opts := installSandbox(t)
		stale := filepath.Join(opts.LocalBinDir, depsTempPrefix+"orphan-cancel")
		if err := os.MkdirAll(stale, 0o700); err != nil {
			t.Fatal(err)
		}
		oldTTY := stdinIsTerminal
		oldIn := confirmInput
		stdinIsTerminal = func() bool { return true }
		confirmInput = strings.NewReader("n\n")
		t.Cleanup(func() {
			stdinIsTerminal = oldTTY
			confirmInput = oldIn
		})

		var out bytes.Buffer
		err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: false}, &out)
		if err == nil || !strings.Contains(err.Error(), "cancelada") {
			t.Fatalf("se esperaba cancelación, obtenido: %v\n%s", err, out.String())
		}
		if _, statErr := os.Stat(stale); statErr != nil {
			t.Fatalf("el temporal %s se borró pese a cancelar: %v", stale, statErr)
		}
	})

	t.Run("no_tty", func(t *testing.T) {
		opts := installSandbox(t)
		stale := filepath.Join(opts.LocalBinDir, depsTempPrefix+"orphan-notty")
		if err := os.MkdirAll(stale, 0o700); err != nil {
			t.Fatal(err)
		}
		oldTTY := stdinIsTerminal
		stdinIsTerminal = func() bool { return false }
		t.Cleanup(func() { stdinIsTerminal = oldTTY })

		var out bytes.Buffer
		err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: false}, &out)
		if err == nil || !strings.Contains(err.Error(), "terminal interactiva") {
			t.Fatalf("se esperaba rechazo sin TTY, obtenido: %v\n%s", err, out.String())
		}
		if _, statErr := os.Stat(stale); statErr != nil {
			t.Fatalf("el temporal %s se borró sin confirmación ni TTY: %v", stale, statErr)
		}
	})
}

// HCD-064 T3 (H-03 / C-001): depsTempDir escribe .pid y RemoveStaleDepsTemps
// respeta los temporales cuyo PID propietario sigue vivo en otro proceso.
func TestHCD064RemoveStaleRespectsLivePID(t *testing.T) {
	opts := toolSandbox(t)
	dir, cleanup, err := depsTempDir(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	pidBytes, err := os.ReadFile(filepath.Join(dir, ".pid"))
	if err != nil {
		t.Fatalf("depsTempDir debe escribir .pid: %v", err)
	}
	if strings.TrimSpace(string(pidBytes)) != strconv.Itoa(os.Getpid()) {
		t.Fatalf(".pid = %q, se esperaba %d", string(pidBytes), os.Getpid())
	}

	otherLive := filepath.Join(opts.LocalBinDir, depsTempPrefix+"other-live")
	otherDead := filepath.Join(opts.LocalBinDir, depsTempPrefix+"other-dead")
	if err := os.MkdirAll(otherLive, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherLive, ".pid"), []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(otherDead, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherDead, ".pid"), []byte("99999999\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	RemoveStaleDepsTemps(opts.LocalBinDir)

	if _, err := os.Stat(otherLive); err != nil {
		t.Fatalf("el temporal %s de otro proceso vivo no debía borrarse: %v", otherLive, err)
	}
	if _, err := os.Stat(otherDead); !os.IsNotExist(err) {
		t.Fatalf("el temporal %s de un proceso muerto debía borrarse: %v", otherDead, err)
	}
}

// HCD-064 T4 (H-04 / C-004): PI_GOMEMORY_BIN_DIR relativo se normaliza a ruta
// absoluta y no provoca doble resolución de ruta en installByScript.
func TestHCD064RelativeBinDir(t *testing.T) {
	work := t.TempDir()
	t.Chdir(work)
	t.Setenv("PI_GOMEMORY_BIN_DIR", "relbin")

	defOpts, err := DefaultOptions()
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(defOpts.LocalBinDir) {
		t.Fatalf("DefaultOptions().LocalBinDir debe ser absoluta, obtenido: %q", defOpts.LocalBinDir)
	}

	opts := toolSandbox(t)
	opts.LocalBinDir = "relbin2"
	marker := filepath.Join(work, "ran.txt")
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#!/bin/sh\necho ok > '" + marker + "'\n"))
	}))
	defer srv.Close()
	oldClient := hooks.Client
	hooks.Client = srv.Client()
	t.Cleanup(func() { hooks.Client = oldClient })

	var out bytes.Buffer
	tool := Tool{Name: "fake", Script: srv.URL + "/install.sh", Interp: "sh"}
	if err := installByScript(tool, opts, &out); err != nil {
		t.Fatalf("installByScript falló con LocalBinDir relativo: %v\n%s", err, out.String())
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("el script no se ejecutó: %v", err)
	}
}

// HCD-064 T5 (H-05 / C-005): con TTY activo, al vencer el plazo de
// runInstaller también deben terminar los descendientes (nietos) del hijo.
func TestHCD064TTYTimeoutKillsDescendants(t *testing.T) {
	opts := toolSandbox(t)
	oldTTY := hooks.TTY
	hooks.TTY = func() bool { return true }
	t.Cleanup(func() { hooks.TTY = oldTTY })

	pidFile := filepath.Join(opts.HomeDir, "grandchild.pid")
	var out bytes.Buffer
	err := runInstaller("sh", []string{"-c", "sleep 30 & echo $! > '" + pidFile + "'; wait"}, nil, opts.HomeDir, &out, 400*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "superó el plazo") {
		t.Fatalf("se esperaba timeout, obtenido: %v", err)
	}

	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("el nieto no escribió su PID: %v", err)
	}
	gcPID, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || gcPID <= 0 {
		t.Fatalf("PID de nieto inválido: %q", string(b))
	}
	t.Cleanup(func() { _ = syscall.Kill(gcPID, syscall.SIGKILL) })

	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if errors.Is(syscall.Kill(gcPID, 0), syscall.ESRCH) {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("el proceso nieto (pid %d) siguió vivo tras vencer el timeout con TTY", gcPID)
}

// HCD-064 T6 (H-06 / C-006): extractExact debe aceptar tarballs estándar con
// entradas de directorio ./ o dir/ y nombres benignos con "..", manteniendo el
// rechazo de traversal (../) y enlaces simbólicos.
func TestHCD064ExtractExactDirEntries(t *testing.T) {
	makeTarGz := func(entries []struct {
		hdr  tar.Header
		body []byte
	}) []byte {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		for _, e := range entries {
			h := e.hdr
			h.Size = int64(len(e.body))
			if err := tw.WriteHeader(&h); err != nil {
				t.Fatal(err)
			}
			if len(e.body) > 0 {
				if _, err := tw.Write(e.body); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}

	good := makeTarGz([]struct {
		hdr  tar.Header
		body []byte
	}{
		{hdr: tar.Header{Name: "./", Mode: 0o755, Typeflag: tar.TypeDir}},
		{hdr: tar.Header{Name: "docs/", Mode: 0o755, Typeflag: tar.TypeDir}},
		{hdr: tar.Header{Name: "docs/NOTES..md", Mode: 0o644, Typeflag: tar.TypeReg}, body: []byte("nota")},
		{hdr: tar.Header{Name: "./codebase-memory-mcp", Mode: 0o755, Typeflag: tar.TypeReg}, body: []byte("BIN-OK")},
	})
	got, err := extractExact(good, "codebase-memory-mcp", 1<<20)
	if err != nil {
		t.Fatalf("extractExact rechazó un tarball válido con ./, dir/ y NOTES..md: %v", err)
	}
	if string(got) != "BIN-OK" {
		t.Fatalf("contenido = %q, se esperaba BIN-OK", string(got))
	}

	traversal := makeTarGz([]struct {
		hdr  tar.Header
		body []byte
	}{
		{hdr: tar.Header{Name: "../evil", Mode: 0o644, Typeflag: tar.TypeReg}, body: []byte("x")},
		{hdr: tar.Header{Name: "codebase-memory-mcp", Mode: 0o755, Typeflag: tar.TypeReg}, body: []byte("BIN")},
	})
	if _, err := extractExact(traversal, "codebase-memory-mcp", 1<<20); err == nil {
		t.Fatal("extractExact debe seguir rechazando rutas con segmento ..")
	}

	symlink := makeTarGz([]struct {
		hdr  tar.Header
		body []byte
	}{
		{hdr: tar.Header{Name: "link", Linkname: "/etc/passwd", Mode: 0o777, Typeflag: tar.TypeSymlink}},
		{hdr: tar.Header{Name: "codebase-memory-mcp", Mode: 0o755, Typeflag: tar.TypeReg}, body: []byte("BIN")},
	})
	if _, err := extractExact(symlink, "codebase-memory-mcp", 1<<20); err == nil {
		t.Fatal("extractExact debe seguir rechazando enlaces simbólicos")
	}
}
