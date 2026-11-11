package installer

import (
	"strings"
	"testing"
)

// TestHCD018TDDLiveExecutionMultiTest verifica que el adaptador de backend procesa
// la salida JSON de go test registrando cada test individual en el TDDStore,
// y que al correr suites multi-dominio las suites no quedan vacías (0/0).
func TestHCD018TDDLiveExecutionMultiTest(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import tddExtensionFactory, { createTDDStore } from "./gomemory-tdd.ts";

const fakeJsonOutput = [
	'{"Action":"run","Test":"TestLogin"}',
	'{"Action":"pass","Test":"TestLogin","Elapsed":0.05}',
	'{"Action":"run","Test":"TestCart"}',
	'{"Action":"pass","Test":"TestCart","Elapsed":0.08}',
	'{"Action":"run","Test":"TestCheckout"}',
	'{"Action":"fail","Test":"TestCheckout","Elapsed":0.15}',
].join("\n");

let registeredTool = null;
const fakePi = {
	registerCommand: () => {},
	registerTool: (def) => { registeredTool = def; },
	on: () => () => {},
	exec: async (cmd, args) => {
		if (cmd === "go" && args.includes("-json")) {
			return { code: 1, stdout: fakeJsonOutput, stderr: "" };
		}
		return { code: 0, stdout: "{}", stderr: "" };
	},
};

tddExtensionFactory(fakePi);

const fails = [];
if (!registeredTool) {
	fails.push("no se registro la tool run_test_suite");
} else {
	// Ejecutar la suite
	registeredTool.execute("call-1", { suite: "all" }, null, null, { cwd: "/dummy" }).then(res => {
		const state = res.details;
		const backendSuite = state.suites.find(s => s.id === "backend");
		if (!backendSuite) {
			fails.push("falta suite backend");
		} else if (backendSuite.tests.length < 3) {
			fails.push("backend debe tener al menos 3 tests parseados, tiene " + backendSuite.tests.length);
		} else {
			const login = backendSuite.tests.find(t => t.name === "TestLogin");
			const checkout = backendSuite.tests.find(t => t.name === "TestCheckout");
			if (!login || login.status !== "passed") fails.push("TestLogin debio ser passed");
			if (!checkout || checkout.status !== "failed") fails.push("TestCheckout debio ser failed");
		}

		if (state.stats.total < 3) {
			fails.push("total de tests debe ser >= 3, dio " + state.stats.total);
		}

		console.log(fails.length ? "FAIL: " + fails.join(" | ") : "PASS");
	});
}
`)

	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD018TDDLiveExecutionMultiTest fallo (rojo esperado):\n%s", out)
	}
}
