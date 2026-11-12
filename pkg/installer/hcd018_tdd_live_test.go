package installer

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestHCD018TDDLiveExecutionMultiTest verifica que el adaptador de backend registra
// cada test individual de `go test -json` en el TDDStore (pasados y fallidos) y que
// la suite no queda vacía (0/0).
//
// HCD-020: antes simulaba pi.exec con un JSON inventado sobre cwd "/dummy"; ese
// contrato ocultaba que /tdd no ejecutaba nada real. Ahora corre un módulo Go real.
func TestHCD018TDDLiveExecutionMultiTest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module multi\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "shop", "shop_test.go"), `package shop

import "testing"

func TestLogin(t *testing.T)    {}
func TestCart(t *testing.T)     {}
func TestCheckout(t *testing.T) { t.Fatal("checkout roto") }
`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import tddExtensionFactory from "./gomemory-tdd.ts";

let registeredTool = null;
tddExtensionFactory({
	registerCommand: () => {},
	registerTool: (def) => { registeredTool = def; },
	on: () => () => {},
	exec: async () => ({ code: 0, stdout: "", stderr: "" }),
});

const fails = [];
const res = await registeredTool.execute("call-1", { suite: "backend" }, undefined, undefined, { cwd: `+jsStr(dir)+` });
const backend = res.details.suites.find((s) => s.id === "backend");
if (!backend || backend.tests.length < 3) {
	fails.push("backend debe tener al menos 3 tests, tiene " + backend?.tests.length);
} else {
	const byName = (n) => backend.tests.find((t) => t.name === n);
	if (byName("TestLogin")?.status !== "passed") fails.push("TestLogin debió ser passed");
	if (byName("TestCheckout")?.status !== "failed") fails.push("TestCheckout debió ser failed");
}
if (res.details.stats.total < 3) fails.push("total de tests debe ser >= 3, dio " + res.details.stats.total);
console.log(fails.length ? "FAIL: " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD018TDDLiveExecutionMultiTest: %s", out)
	}
}
