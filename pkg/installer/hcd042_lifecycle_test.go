package installer

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
)

// Cuenta referencias reales al inode, incluso después de quitar su nombre.
// No abre el archivo inspeccionado ni fuerza al sistema a reutilizar inodes.
func hcd042OpenReferences(t *testing.T, original os.FileInfo) int {
	t.Helper()
	dir := "/dev/fd"
	if runtime.GOOS == "linux" {
		dir = "/proc/self/fd"
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("no se pueden inspeccionar descriptores: %v", err)
	}
	planned, ok := original.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("FileInfo sin identidad Unix: %T", original.Sys())
	}
	n := 0
	for _, entry := range entries {
		fd, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		// Fstat inspecciona el descriptor real. En macOS Stat(/dev/fd/N)
		// devuelve metadatos de fdescfs, no los del archivo retenido.
		var held syscall.Stat_t
		if syscall.Fstat(fd, &held) == nil && held.Dev == planned.Dev && held.Ino == planned.Ino {
			n++
		}
	}
	return n
}

func hcd042RequireCloser(t *testing.T, p *UninstallPlan) io.Closer {
	t.Helper()
	closer, ok := any(p).(io.Closer)
	if !ok {
		t.Fatal("UninstallPlan debe ofrecer cierre explícito de las referencias retenidas")
	}
	t.Cleanup(func() {
		if err := closer.Close(); err != nil {
			t.Errorf("cerrar plan: %v", err)
		}
	})
	return closer
}

// Observa el plan dentro de Uninstall, antes de ejecutar o salir por otra ruta.
type hcd042InspectWriter struct {
	bytes.Buffer
	once    sync.Once
	inspect func()
}

func (w *hcd042InspectWriter) Write(p []byte) (int, error) {
	w.once.Do(w.inspect)
	return w.Buffer.Write(p)
}

