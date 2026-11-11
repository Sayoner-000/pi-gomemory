package installer

import (
	"testing"
)

func TestHCD015SidebarUnicodeBars(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import { renderSidebar, formatProgressBar, compactBucketName } from "./gomemory-sidebar.ts";

const fails = [];

// 1. Barras Unicode
const b100 = formatProgressBar(1.0, 8);
if (b100 !== "████████") fails.push("barra 100% no es sólida: " + b100);

const b50 = formatProgressBar(0.5, 8);
if (b50 !== "████░░░░") fails.push("barra 50% incorrecta: " + b50);

const b0 = formatProgressBar(0.0, 8);
if (b0 !== "░░░░░░░░") fails.push("barra 0% incorrecta: " + b0);

// 2. Nombres compactos
if (compactBucketName("Weekly Limit Remaining") !== "Semanal") fails.push("compact semanal");
if (compactBucketName("Five Hour Limit Remaining") !== "5 Horas") fails.push("compact 5 horas");
if (compactBucketName("Gemini 4 Argon Weekly Limit") !== "Argon") fails.push("compact argon");

const dummyTheme = {
	fg: (color, text) => text,
	bg: (color, text) => text,
	style: (text, opts) => text,
};

// 3. Render con Claude Usage
const stateClaude = {
	title: "HCD-015",
	tokens: 450,
	contextLimit: 200000,
	cost: 0.02,
	servers: [{ name: "gomemory", status: "Configured" }],
	isClaudeProvider: true,
	claudeUsage: {
		sessions: 12,
		today: { input: 1000, output: 500, cache_create: 2000, cache_read: 8000, messages: 5 },
		week: { input: 10000, output: 5000, cache_create: 20000, cache_read: 80000, messages: 50 },
		total: { input: 100000, output: 50000, cache_create: 200000, cache_read: 800000, messages: 500 },
	},
};

const linesClaude = renderSidebar(stateClaude, 36, dummyTheme);
const textClaude = linesClaude.join("\n");

if (!textClaude.includes("Claude Code Usage")) fails.push("falta header Claude Code Usage");
if (!textClaude.includes("Hoy:")) fails.push("falta Hoy en Claude");
if (!textClaude.includes("Semana:")) fails.push("falta Semana en Claude");
if (!textClaude.includes("Total:")) fails.push("falta Total en Claude");
if (!textClaude.includes("Ahorro caché:")) fails.push("falta Ahorro caché");

// Verificar que ninguna línea excede 36 caracteres visibles
for (const line of linesClaude) {
	if (line.length > 40) { // con bordes
		fails.push("Línea demasiado ancha: " + line);
	}
}

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)

	if out != "PASS" {
		t.Fatal(out)
	}
}
