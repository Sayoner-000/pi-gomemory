package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// HCD-065 T2, T5 y T6 cubrían /tdd y se retiraron con él en HCD-085.

// HCD-065 T3 (H-03 / C-006): un refresh de /tasks en vuelo que se resuelve tras
// session_shutdown no debe remontar el widget ni reactivar timers (500 ms / 10 s).
func TestHCD065TasksShutdownStopsLateRefresh(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import tasksExt from "./gomemory-tasks.ts";
const fails = [];
const handlers = {};
let resolveExec;
const live = new Set();
const origSetInterval = globalThis.setInterval;
const origClearInterval = globalThis.clearInterval;
globalThis.setInterval = (...args) => {
	const t = origSetInterval(...args);
	live.add(t);
	return t;
};
globalThis.clearInterval = (t) => {
	live.delete(t);
	return origClearInterval(t);
};

tasksExt({
	on(ev, fn) { handlers[ev] = fn; },
	registerCommand() {},
	registerShortcut() {},
	exec() { return new Promise((res) => { resolveExec = res; }); },
});

let widgetMounts = 0;
const ctx = {
	cwd: process.cwd(),
	ui: {
		setWidget(_id, factory) {
			if (factory) widgetMounts++;
		},
	},
};

const pending = handlers["session_start"]({}, ctx);
await new Promise((r) => setTimeout(r, 10));
await handlers["session_shutdown"]({}, ctx);
if (widgetMounts !== 0) fails.push("montó antes de tiempo: " + widgetMounts);
if (live.size !== 0) fails.push("timers vivos tras shutdown: " + live.size);

resolveExec({
	code: 0,
	stdout: JSON.stringify({
		ok: true,
		unit_id: "HCD-999",
		kind: "fix",
		stage: "implement",
		state: "active",
		tasks: [{ id: "1", text: "Demo task", status: "in_progress" }],
	}),
});
await pending;

if (widgetMounts !== 0) fails.push("remontó widget tras shutdown: " + widgetMounts);
if (live.size !== 0) fails.push("fugó timers tras shutdown: " + live.size);
for (const t of live) origClearInterval(t);
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD065TasksShutdownStopsLateRefresh: %s", out)
	}
}

// HCD-065 T4 (H-04 / C-007): tanto el sidebar como el footer deben incluir
// cacheRead y cacheWrite (con respaldo totalTokens) al calcular el uso de
// contexto de la sesión.
func TestHCD065ContextMetricsIncludesCache(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts", "extensions/gomemory-status.ts"}, `
import { collectContextMetrics } from "./gomemory-sidebar.ts";
import statusExt from "./gomemory-status.ts";

const fails = [];
const branch = [
	{
		type: "message",
		message: {
			role: "assistant",
			usage: {
				input: 100,
				output: 200,
				cacheRead: 100000,
				cacheWrite: 5000,
				totalTokens: 105300,
				cost: { total: 0.15 },
			},
		},
	},
];
const ctx = {
	model: { id: "claude-sonnet", contextWindow: 200000 },
	sessionManager: { getBranch: () => branch },
};

const m = collectContextMetrics(ctx);
if (m.tokens !== 105300) fails.push("sidebar tokens=" + m.tokens + ", esperado 105300");

// Respaldo totalTokens cuando input/output/cache vienen a cero.
const fallback = collectContextMetrics({
	model: { contextWindow: 200000 },
	sessionManager: {
		getBranch: () => [{ type: "message", message: { role: "assistant", usage: { totalTokens: 60000 } } }],
	},
});
if (fallback.tokens !== 60000) fails.push("sidebar fallback totalTokens=" + fallback.tokens + ", esperado 60000");

// Footer de gomemory-status.ts
const handlers = {};
let footerFactory = null;
statusExt({
	on(ev, fn) { handlers[ev] = fn; },
	registerCommand() {},
	exec: async () => ({ code: 1, stdout: "", stderr: "" }),
});
const statusCtx = {
	...ctx,
	ui: {
		setFooter(fn) { footerFactory = fn; },
	},
};
await handlers["session_start"]({}, statusCtx);
if (!footerFactory) {
	fails.push("status no registró setFooter");
} else {
	const theme = {
		fg: (_role, text) => text,
		style: (text) => text,
	};
	const footer = footerFactory({ requestRender() {} }, theme, {});
	const line = footer.render(140)[0] || "";
	if (!line.includes("53%")) fails.push("footer no muestra 53%: " + line);
	if (!line.includes("105.3k/200.0k")) fails.push("footer no muestra 105.3k/200.0k: " + line);
}

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD065ContextMetricsIncludesCache: %s", out)
	}
}

