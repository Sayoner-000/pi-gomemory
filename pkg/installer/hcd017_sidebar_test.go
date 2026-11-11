package installer

import (
	"strings"
	"testing"
)

// TestHCD017SidebarTurnEndRefresh verifica que el sidebar se suscribe a turn_end y refresca con fuerza
func TestHCD017SidebarTurnEndRefresh(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import sidebarFactory from "./gomemory-sidebar.ts";

const eventsRegistered = [];
const fakePi = {
	registerCommand: () => {},
	on: (event, handler) => {
		eventsRegistered.push(event);
		return () => {};
	},
	exec: async () => ({ code: 0, stdout: "" }),
};

sidebarFactory(fakePi);

const fails = [];
if (!eventsRegistered.includes("turn_end")) {
	fails.push("no se registro handler para turn_end");
}

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)

	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD017SidebarTurnEndRefresh fallo: %s", out)
	}
}

// TestHCD017SidebarHeartbeatTimer verifica que el heartbeat y TTL no bloquean la actualizacion
func TestHCD017SidebarHeartbeatTimer(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import { shouldRefreshQuota } from "./gomemory-sidebar.ts";

const fails = [];
// Al cumplirse el intervalo de heartbeat (30s) sin error previo, debe permitir refrescar
const now = 100000;
const lastFetch = now - 30000; // 30s atras
const allowHeartbeat = shouldRefreshQuota(now, lastFetch, false, false);
if (!allowHeartbeat) {
	fails.push("shouldRefreshQuota rechazo refresco tras 30s de heartbeat");
}

// Ante consumo de turno (force = true), siempre debe permitir refrescar independientemente del tiempo transcurrido
const allowTurn = shouldRefreshQuota(now, now - 1000, true, false);
if (!allowTurn) {
	fails.push("shouldRefreshQuota rechazo refresco forzado en fin de turno");
}

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)

	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD017SidebarHeartbeatTimer fallo: %s", out)
	}
}

// TestHCD017SidebarProviderDetection verifica la deteccion robusta de proveedores
func TestHCD017SidebarProviderDetection(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import { detectProvider } from "./gomemory-sidebar.ts";

const fails = [];

// 1. Antigravity desde ctx.model
if (detectProvider({ model: { provider: "antigravity", id: "gemini-2.5-pro" } }) !== "antigravity") {
	fails.push("no detecto antigravity en ctx.model");
}

// 2. Agy alias
if (detectProvider({ model: { provider: "agy", id: "claude-3-5-sonnet" } }) !== "antigravity") {
	fails.push("no detecto agy como antigravity");
}

// 3. Fallback a process.env.PI_PROVIDER
process.env.PI_PROVIDER = "antigravity";
if (detectProvider({}) !== "antigravity") {
	fails.push("no detecto antigravity via process.env.PI_PROVIDER");
}

// 4. Claude Code / Anthropic
delete process.env.PI_PROVIDER;
if (detectProvider({ model: { provider: "anthropic", id: "claude-3-7-sonnet" } }) !== "claude") {
	fails.push("no detecto claude en ctx.model");
}

// 5. OpenCode / Codex
if (detectProvider({ model: { provider: "opencode", id: "codex-preview" } }) !== "opencode") {
	fails.push("no detecto opencode en ctx.model");
}

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)

	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD017SidebarProviderDetection fallo: %s", out)
	}
}
