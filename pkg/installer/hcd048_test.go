package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHCD048H01SidebarReentrancy(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import factory from "./gomemory-sidebar.ts";
const commands = {};
const handlers = {};
const pendingExecs = [];
const fakePi = {
	registerCommand: (name, def) => { commands[name] = def; },
	registerTool: () => {},
	on: (ev, fn) => { handlers[ev] = fn; return () => {}; },
	exec: () => new Promise((resolve) => {
		pendingExecs.push(() => resolve({ code: 1, stdout: "", stderr: "" }));
	}),
};
const flushExecs = async () => {
	for (let i = 0; i < 6; i++) {
		while (pendingExecs.length > 0) {
			pendingExecs.shift()();
		}
		await new Promise((r) => setTimeout(r, 5));
	}
};
const tui = { terminal: { rows: 40, columns: 160 }, requestRender: () => {} };
let customCount = 0;
const ctx = {
	hasUI: true,
	cwd: process.cwd(),
	ui: {
		notify: () => {},
		setWidget: () => {},
		custom: (factoryFn) => new Promise((resolve) => {
			customCount++;
			factoryFn(tui, { fg: (_c, s) => s, style: (s) => s }, {}, () => { resolve(); });
		}),
	},
};

factory(fakePi);

// 1. Dos invocaciones concurrentes mientras fetchTitle espera a pi.exec
const p1 = commands.sidebar.handler("", ctx);
const p2 = commands.sidebar.handler("", ctx);
await flushExecs();
await Promise.all([p1, p2]);

const fails = [];
if (customCount !== 1) fails.push("concurrent open mounted " + customCount + " overlays (want 1)");

// Cerrar el panel abierto antes de probar session_shutdown
await commands.sidebar.handler("", ctx);
customCount = 0;

// 2. session_shutdown durante fetchTitle pendiente no debe abrir overlay
const p3 = commands.sidebar.handler("", ctx);
await new Promise((r) => setTimeout(r, 10));
await handlers.session_shutdown({}, ctx);
await flushExecs();
await p3;

if (customCount !== 0) fails.push("open after session_shutdown mounted " + customCount + " overlays (want 0)");

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD048H01SidebarReentrancy: %s", out)
	}
}

func TestHCD048H02PlaywrightNonZeroExit(t *testing.T) {
	project, bin := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(project, "playwright.config.ts"), "export default {}")
	writeExec(t, filepath.Join(bin, "npx"), `
if [ -n "$PLAYWRIGHT_JSON_OUTPUT_NAME" ]; then
  cat << 'EOF' > "$PLAYWRIGHT_JSON_OUTPUT_NAME"
{
  "suites": [
    {
      "file": "tests/demo.spec.ts",
      "line": 1,
      "column": 1,
      "specs": [
        {
          "title": "passes before teardown crash",
          "file": "tests/demo.spec.ts",
          "line": 1,
          "column": 1,
          "tests": [
            {
              "projectName": "chromium",
              "status": "expected",
              "results": [{"status": "passed", "duration": 12, "errors": []}]
            }
          ]
        }
      ]
    }
  ]
}
EOF
fi
echo "Error in globalTeardown: boom" >&2
exit 1
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"},
		`process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
`+tddToolHarness+`
const result = await run("playwright", `+jsStr(project)+`);
const suite = suiteOf(result.details, "playwright");
if (!suite || suite.status !== "failed") {
	console.log("FAIL status=" + (suite && suite.status));
} else {
	console.log("PASS");
}
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD048H02PlaywrightNonZeroExit: %s", out)
	}
}

func TestHCD048H03UninstallMarkerOnFail(t *testing.T) {
	t.Parallel()
	// Como root, chmod 0500 no impide escribir y Uninstall no puede fallar (HCD-079, Q1).
	if os.Geteuid() == 0 {
		t.Skip("como root los permisos de solo lectura no bloquean la escritura")
	}

	home := t.TempDir()
	opts := &Options{
		HomeDir:     home,
		PiAgentDir:  filepath.Join(home, ".pi", "agent"),
		LocalBinDir: filepath.Join(home, ".local", "bin"),
	}
	themesDir := filepath.Join(opts.PiAgentDir, "themes")
	themeFile := filepath.Join(themesDir, "gomemory.json")
	writeFile(t, themeFile, `{"name":"gomemory"}`)
	if err := WriteManifest(opts, "v1.15.1"); err != nil {
		t.Fatal(err)
	}
	// Bloquear escritura en themesDir para que Uninstall falle al retirar themeFile.
	if err := os.Chmod(themesDir, 0500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(themesDir, 0755) }()

	var buf bytes.Buffer
	err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: true}, &buf)
	if err == nil {
		t.Fatalf("expected Uninstall to fail when themesDir is read-only; output:\n%s", buf.String())
	}
	if IsUninstalled(home) {
		t.Fatalf("IsUninstalled(%q) = true after failed Uninstall; expected false", home)
	}
}

func TestHCD048H04SanitizeFrontmatterLongLine(t *testing.T) {
	t.Parallel()

	longLine := strings.Repeat("a", 80*1024)
	input := "---\nname: demo\ndescription: a: b\n---\n" + longLine + "\nTAIL\n"
	got := string(SanitizeFrontmatter([]byte(input)))
	if !strings.Contains(got, "\nTAIL\n") {
		t.Fatalf("SanitizeFrontmatter truncated content after >64 KiB line (got len=%d, want >= %d)", len(got), len(input))
	}
	if !strings.Contains(got, `description: "a: b"`) {
		t.Fatalf("SanitizeFrontmatter did not sanitize description")
	}
}

func TestHCD048H05TasksConcurrentToggle(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import factory from "./gomemory-tasks.ts";
const commands = {};
const handlers = {};
let resolveExec;
const fakePi = {
	registerCommand: (name, def) => { commands[name] = def; },
	on: (ev, fn) => { handlers[ev] = fn; },
	exec: () => new Promise((resolve) => { resolveExec = resolve; }),
};
const widgetCalls = [];
const ctx = {
	cwd: process.cwd(),
	ui: {
		notify: () => {},
		setWidget: (_name, lines) => { widgetCalls.push(lines); },
	},
};

factory(fakePi);

// Inicia un refresh en segundo plano (busy = true)
const bg = handlers.agent_end({}, ctx);
await new Promise((r) => setTimeout(r, 10));

// Mientras pi.exec está en vuelo, la persona oculta el panel con /tasks
const toggle = commands.tasks.handler("", ctx);

// Ahora resuelve pi.exec con datos válidos de una unidad
resolveExec({
	code: 0,
	stdout: JSON.stringify({
		ok: true,
		state: "active",
		unit_id: "HCD-048",
		kind: "fix",
		stage: "implement",
		tasks: [{ id: "1", text: "demo", status: "pending" }],
	}),
	stderr: "",
});
await Promise.all([bg, toggle]);

const last = widgetCalls[widgetCalls.length - 1];
if (widgetCalls.length === 0 || last !== undefined) {
	console.log("FAIL last setWidget=" + JSON.stringify(last) + " calls=" + JSON.stringify(widgetCalls));
} else {
	console.log("PASS");
}
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD048H05TasksConcurrentToggle: %s", out)
	}
}
