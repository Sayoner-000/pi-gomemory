package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// HCD-066 T1 (1A): InstallTheme fija hideThinkingBlock=true por defecto en
// settings.json sin tocar defaultThinkingLevel ni pisar una preferencia previa
// explícita del usuario, y RunDoctor informa su estado.
func TestHCD066ThinkingDefaultsAndDoctor(t *testing.T) {
	t.Parallel()

	t.Run("default_true_on_fresh_settings", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		piDir := filepath.Join(home, ".pi", "agent")
		if err := InstallTheme(repoAssets(t), piDir); err != nil {
			t.Fatalf("InstallTheme: %v", err)
		}
		raw, err := os.ReadFile(filepath.Join(piDir, "settings.json"))
		if err != nil {
			t.Fatal(err)
		}
		var s map[string]any
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatal(err)
		}
		if got, ok := s["hideThinkingBlock"].(bool); !ok || !got {
			t.Fatalf("hideThinkingBlock = %#v, esperado true", s["hideThinkingBlock"])
		}
		if _, hasLevel := s["defaultThinkingLevel"]; hasLevel {
			t.Fatalf("no debe fijar defaultThinkingLevel (opción 1A): %#v", s["defaultThinkingLevel"])
		}

		opts := &Options{HomeDir: home, PiAgentDir: piDir, LocalBinDir: filepath.Join(home, ".local", "bin")}
		items := RunDoctor(opts)
		found := false
		for _, it := range items {
			if strings.Contains(it.Name, "Thinking") && it.Status == "OK" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("RunDoctor debe reportar check OK para Thinking colapsado (Ctrl+T): %+v", items)
		}
	})

	t.Run("preserves_explicit_user_false", func(t *testing.T) {
		t.Parallel()
		home := t.TempDir()
		piDir := filepath.Join(home, ".pi", "agent")
		if err := os.MkdirAll(piDir, 0o755); err != nil {
			t.Fatal(err)
		}
		settingsPath := filepath.Join(piDir, "settings.json")
		if err := os.WriteFile(settingsPath, []byte(`{"hideThinkingBlock":false}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := InstallTheme(repoAssets(t), piDir); err != nil {
			t.Fatalf("InstallTheme: %v", err)
		}
		raw, err := os.ReadFile(settingsPath)
		if err != nil {
			t.Fatal(err)
		}
		var s map[string]any
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatal(err)
		}
		if got, ok := s["hideThinkingBlock"].(bool); !ok || got {
			t.Fatalf("hideThinkingBlock fue sobrescrito: %#v, debía respetarse false", s["hideThinkingBlock"])
		}
	})
}

// HCD-066 T2 (2A): gomemory-status.ts configura en session_start la etiqueta
// "(thinking...)" y el indicador animado de gotitas corriendo en glifos de
// texto puro (sin emojis), soporta hosts sin esas APIs y restaura en shutdown.
func TestHCD066ThinkingIndicator(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-status.ts"}, `
import statusExt from "./gomemory-status.ts";

const fails = [];
const handlers = {};
statusExt({
	on(ev, fn) { handlers[ev] = fn; },
	registerCommand() {},
	exec: async () => ({ code: 1, stdout: "", stderr: "" }),
});

const labels = [];
const indicators = [];
const ctx = {
	ui: {
		theme: { fg: (role, text) => "<" + role + ":" + text + ">" },
		setFooter() {},
		setStatus() {},
		setHiddenThinkingLabel(v) { labels.push(v); },
		setWorkingIndicator(v) { indicators.push(v); },
	},
};

await handlers["session_start"]({}, ctx);
if (labels.length !== 1 || labels[0] !== "(thinking...)") {
	fails.push("etiqueta inicial inesperada: " + JSON.stringify(labels));
}
const ind = indicators[0];
if (!ind || !Array.isArray(ind.frames) || ind.frames.length < 4) {
	fails.push("frames insuficientes: " + JSON.stringify(ind));
} else {
	if (!(typeof ind.intervalMs === "number" && ind.intervalMs >= 60 && ind.intervalMs <= 300)) {
		fails.push("intervalMs fuera de rango: " + ind.intervalMs);
	}
	const emojiRe = /\p{Extended_Pictographic}/u;
	for (const f of ind.frames) {
		if (emojiRe.test(f)) fails.push("frame contiene emoji (viola 2A/HCD-020): " + f);
		if (!f.includes("<accent:")) fails.push("frame sin color accent del tema: " + f);
	}
}

if (typeof handlers["session_shutdown"] !== "function") {
	fails.push("falta manejador session_shutdown");
} else {
	await handlers["session_shutdown"]({}, ctx);
	if (labels.length < 2 || labels.at(-1) !== undefined) {
		fails.push("session_shutdown no restauró hiddenThinkingLabel: " + JSON.stringify(labels));
	}
	if (indicators.length < 2 || indicators.at(-1) !== undefined) {
		fails.push("session_shutdown no restauró workingIndicator: " + JSON.stringify(indicators));
	}
}

// Host mínimo sin las APIs opcionales: no debe lanzar error.
await handlers["session_start"]({}, { ui: { setFooter() {}, setStatus() {} } });

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD066ThinkingIndicator: %s", out)
	}
}

// HCD-066 T3 (3B): al entrar en /plan, la sección gomemory-plan (tanto con
// mem plan-context como en FALLBACK) y piBlock en gomemory-context.ts exigen
// validar el impacto del proyecto con codegraph y codebase-memory en solo
// lectura antes de abrir un HCD.
func TestHCD066PlanImpactProtocol(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-plan.ts", "extensions/gomemory-plan-patterns.json", "extensions/gomemory-context.ts"}, `
import registerPlan, { PLAN_SECTION } from "./gomemory-plan.ts";
import { piBlock } from "./gomemory-context.ts";

const fails = [];

async function captureSection(execImpl) {
	const hooks = new Map();
	let cmd;
	registerPlan({
		registerCommand: (_n, opt) => { cmd = opt.handler; },
		on: (n, h) => hooks.set(n, h),
		exec: execImpl,
		sendUserMessage() {},
	});
	const ctx = { cwd: process.cwd(), ui: { notify() {}, setStatus() {} } };
	await cmd("on", ctx);
	const sections = {};
	await hooks.get("before_agent_start")({ systemPromptOptions: { sections } });
	return sections[PLAN_SECTION] || "";
}

const requiredTokens = [
	"codegraph_explore",
	"Mapa de impacto",
];

const fromMem = await captureSection(async () => ({ code: 0, stdout: "Método base" }));
for (const tok of requiredTokens) {
	if (!fromMem.includes(tok)) fails.push("loadMethod omite " + tok);
}

const fromFallback = await captureSection(async () => ({ code: 1, stdout: "" }));
for (const tok of requiredTokens) {
	if (!fromFallback.includes(tok)) fails.push("FALLBACK omite " + tok);
}

const block = piBlock([]);
if (!block.includes("codegraph") || !block.includes("impacto")) {
	fails.push("piBlock no menciona validación de impacto con codegraph en Modo plan");
}

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD066PlanImpactProtocol: %s", out)
	}
}

// HCD-066 T4 (3B): modo plan permite consultas de codegraph/codebase-memory e
// index_repository sin persistence, y bloquea mutaciones de código, HCD y
// escrituras en el repositorio (codegraph index, persistence:true, delete_project,
// manage_adr de escritura).
func TestHCD066PlanReadOnlyGuards(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-plan.ts", "extensions/gomemory-plan-patterns.json"}, `
import registerPlan from "./gomemory-plan.ts";

const fails = [];
const hooks = new Map();
let cmd;
registerPlan({
	registerCommand: (_n, opt) => { cmd = opt.handler; },
	on: (n, h) => hooks.set(n, h),
	exec: async () => ({ code: 0, stdout: "ok" }),
	sendUserMessage() {},
});
const ctx = { cwd: process.cwd(), ui: { notify() {}, setStatus() {} } };
await cmd("on", ctx);
const call = (toolName, input) => hooks.get("tool_call")({ toolName, input });

for (const [tool, input] of [
	["codegraph_explore", { query: "InstallAll" }],
	["bash", { command: "mem hcd show --json" }],
	["bash", { command: "codegraph explore Foo" }],
]) {
	if (await call(tool, input)) fails.push("bloqueó lectura " + tool + " " + JSON.stringify(input));
}

for (const [tool, input] of [
	["hcd_rename", { title: "nuevo" }],
	["mcp__gomemory__hcd_rename", { title: "nuevo" }],
	["bash", { command: "mem hcd start --kind fix --title x" }],
	["bash", { command: "mem hcd verify --task 1 -- go test" }],
	["bash", { command: "mem hcd close --quote ok" }],
	["bash", { command: "codegraph index ." }],
]) {
	const res = await call(tool, input);
	if (!res?.block) fails.push("no bloqueó mutación " + tool + " " + JSON.stringify(input));
}

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD066PlanReadOnlyGuards: %s", out)
	}
}

// HCD-066 T7: el protocolo base de Pi (gomemory-context.ts) debe exigir medir
// el impacto completo del proyecto con codegraph y codebase-memory SIEMPRE,
// independientemente de si /plan está activo o no, antes de cualquier cambio o HCD.
func TestHCD066AlwaysOnImpactProtocol(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-context.ts"}, `
import { piBlock } from "./gomemory-context.ts";

const fails = [];
const block = piBlock([]);
const impactLine = block.split("\n").find((l) => /impacto obligatorio/i.test(l));
if (!impactLine) {
	fails.push("falta viñeta dedicada de Impacto obligatorio independiente de /plan");
} else {
	for (const tok of ["con o sin /plan", "codegraph_explore", "HCD"]) {
		if (!impactLine.includes(tok)) fails.push("viñeta de impacto omite " + tok);
	}
}
if (/mcp__\w+__\w+/.test(block)) {
	fails.push("piBlock no debe nombrar tools mcp__ concretas (C-04 HCD-045)");
}
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD066AlwaysOnImpactProtocol: %s", out)
	}
}
