package installer

import (
	"strings"
	"testing"
)

// C-004: con un título largo, renderSidebar debe conservar el control [−].
// Los tests de cuota Antigravity (C-005) y de /tdd (C-003, C-006) se retiraron
// con esas funciones en HCD-085.
func TestHCD021SidebarLongTitle(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import { renderSidebar } from "./gomemory-sidebar.ts";
const theme = { fg: (_c, s) => s, style: (s) => s };
const state = {
	title: "HCD · HCD-020 · evolution · paso close · v1.9.0", tokens: 1, contextLimit: 10, cost: 0,
	servers: [], providerName: "p", isClaudeProvider: false, claudeUsage: null,
};
const fails = [];
for (const width of [36, 24]) {
	const header = renderSidebar(state, width, theme)[1];
	if (!header.includes("[−]")) fails.push("ancho " + width + ": la cabecera perdió [−]: " + JSON.stringify(header));
	if (!header.includes("…")) fails.push("ancho " + width + ": el título largo debe recortarse con …");
	if ([...header].length !== width) fails.push("ancho " + width + ": la fila mide " + [...header].length);
}
const short = renderSidebar({ ...state, title: "HCD · corto" }, 36, theme)[1];
if (!short.includes("HCD · corto") || short.includes("…")) fails.push("un título corto no se recorta");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD021SidebarLongTitle: %s", out)
	}
}