func TestHCD042PlanLifecycle(t *testing.T) {
	for _, kind := range []string{"cache", "manifest"} {
		for _, outcome := range []string{"discard", "success", "gone", "recreated", "remove error"} {
			t.Run(kind+"/"+outcome, func(t *testing.T) {
				if outcome == "remove error" {
					skipIfRoot(t)
				}
				dir := t.TempDir()
				file := filepath.Join(dir, "owned")
				writeFile(t, file, "original")
				original, err := os.Lstat(file)
				if err != nil {
					t.Fatal(err)
				}
				p := hcd042RemovalPlan(t, file, kind)
				closer := hcd042RequireCloser(t, p)
				if n := hcd042OpenReferences(t, original); n != 1 {
					t.Fatalf("debe retener exactamente una referencia al original, hay %d", n)
				}
				if len(p.Actions) != 1 || p.Actions[0].apply == nil {
					t.Fatalf("plan: %+v", p.Actions)
				}
				action := p.Actions[0].apply
				if outcome == "discard" {
					if err := closer.Close(); err != nil {
						t.Fatal(err)
					}
					if err := closer.Close(); err != nil {
						t.Fatalf("Close no idempotente: %v", err)
					}
					if n := hcd042OpenReferences(t, original); n != 0 {
						t.Fatalf("fuga al descartar: %d", n)
					}
					if err := action(); err == nil {
						t.Fatal("un plan descartado no puede ejecutar su retirada")
					}
					if _, err := os.Lstat(file); err != nil {
						t.Fatalf("borró archivo tras Close: %v", err)
					}
					return
				}
				if outcome == "gone" || outcome == "recreated" {
					if err := os.Remove(file); err != nil {
						t.Fatal(err)
					}
					if n := hcd042OpenReferences(t, original); n != 1 {
						t.Fatalf("no retuvo original sin nombre: %d", n)
					}
				}
				if outcome == "recreated" {
					writeFile(t, file, "SYNTH-NEW")
					current, err := os.Lstat(file)
					if err != nil {
						t.Fatal(err)
					}
					if os.SameFile(original, current) {
						t.Fatal("reutilizó inode pese a referencia retenida")
					}
				}
				if outcome == "remove error" {
					chmodFor(t, dir, 0555)
				}
				err = action()
				switch outcome {
				case "success":
					if err != nil {
						t.Fatal(err)
					}
				case "gone":
					if !errors.Is(err, errGoneAfterPlan) {
						t.Fatalf("gone: %v", err)
					}
				case "recreated":
					if !errors.Is(err, errChangedAfterPlan) {
						t.Fatalf("changed: %v", err)
					}
					if data, err := os.ReadFile(file); err != nil || string(data) != "SYNTH-NEW" {
						t.Fatalf("sustituto perdido: %q %v", data, err)
					}
				case "remove error":
					if err == nil {
						t.Fatal("Remove debía fallar")
					}
				}
				if n := hcd042OpenReferences(t, original); n != 0 {
					t.Fatalf("fuga tras acción %s: %d", outcome, n)
				}
				if err := closer.Close(); err != nil {
					t.Fatalf("Close tras acción: %v", err)
				}
			})
		}
	}

	for _, outcome := range []string{"dry-run", "cancel", "no terminal", "success", "execution error", "inspection error"} {
		t.Run("Uninstall/"+outcome, func(t *testing.T) {
			if outcome == "execution error" || outcome == "inspection error" {
				skipIfRoot(t)
			}
			opts := sandbox(t)
			paths := []string{DefaultDepsCachePath(opts.PiAgentDir), DefaultUpdateCachePath(opts.PiAgentDir), ManifestPath(opts.HomeDir)}
			var originals []os.FileInfo
			for _, path := range paths {
				writeFile(t, path, `{"version":"synthetic","files":{}}`)
				info, err := os.Lstat(path)
				if err != nil {
					t.Fatal(err)
				}
				originals = append(originals, info)
			}
			oldTTY, oldIn := stdinIsTerminal, confirmInput
			t.Cleanup(func() { stdinIsTerminal, confirmInput = oldTTY, oldIn })
			stdinIsTerminal = func() bool { return outcome != "no terminal" }
			confirmInput = strings.NewReader("n\n")
			flags := UninstallFlags{Yes: true}
			if outcome == "dry-run" {
				flags.DryRun = true
			}
			if outcome == "cancel" || outcome == "no terminal" {
				flags.Yes = false
			}
			if outcome == "inspection error" {
				chmodFor(t, paths[0], 0000)
			}
			observed := false
			w := &hcd042InspectWriter{inspect: func() {
				observed = true
				for i, info := range originals {
					want := 1
					if outcome == "inspection error" && i == 0 {
						want = 0
					}
					if n := hcd042OpenReferences(t, info); n != want {
						t.Errorf("plan no retiene %s: referencias=%d esperado=%d", paths[i], n, want)
					}
				}
				if outcome == "execution error" {
					chmodFor(t, opts.PiAgentDir, 0555)
				}
			}}
			err := Uninstall(repoAssets(t), opts, flags, w)
			wantError := outcome == "cancel" || outcome == "no terminal" || outcome == "execution error" || outcome == "inspection error"
			if (err != nil) != wantError {
				t.Fatalf("error=%v salida=%s", err, w.String())
			}
			if !observed {
				t.Fatal("no se observó el plan antes de salir")
			}
			for i, info := range originals {
				if n := hcd042OpenReferences(t, info); n != 0 {
					t.Errorf("fuga en %s tras %s: %d", paths[i], outcome, n)
				}
				if outcome == "dry-run" || outcome == "cancel" || outcome == "no terminal" {
					if _, err := os.Lstat(paths[i]); err != nil {
						t.Errorf("entrada perdida al descartar: %v", err)
					}
				}
			}
			if outcome == "execution error" && !manifestExists(t, opts.HomeDir) {
				t.Fatal("se perdió manifiesto condicionado tras fallo")
			}
		})
	}
}
