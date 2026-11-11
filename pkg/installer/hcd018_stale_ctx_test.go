package installer

import (
	"strings"
	"testing"
)

// TestHCD018StaleCtxAndSessionShutdown verifica que las extensiones gomemory-sidebar y gomemory-tdd
// se suscriben a session_shutdown para limpiar timers y overlays, y que los lectores de ctx
// (detectProvider, collectContextMetrics) no arrojan si ctx.model lanza un error de ctx obsoleto (stale).
func TestHCD018StaleCtxAndSessionShutdown(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts", "extensions/gomemory-tdd.ts"}, `
import sidebarFactory, { detectProvider, collectContextMetrics } from "./gomemory-sidebar.ts";
import tddFactory from "./gomemory-tdd.ts";

const fails = [];

// 1. Simular un stale ctx cuyo getter model arroja el error exacto de Pi en reload
const staleCtx = {
	get model() {
		throw new Error("This extension ctx is stale after session replacement or reload.");
	},
	sessionManager: {
		getBranch: () => {
			throw new Error("This extension ctx is stale after session replacement or reload.");
		},
	},
	cwd: "/dummy",
};

try {
	const prov = detectProvider(staleCtx);
	if (typeof prov !== "string") {
		fails.push("detectProvider con stale ctx debe retornar string de fallback, dio: " + prov);
	}
} catch (e) {
	fails.push("detectProvider no debe lanzar excepcion con stale ctx: " + e.message);
}

try {
	const metrics = collectContextMetrics(staleCtx);
	if (typeof metrics.tokens !== "number" || typeof metrics.contextLimit !== "number") {
		fails.push("collectContextMetrics debe devolver métricas por defecto ante stale ctx");
	}
} catch (e) {
	fails.push("collectContextMetrics no debe lanzar excepcion con stale ctx: " + e.message);
}

// 2. Verificar que sidebar y tdd se suscriben a session_shutdown
const sidebarEvents = [];
const fakePiSidebar = {
	registerCommand: () => {},
	registerTool: () => {},
	on: (event, handler) => { sidebarEvents.push(event); return () => {}; },
	exec: async () => ({ code: 0, stdout: "" }),
};
sidebarFactory(fakePiSidebar);

if (!sidebarEvents.includes("session_shutdown")) {
	fails.push("gomemory-sidebar debe suscribirse a session_shutdown para limpiar timer y overlays");
}

const tddEvents = [];
const fakePiTDD = {
	registerCommand: () => {},
	registerTool: () => {},
	on: (event, handler) => { tddEvents.push(event); return () => {}; },
	exec: async () => ({ code: 0, stdout: "" }),
};
tddFactory(fakePiTDD);

if (!tddEvents.includes("session_shutdown")) {
	fails.push("gomemory-tdd debe suscribirse a session_shutdown para limpiar overlays");
}

console.log(fails.length ? "FAIL: " + fails.join(" | ") : "PASS");
`)

	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD018StaleCtxAndSessionShutdown fallo (comprobando rojo):\n%s", out)
	}
}