// HCD-065 T7 (H-07 / C-002): un directorio residual bajo depsTempPrefix que no
// está en liveTemps pero contiene el PID del proceso actual (por reciclaje de
// PID) es un huérfano y RemoveStaleDepsTemps debe eliminarlo, mientras sigue
// preservando un temporal cuyo propietario es otro proceso vivo (p. ej. ppid).
func TestHCD065RemoveStaleOwnPIDOrphan(t *testing.T) {
	opts := toolSandbox(t)
	if err := os.MkdirAll(opts.LocalBinDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ownOrphan := filepath.Join(opts.LocalBinDir, depsTempPrefix+"own-pid-orphan")
	if err := os.MkdirAll(ownOrphan, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ownOrphan, ".pid"), []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	otherLive := filepath.Join(opts.LocalBinDir, depsTempPrefix+"other-ppid-live")
	if err := os.MkdirAll(otherLive, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherLive, ".pid"), []byte(strconv.Itoa(os.Getppid())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	RemoveStaleDepsTemps(opts.LocalBinDir)

	if _, err := os.Stat(ownOrphan); !os.IsNotExist(err) {
		t.Fatalf("el huérfano %s con pid==os.Getpid() no registrado en liveTemps debía eliminarse: %v", ownOrphan, err)
	}
	if _, err := os.Stat(otherLive); err != nil {
		t.Fatalf("el temporal %s de otro proceso vivo (ppid) debía preservarse: %v", otherLive, err)
	}
}

// HCD-065 T8 (H-08 / C-003): depsTempDir debe escribir .pid antes de exponer el
// directorio bajo depsTempPrefix (vía staging + os.Rename) y propagar el error
// si falla la escritura de .pid, sin dejar ventana TOCTOU ni tragar errores.
func TestHCD065DepsTempDirAtomicPID(t *testing.T) {
	opts := toolSandbox(t)
	if err := os.MkdirAll(opts.LocalBinDir, 0o755); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile("tools_exec.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "_ = writeFileAtomic(") {
		t.Fatalf("tools_exec.go no debe ignorar el error de writeFileAtomic al escribir .pid")
	}

	// Prueba concurrente bajo -race: mientras se crean y borran temporales,
	// ningún directorio visible con depsTempPrefix debe carecer de .pid.
	stop := make(chan struct{})
	seenMissingPID := make(chan string, 1)
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				entries, _ := os.ReadDir(opts.LocalBinDir)
				for _, e := range entries {
					if e.IsDir() && strings.HasPrefix(e.Name(), depsTempPrefix) {
						p := filepath.Join(opts.LocalBinDir, e.Name())
						if _, err := os.Stat(filepath.Join(p, ".pid")); os.IsNotExist(err) {
							// Confirmar que el directorio no fue borrado entre ReadDir y Stat.
							if _, dirErr := os.Stat(p); dirErr == nil {
								select {
								case seenMissingPID <- p:
								default:
								}
								return
							}
						}
					}
				}
			}
		}
	}()

	var cleanups []func()
	defer func() {
		for _, fn := range cleanups {
			fn()
		}
	}()
	for i := 0; i < 40; i++ {
		dir, cleanup, err := depsTempDir(opts)
		if err != nil {
			close(stop)
			t.Fatalf("depsTempDir iter %d: %v", i, err)
		}
		cleanups = append(cleanups, cleanup)
		if !strings.HasPrefix(filepath.Base(dir), depsTempPrefix) {
			close(stop)
			t.Fatalf("nombre inesperado: %s", dir)
		}
		if b, err := os.ReadFile(filepath.Join(dir, ".pid")); err != nil || strings.TrimSpace(string(b)) != strconv.Itoa(os.Getpid()) {
			close(stop)
			t.Fatalf(".pid inválido en %s: %q (%v)", dir, string(b), err)
		}
	}
	close(stop)

	select {
	case p := <-seenMissingPID:
		t.Fatalf("ventana TOCTOU: se observó %s bajo %s sin archivo .pid", p, depsTempPrefix)
	default:
	}
}

// HCD-065 T9 (H-09 / C-004): Uninstall debe limpiar temporales huérfanos tanto
// en opts.LocalBinDir (p. ej. PI_GOMEMORY_BIN_DIR) como en manifest.BinDir,
// sin que el manifiesto anule el directorio explícito del entorno.
func TestHCD065UninstallCleansEnvAndManifestBinDir(t *testing.T) {
	opts := installSandbox(t)
	binManifest := opts.LocalBinDir
	binEnv := filepath.Join(opts.HomeDir, "bin-env")

	orphanManifest := filepath.Join(binManifest, depsTempPrefix+"orphan-manifest")
	orphanEnv := filepath.Join(binEnv, depsTempPrefix+"orphan-env")
	for _, d := range []string{orphanManifest, orphanEnv} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	uninstallOpts := *opts
	uninstallOpts.LocalBinDir = binEnv
	var buf bytes.Buffer
	if err := Uninstall(repoAssets(t), &uninstallOpts, UninstallFlags{Yes: true}, &buf); err != nil {
		t.Fatalf("Uninstall: %v\n%s", err, buf.String())
	}

	if _, err := os.Stat(orphanManifest); !os.IsNotExist(err) {
		t.Fatalf("el huérfano en manifest.BinDir (%s) debía eliminarse: %v", orphanManifest, err)
	}
	if _, err := os.Stat(orphanEnv); !os.IsNotExist(err) {
		t.Fatalf("el huérfano en PI_GOMEMORY_BIN_DIR (%s) debía eliminarse: %v", orphanEnv, err)
	}
}
