package installer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRound3PrereleaseThroughCommand(t *testing.T) {
	current := versionFromCmd("sh", "-c", "printf 'tool 1.0.0-rc.1\\n'")
	v, err := current(context.Background())
	if err != nil || v != "1.0.0-rc.1" {
		t.Fatalf("reader version=%q error=%v (esperado 1.0.0-rc.1)", v, err)
	}
	cache := hcd042Cache(t, DepState{Name: "synthetic", Current: "1.0.0", Latest: "1.0.0"})
	var out, errs bytes.Buffer
	rc := RunDepsUpdate(cache, []DepProbe{{Name: "synthetic", Current: current}}, &out, &errs)
	if rc == 0 || strings.Contains(out.String(), "están al día") {
		t.Fatalf("prerelease reported healthy: rc=%d out=%q err=%q", rc, out.String(), errs.String())
	}

	latest := latestFromCmd(regexp.MustCompile(`Disponible:\s*v?(\d+\.\d+\.\d+[0-9A-Za-z.+-]*)`), "sh", "-c", "printf 'Disponible: v2.0.0-rc.1\\n'")
	lv, err := latest(context.Background())
	if err != nil || lv != "2.0.0-rc.1" {
		t.Fatalf("latest reader version=%q error=%v (esperado 2.0.0-rc.1)", lv, err)
	}
}

func TestRound3SelfReplacement(t *testing.T) {
	for _, kind := range []string{"regular", "symlink", "directory"} {
		t.Run(kind, func(t *testing.T) {
			opts := sandbox(t)
			file := filepath.Join(opts.LocalBinDir, "pi-gomemory-setup")
			writeFile(t, file, "SYNTH-ORIGINAL")
			old := selfExecutable
			selfExecutable = func() (string, error) { return file, nil }
			t.Cleanup(func() { selfExecutable = old })
			p := &UninstallPlan{}
			defer p.Close()
			p.planSelf(nil, opts)
			if len(p.Actions) != 1 || p.Actions[0].apply == nil {
				t.Fatalf("plan: %+v", p.Actions)
			}
			if err := os.Remove(file); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "regular":
				writeFile(t, file, "SYNTH-NEW")
			case "symlink":
				target := filepath.Join(opts.LocalBinDir, "personal")
				writeFile(t, target, "SYNTH-PERSONAL")
				if err := os.Symlink(target, file); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(file, 0700); err != nil {
					t.Fatal(err)
				}
			}
			err := p.Actions[0].apply()
			if _, statErr := os.Lstat(file); statErr != nil {
				t.Fatalf("replacement removed: action=%v lstat=%v", err, statErr)
			}
		})
	}

	t.Run("original_regular_removed", func(t *testing.T) {
		opts := sandbox(t)
		file := filepath.Join(opts.LocalBinDir, "pi-gomemory-setup")
		writeFile(t, file, "SYNTH-ORIGINAL")
		old := selfExecutable
		selfExecutable = func() (string, error) { return file, nil }
		t.Cleanup(func() { selfExecutable = old })
		p := &UninstallPlan{}
		defer p.Close()
		p.planSelf(nil, opts)
		if len(p.Actions) != 1 || p.Actions[0].apply == nil {
			t.Fatalf("plan: %+v", p.Actions)
		}
		if err := p.Actions[0].apply(); err != nil {
			t.Fatalf("apply original regular: %v", err)
		}
		if _, err := os.Lstat(file); !os.IsNotExist(err) {
			t.Fatalf("original regular still exists: %v", err)
		}
	})
}

// Cada bloqueo se ejecuta en un proceso/grupo aislado, terminado por el padre.
func TestRound3ProbeChild(t *testing.T) {
	mode := os.Getenv("ROUND3_CASE")
	if mode == "" {
		t.Skip("helper subprocess only")
	}
	home := os.Getenv("ROUND3_HOME")
	opts := &Options{HomeDir: home, PiAgentDir: filepath.Join(home, ".pi", "agent"), LocalBinDir: filepath.Join(home, ".local", "bin")}
	fmt.Println("entered synthetic probe")
	if mode == "fifo" {
		if err := Uninstall(repoAssets(t), opts, UninstallFlags{DryRun: true}, os.Stdout); err != nil {
			t.Log(err)
		}
		return
	}
	items := RunDoctor(opts)
	foundWarn := false
	for _, it := range items {
		if it.Name == "MCP en Pi" && it.Status == "WARN" {
			foundWarn = true
		}
	}
	if !foundWarn {
		t.Fatalf("expected MCP en Pi WARN, got %+v", items)
	}
}

func TestRound3ManifestFIFO(t *testing.T) {
	home := t.TempDir()
	path := ManifestPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	round3BoundedProbe(t, "fifo", home, "", 3*time.Second)
}

func TestRound3DoctorDescendant(t *testing.T) {
	home := t.TempDir()
	bin := t.TempDir()
	// El shell directo muere a los 10s de doctor, el sleep conserva las tuberías.
	writeExec(t, filepath.Join(bin, "pi"), "/bin/sleep 30 &\nwait\n")
	round3BoundedProbe(t, "doctor", home, bin, 15*time.Second)
}

func round3BoundedProbe(t *testing.T, mode, home, bin string, budget time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRound3ProbeChild$", "-test.count=1")
	cmd.Env = append(os.Environ(), "ROUND3_CASE="+mode, "ROUND3_HOME="+home, "HOME="+home)
	if bin != "" {
		cmd.Env = append(cmd.Env, "PATH="+bin+":"+os.Getenv("PATH"))
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 200 * time.Millisecond
	started := time.Now()
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(started)
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	if ctx.Err() != nil {
		t.Fatalf("%s exceeded safe budget %s (elapsed=%s): %v output=%q", mode, budget, elapsed, err, out)
	}
	if err != nil {
		t.Fatalf("probe failed: %v output=%q", err, out)
	}
}
