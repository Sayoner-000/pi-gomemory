package installer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHCD027PlaywrightIntegration(t *testing.T) {
	cliPath := os.Getenv("PLAYWRIGHT_TEST_CLI")
	if strings.TrimSpace(cliPath) == "" {
		// Integración opt-in: sin ejecutor declarado se omite (no falla la suite por defecto,
		// /tdd ni CI). Con la variable definida, una ruta rota sigue siendo un fallo.
		t.Skip("bloqueo: PLAYWRIGHT_TEST_CLI no está configurado; no se sustituye por fixture ni se instalan dependencias")
	}
	if _, err := os.Stat(cliPath); err != nil {
		t.Fatalf("bloqueo: ejecutor PLAYWRIGHT_TEST_CLI no accesible en %s: %v", cliPath, err)
	}

	// 1. Obtener y registrar la versión real del ejecutor
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("node no encontrado en PATH: %v", err)
	}
	cmdVersion := exec.Command(node, cliPath, "--version")
	outVersion, err := cmdVersion.CombinedOutput()
	if err != nil {
		t.Fatalf("fallo al consultar versión de Playwright (%s): %v\n%s", cliPath, err, outVersion)
	}
	pwVersion := strings.TrimSpace(string(outVersion))
	t.Logf("Límite verificado: ejecutor Playwright preinstalado versión %q (pre-release; compatibilidad estable no verificada)", pwVersion)

	cliDir := filepath.Dir(cliPath)
	testModulePath := filepath.Join(cliDir, "test.js")
	if _, err := os.Stat(testModulePath); err != nil {
		t.Fatalf("módulo de prueba no accesible en %s: %v", testModulePath, err)
	}

	// 2. Preparar proyecto real con proyectos múltiples, aprobados, omitidos, fallidos y flaky
	bin := t.TempDir()
	writeExec(t, filepath.Join(bin, "npx"), `if [ "$1" = "playwright" ]; then shift; fi
exec `+node+` `+cliPath+` "$@"`)

	project := t.TempDir()
	testsDir := filepath.Join(project, "tests")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatal(err)
	}

	configFile := `
module.exports = {
  testDir: "./tests",
  projects: [
    { name: "alpha", retries: 1 },
    { name: "beta", retries: 0 }
  ]
};
`
	writeFile(t, filepath.Join(project, "playwright.config.js"), configFile)

	passSpec := `
const { test, expect } = require(` + jsStr(testModulePath) + `);
test("math works", () => {
  expect(2 + 2).toBe(4);
});
`
	writeFile(t, filepath.Join(testsDir, "pass.spec.js"), passSpec)

	skipSpec := `
const { test } = require(` + jsStr(testModulePath) + `);
test.skip("skipped feature", () => {});
`
	writeFile(t, filepath.Join(testsDir, "skip.spec.js"), skipSpec)

	flakySpec := `
const { test, expect } = require(` + jsStr(testModulePath) + `);
test("flaky or fail", ({}, testInfo) => {
  if (testInfo.project.name === "alpha") {
    if (testInfo.retry === 0) {
      expect(1).toBe(2);
    } else {
      expect(1).toBe(1);
    }
  } else {
    expect(1).toBe(99);
  }
});
`
	writeFile(t, filepath.Join(testsDir, "flaky.spec.js"), flakySpec)

	// Ejecutar integración completa a través de gomemory-tdd
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import assert from "node:assert/strict";
`+tddToolHarness+`

const project = `+jsStr(project)+`;
const result = await run("playwright", project);
const state = result.details;
const suite = suiteOf(state, "playwright");
assert.ok(suite, "suite playwright debe existir");

// Comprobaciones de resultados
const tests = suite.tests;

// 1. Proyectos: alpha y beta diferenciados
const alphaPass = tests.find(t => t.id.includes("alpha") && t.name.includes("math works"));
const betaPass = tests.find(t => t.id.includes("beta") && t.name.includes("math works"));
assert.ok(alphaPass, "alpha math works debe existir");
assert.ok(betaPass, "beta math works debe existir");
assert.notEqual(alphaPass.id, betaPass.id, "identidades de proyectos deben ser distintas");
assert.equal(alphaPass.status, "passed", "alpha math works aprobado");
assert.equal(betaPass.status, "passed", "beta math works aprobado");

// 2. Omitidos
const skippedTests = tests.filter(t => t.status === "skipped");
assert.equal(skippedTests.length, 2, "debe haber 2 tests omitidos (alpha y beta)");

// 3. Flaky recuperado en alpha
const alphaFlaky = tests.find(t => t.id.includes("alpha") && t.name.includes("flaky or fail"));
assert.ok(alphaFlaky, "alpha flaky test debe existir");
assert.equal(alphaFlaky.status, "flaky", "alpha test recuperado en retry debe tener estado flaky");
assert.ok(alphaFlaky.failure?.message, "alpha flaky debe conservar diagnóstico del fallo inicial");

// 4. Fallido permanente en beta
const betaFail = tests.find(t => t.id.includes("beta") && t.name.includes("flaky or fail"));
assert.ok(betaFail, "beta failed test debe existir");
assert.equal(betaFail.status, "failed", "beta test debe fallar");
assert.ok(betaFail.failure?.message.includes("99"), "beta fail debe conservar diagnóstico de fallo");

// 5. Métricas y prevalencia
assert.equal(state.stats.passed, 2, "conteo de passed debe ser 2");
assert.equal(state.stats.failed, 1, "conteo de failed debe ser 1");
assert.equal(state.stats.flaky, 1, "conteo de flaky debe ser 1");
assert.equal(state.stats.running, 0, "ningún test en running");
assert.equal(suite.status, "failed", "el fallo permanente prevalece en el estado de la suite");
assert.equal(state.status, "failed", "el fallo permanente prevalece en el estado global");

console.log("PASS TestHCD027PlaywrightIntegration suite real");
`)
	t.Log(out)

	// 3. Caso de interrupción real con Playwright
	projectInterrupt := t.TempDir()
	testsDirInt := filepath.Join(projectInterrupt, "tests")
	_ = os.MkdirAll(testsDirInt, 0755)
	writeFile(t, filepath.Join(projectInterrupt, "playwright.config.js"), `module.exports = { testDir: "./tests" };`)
	writeFile(t, filepath.Join(testsDirInt, "hang.spec.js"), `
const { test } = require(`+jsStr(testModulePath)+`);
test("hanging test", async () => {
  await new Promise(r => setTimeout(r, 30000));
});
`)

	outInterrupt := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import assert from "node:assert/strict";
`+tddToolHarness+`

const project = `+jsStr(projectInterrupt)+`;
const ac = new AbortController();
const timeout = setTimeout(() => ac.abort(), 1200);

const result = await tool.execute("c_int", { suite: "playwright" }, ac.signal, () => {}, { cwd: project });
clearTimeout(timeout);

const state = result.details;
const suite = suiteOf(state, "playwright");
assert.ok(suite, "suite playwright debe existir");
assert.equal(state.stats.running, 0, "interrupción no debe dejar running");
assert.ok(suite.tests.every(t => t.status !== "passed"), "interrupción no debe aprobar pruebas");
assert.equal(state.status, "failed", "estado debe ser fallido tras interrupción");

console.log("PASS TestHCD027PlaywrightIntegration interrupción real");
`)
	t.Log(outInterrupt)
}
