package installer

import (
	"path/filepath"
	"testing"
)

func TestHCD027PlaywrightIdentity(t *testing.T) {
	project, bin := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(project, "playwright.config.ts"), "export default {}")

	// 1. Two projects at the same location, one fails, one passes
	// Plus a retry that must not create a duplicate test
	writeExec(t, filepath.Join(bin, "npx"), `printf '%s\n' \
  '[1/3] [chromium] › tests/demo.spec.ts:1:1 › multi' \
  '  1) [chromium] › tests/demo.spec.ts:1:1 › multi ─────' \
  '    Error: chromium failure' \
  '[2/3] [firefox] › tests/demo.spec.ts:1:1 › multi' \
  '[3/3] [webkit] › tests/retry.spec.ts:5:1 › withRetry' \
  '  2) [webkit] › tests/retry.spec.ts:5:1 › withRetry ─────' \
  '    Error: first attempt failure' \
  '[3/3] [webkit] › tests/retry.spec.ts:5:1 › withRetry (retry #1)' \
  '  1 failed, 1 passed, 1 flaky (2s)'

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
          "title": "multi",
          "file": "tests/demo.spec.ts",
          "line": 1,
          "column": 1,
          "tests": [
            {
              "projectName": "chromium",
              "status": "unexpected",
              "results": [{
                "status": "failed",
                "duration": 20,
                "errors": [{"message": "Error: chromium failure"}]
              }]
            },
            {
              "projectName": "firefox",
              "status": "expected",
              "results": [{
                "status": "passed",
                "duration": 25,
                "errors": []
              }]
            }
          ]
        }
      ]
    },
    {
      "file": "tests/retry.spec.ts",
      "line": 5,
      "column": 1,
      "specs": [
        {
          "title": "withRetry",
          "file": "tests/retry.spec.ts",
          "line": 5,
          "column": 1,
          "tests": [
            {
              "projectName": "webkit",
              "status": "flaky",
              "results": [
                {
                  "status": "failed",
                  "duration": 15,
                  "errors": [{"message": "Error: first attempt failure"}]
                },
                {
                  "status": "passed",
                  "duration": 18,
                  "errors": []
                }
              ]
            }
          ]
        }
      ]
    }
  ],
  "stats": {"expected": 1, "unexpected": 1, "flaky": 1}
}
EOF
fi
exit 1`)

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"},
		`process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import assert from "node:assert/strict";
`+tddToolHarness+`
const result = await run("playwright", `+jsStr(project)+`);
const suite = suiteOf(result.details, "playwright");
assert.ok(suite, "playwright suite must exist");

// Assertion 1: Two projects at same location produce two distinct results
const chromiumTest = suite.tests.find(t => t.id.includes("chromium") && t.name.includes("multi"));
const firefoxTest = suite.tests.find(t => t.id.includes("firefox") && t.name.includes("multi"));
assert.ok(chromiumTest, "chromium test must exist");
assert.ok(firefoxTest, "firefox test must exist");
assert.notEqual(chromiumTest.id, firefoxTest.id, "projects must have distinct identities");
assert.equal(chromiumTest.status, "failed", "chromium test must fail");
assert.ok(chromiumTest.failure?.message.includes("chromium failure"), "chromium diagnostic must be preserved");
assert.equal(firefoxTest.status, "passed", "firefox test must pass");

// Assertion 2: Retry does NOT create another test
const retryTests = suite.tests.filter(t => t.name.includes("withRetry"));
assert.equal(retryTests.length, 1, "retry must not create another test; found: " + retryTests.length);

console.log("PASS TestHCD027PlaywrightIdentity");
`)
	t.Log(out)
}

