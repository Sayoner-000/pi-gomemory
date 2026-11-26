package installer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// HCD-069 T1 y T4 cubrían /tdd y se retiraron con él en HCD-085.

// HCD-069 T2 (H-02 / ref C-86): depsTempDir protege el temporal con un bloqueo
// del kernel (.lock vía flock). Si un temporal con .lock queda huérfano y el
// PID guardado en .pid pertenece ahora a otro proceso vivo ajeno (reutilización
// de PID), RemoveStaleDepsTemps debe eliminarlo porque nadie retiene el lock.
func TestHCD069RemoveStaleReusedPIDOrphan(t *testing.T) {
	opts := toolSandbox(t)
	dir, cleanup, err := depsTempDir(opts)
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(dir, ".lock")
	if _, err := os.Stat(lockPath); err != nil {
		cleanup()
		t.Fatalf("depsTempDir debe crear %s: %v", lockPath, err)
	}
	cleanup()

	// Lanzar un proceso ajeno vivo para simular reutilización de PID.
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("no se pudo lanzar sleep auxiliar: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	orphan := filepath.Join(opts.LocalBinDir, depsTempPrefix+"reused-pid")
	if err := os.MkdirAll(orphan, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orphan, ".pid"), []byte(strconv.Itoa(cmd.Process.Pid)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orphan, ".lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	RemoveStaleDepsTemps(opts.LocalBinDir)

	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("el huérfano %s con .lock sin propietario debía eliminarse aunque el PID %d siga vivo en otro proceso: %v", orphan, cmd.Process.Pid, err)
	}
}

// HCD-069 T5 (H-05 / C-004): si un refresh anterior de /tasks sigue en vuelo
// durante session_shutdown -> session_start, el pendingCtx encolado por la
// nueva sesión no debe descartarse al resolver el exec antiguo.
func TestHCD069TasksSessionSwitchRunsPendingRefresh(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import tasksExt from "./gomemory-tasks.ts";

const fails = [];
const handlers = {};
let execCalls = 0;
let resolveFirst;
tasksExt({
	on(ev, fn) { handlers[ev] = fn; },
	registerCommand() {},
	registerShortcut() {},
	exec() {
		execCalls++;
		if (execCalls === 1) return new Promise((res) => { resolveFirst = res; });
		return Promise.resolve({
			code: 0,
			stdout: JSON.stringify({
				ok: true,
				unit_id: "HCD-999",
				kind: "fix",
				stage: "implement",
				state: "active",
				tasks: [{ id: "1", text: "Demo", status: "in_progress" }],
			}),
		});
	},
});

let mounts = 0;
const ctx1 = { cwd: process.cwd(), ui: { setWidget(_id, f) { if (f) mounts++; } } };
const ctx2 = { cwd: process.cwd(), ui: { setWidget(_id, f) { if (f) mounts++; } } };

const p1 = handlers["session_start"]({}, ctx1);
await new Promise((r) => setTimeout(r, 10));
await handlers["session_shutdown"]({}, ctx1);
await handlers["session_start"]({}, ctx2);

resolveFirst({ code: 0, stdout: "{}" });
await p1;
await new Promise((r) => setTimeout(r, 20));
await handlers["session_shutdown"]({}, ctx2);

if (execCalls < 2) fails.push("execCalls=" + execCalls + ", esperado >=2 (pendingCtx descartado)");
if (mounts < 1) fails.push("mounts=" + mounts + ", esperado >=1 para la nueva sesión");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD069TasksSessionSwitchRunsPendingRefresh: %s", out)
	}
}

// HCD-069 T6 (H-06 / C-005): RemoveStaleDepsTemps debe barrer también los
// directorios .pi-gomemory-staging-* huérfanos que hayan quedado tras una
// interrupción antes de os.Rename.
func TestHCD069RemoveStaleStagingTemps(t *testing.T) {
	opts := toolSandbox(t)
	if err := os.MkdirAll(opts.LocalBinDir, 0o755); err != nil {
		t.Fatal(err)
	}

	deadStaging := filepath.Join(opts.LocalBinDir, ".pi-gomemory-staging-dead")
	if err := os.MkdirAll(deadStaging, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deadStaging, ".pid"), []byte("99999999\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	RemoveStaleDepsTemps(opts.LocalBinDir)

	if _, err := os.Stat(deadStaging); !os.IsNotExist(err) {
		t.Fatalf("el staging huérfano %s debía eliminarse: %v", deadStaging, err)
	}
}
