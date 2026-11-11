package installer

import (
	"strings"
	"testing"
)

// C-06: el sidebar muestra "Configured" y no inventa servidores.
func TestHCD012Sidebar(t *testing.T) {
	if strings.Contains(readAsset(t, "extensions/gomemory-sidebar.ts"), "Connected") {
		t.Error("gomemory-sidebar.ts todavía contiene Connected")
	}
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { getRegisteredMCPServers } from "./gomemory-sidebar.ts";

const fails = [];
const d = mkdtempSync(join(tmpdir(), "sb-"));
if (getRegisteredMCPServers(d).length !== 0) fails.push("sin mcp.json no está vacío");
writeFileSync(join(d, "mcp.json"), "{no json");
if (getRegisteredMCPServers(d).length !== 0) fails.push("JSON inválido no está vacío");
writeFileSync(join(d, "mcp.json"), JSON.stringify({ mcpServers: { a: {}, b: {} } }));
const s = getRegisteredMCPServers(d);
if (s.length !== 2 || s.some((x) => x.status !== "Configured")) fails.push(JSON.stringify(s));
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if out != "PASS" {
		t.Fatal(out)
	}
}