func TestHCD027PlaywrightTerminalResults(t *testing.T) {
	// Case 1: Valid JSON report determines terminal states and cleans up temp file
	{
		project, bin := t.TempDir(), t.TempDir()
		writeFile(t, filepath.Join(project, "playwright.config.ts"), "export default {}")

		writeExec(t, filepath.Join(bin, "npx"), `printf '%s\n' \
  '[1/2] tests/spec.ts:1:1 › testPass' \
  '[2/2] tests/spec.ts:2:1 › testFail' \
  '  1) tests/spec.ts:2:1 › testFail ─────' \
  '    Error: fail line diagnostic'

if [ -n "$PLAYWRIGHT_JSON_OUTPUT_NAME" ]; then
  cat << 'EOF' > "$PLAYWRIGHT_JSON_OUTPUT_NAME"
{
  "suites": [
    {
      "file": "tests/spec.ts",
      "line": 1,
      "column": 1,
      "specs": [
        {
          "title": "testPass",
          "file": "tests/spec.ts",
          "line": 1,
          "column": 1,
          "tests": [{ "status": "expected", "results": [{ "status": "passed", "duration": 50 }] }]
        },
        {
          "title": "testFail",
          "file": "tests/spec.ts",
          "line": 2,
          "column": 1,
          "tests": [{ "status": "unexpected", "results": [{ "status": "failed", "duration": 30, "errors": [{ "message": "Error: fail line diagnostic" }] }] }]
        }
      ]
    }
  ],
  "stats": { "expected": 1, "unexpected": 1 }
}
EOF
fi
exit 1`)

		out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"},
			`process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
`+tddToolHarness+`
const result = await run("playwright", `+jsStr(project)+`);
const suite = suiteOf(result.details, "playwright");
assert.ok(suite, "suite must exist");

const pass = suite.tests.find(t => t.name.includes("testPass"));
assert.equal(pass?.status, "passed", "valid JSON confirms passed test");
assert.equal(pass?.durationMs, 50, "duration from JSON preserved");

const fail = suite.tests.find(t => t.name.includes("testFail"));
assert.equal(fail?.status, "failed", "valid JSON confirms failed test");
assert.ok(fail?.failure?.message.includes("fail line diagnostic"), "failure diagnostic preserved");

assert.equal(result.details.stats.running, 0, "no tests running");

// Verify no residual .pw-report temp files left in os.tmpdir()
const residual = fs.readdirSync(os.tmpdir()).filter(f => f.startsWith(".pw-report-"));
assert.equal(residual.length, 0, "temporary JSON files must be cleaned up");

console.log("PASS TestHCD027PlaywrightTerminalResults case 1");
`)
		t.Log(out)
	}

	// Case 2: Interrupted / missing JSON must NOT falsely approve tests and must leave no running tests
	{
		project, bin := t.TempDir(), t.TempDir()
		writeFile(t, filepath.Join(project, "playwright.config.ts"), "export default {}")

		// Process streams line reporter but crashes / exits without writing JSON
		writeExec(t, filepath.Join(bin, "npx"), `printf '%s\n' \
  '[1/2] tests/spec.ts:1:1 › testStartedNoJson' \
  '[2/2] tests/spec.ts:2:1 › testInterrupted'
exit 1`)

		out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"},
			`process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import assert from "node:assert/strict";
`+tddToolHarness+`
const result = await run("playwright", `+jsStr(project)+`);
const suite = suiteOf(result.details, "playwright");
assert.ok(suite, "suite must exist");

// Crucial invariant: missing JSON must NOT approve tests
assert.ok(suite.tests.every(t => t.status !== "passed"), "missing JSON must not falsely approve tests");
assert.ok(suite.tests.every(t => t.status === "failed"), "started tests without JSON must be closed as failed");
assert.equal(result.details.stats.running, 0, "interrupted execution must not leave running tests");
assert.equal(result.details.status, "failed", "suite must be failed");

console.log("PASS TestHCD027PlaywrightTerminalResults case 2");
`)
		t.Log(out)
	}
}

func TestHCD027FlakyContract(t *testing.T) {
	// Case 1: Pure flaky test - counts in total and flaky, NOT in passed, failed, or queued
	// Initial diagnostic preserved, suite and global status is flaky
	{
		project, bin := t.TempDir(), t.TempDir()
		writeFile(t, filepath.Join(project, "playwright.config.ts"), "export default {}")

		writeExec(t, filepath.Join(bin, "npx"), `printf '%s\n' \
  '[1/1] [webkit] › tests/flaky.spec.ts:10:5 › unstable' \
  '  1) [webkit] › tests/flaky.spec.ts:10:5 › unstable ─────' \
  '    Error: transient network glitch' \
  '[1/1] [webkit] › tests/flaky.spec.ts:10:5 › unstable (retry #1)' \
  '  1 flaky (3s)'

if [ -n "$PLAYWRIGHT_JSON_OUTPUT_NAME" ]; then
  cat << 'EOF' > "$PLAYWRIGHT_JSON_OUTPUT_NAME"
{
  "suites": [
    {
      "file": "tests/flaky.spec.ts",
      "line": 10,
      "column": 5,
      "specs": [
        {
          "title": "unstable",
          "file": "tests/flaky.spec.ts",
          "line": 10,
          "column": 5,
          "tests": [
            {
              "projectName": "webkit",
              "status": "flaky",
              "results": [
                {
                  "status": "failed",
                  "duration": 40,
                  "errors": [{"message": "Error: transient network glitch"}]
                },
                {
                  "status": "passed",
                  "duration": 45,
                  "errors": []
                }
              ]
            }
          ]
        }
      ]
    }
  ],
  "stats": {"expected": 0, "unexpected": 0, "flaky": 1}
}
EOF
fi
exit 0`)

		out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"},
			`process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import assert from "node:assert/strict";
import { formatAgentReport, renderTDDOverlay, renderTDDWidget, renderTDDMinimized } from "./gomemory-tdd.ts";
`+tddToolHarness+`
const result = await run("playwright", `+jsStr(project)+`);
const state = result.details;
const suite = suiteOf(state, "playwright");
assert.ok(suite, "suite must exist");

// Q2 requirements:
// 1. Un único resultado flaky
assert.equal(suite.tests.length, 1, "exactly one test result");
const test = suite.tests[0];
assert.equal(test.status, "flaky", "status must be flaky");

// 2. Diagnóstico inicial conservado en su detalle
assert.ok(test.failure?.message.includes("transient network glitch"), "initial diagnostic must be preserved in failure.message");

// 3. Contador flaky separado (no passed, failed ni queued)
assert.equal(state.stats.total, 1, "counted in total");
assert.equal(state.stats.flaky, 1, "counted in flaky");
assert.equal(state.stats.passed, 0, "must NOT count in passed");
assert.equal(state.stats.failed, 0, "must NOT count in failed");
assert.equal(state.stats.queued, 0, "must NOT count in queued");

// 4. Suite y estado global flaky si no hay fallos definitivos
assert.equal(suite.status, "flaky", "suite status must be flaky");
assert.equal(state.status, "flaky", "global state status must be flaky");

// 5. Informe para el agente refleja flaky y diagnóstico inicial
const report = formatAgentReport(state, ["playwright"]);
assert.ok(report.includes("flaky"), "report must mention flaky");
assert.ok(report.includes("transient network glitch"), "report must show initial failure diagnostic");

// 6. Componentes visuales TUI
const fakeTheme = {
	fg: (_, s) => s,
	style: (s) => s,
};
const overlay = renderTDDOverlay(state, 60, fakeTheme).join("\n");
assert.ok(overlay.includes("~") || overlay.includes("flaky"), "overlay must show flaky marker");
assert.ok(overlay.includes("transient network glitch"), "overlay must display initial failure headline");

const widget = renderTDDWidget(state, fakeTheme).join("\n");
assert.ok(widget.includes("~"), "widget must show flaky count");

const mini = renderTDDMinimized(state, 60, fakeTheme).join("\n");
assert.ok(mini.includes("~"), "minimized view must show flaky count");

console.log("PASS TestHCD027FlakyContract case 1");
`)
		t.Log(out)
	}

	// Case 2: Permanent failure takes precedence over flaky
	{
		project, bin := t.TempDir(), t.TempDir()
		writeFile(t, filepath.Join(project, "playwright.config.ts"), "export default {}")

		writeExec(t, filepath.Join(bin, "npx"), `printf '%s\n' \
  '[1/2] [chromium] › tests/a.spec.ts:1:1 › flakyTest' \
  '[1/2] [chromium] › tests/a.spec.ts:1:1 › flakyTest (retry #1)' \
  '[2/2] [chromium] › tests/b.spec.ts:1:1 › permanentFail' \
  '  1) [chromium] › tests/b.spec.ts:1:1 › permanentFail ─────' \
  '    Error: permanent defect' \
  '  1 failed, 1 flaky (2s)'

if [ -n "$PLAYWRIGHT_JSON_OUTPUT_NAME" ]; then
  cat << 'EOF' > "$PLAYWRIGHT_JSON_OUTPUT_NAME"
{
  "suites": [
    {
      "file": "tests/a.spec.ts",
      "line": 1,
      "column": 1,
      "specs": [{
        "title": "flakyTest",
        "file": "tests/a.spec.ts",
        "line": 1,
        "column": 1,
        "tests": [{
          "projectName": "chromium",
          "status": "flaky",
          "results": [
            { "status": "failed", "duration": 10, "errors": [{"message": "transient"}] },
            { "status": "passed", "duration": 10, "errors": [] }
          ]
        }]
      }]
    },
    {
      "file": "tests/b.spec.ts",
      "line": 1,
      "column": 1,
      "specs": [{
        "title": "permanentFail",
        "file": "tests/b.spec.ts",
        "line": 1,
        "column": 1,
        "tests": [{
          "projectName": "chromium",
          "status": "unexpected",
          "results": [
            { "status": "failed", "duration": 20, "errors": [{"message": "Error: permanent defect"}] }
          ]
        }]
      }]
    }
  ],
  "stats": {"expected": 0, "unexpected": 1, "flaky": 1}
}
EOF
fi
exit 1`)

		out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"},
			`process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import assert from "node:assert/strict";
`+tddToolHarness+`
const result = await run("playwright", `+jsStr(project)+`);
const state = result.details;
const suite = suiteOf(state, "playwright");
assert.ok(suite, "suite must exist");

// Fallo definitivo prevalece sobre flaky al terminar
assert.equal(state.stats.flaky, 1, "stats.flaky is 1");
assert.equal(state.stats.failed, 1, "stats.failed is 1");
assert.equal(suite.status, "failed", "suite status must be failed because permanent failure prevails");
assert.equal(state.status, "failed", "global state status must be failed because permanent failure prevails");

console.log("PASS TestHCD027FlakyContract case 2");
`)
		t.Log(out)
	}
}
