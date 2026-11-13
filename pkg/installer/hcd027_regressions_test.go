package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Each subtest asserts the corrected behavior against a synthetic reproduction.
// Missing executables and setup errors must not be mistaken for reproduced defects.
func TestHCD027Reproductions(t *testing.T) {
	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{"C001_marked_skill_preserves_added_file", hcd027ReproMarkedSkill},
		{"C002_mcp_preserves_private_mode", hcd027ReproMCPMode},
		{"C003_timeout_cannot_report_success", hcd027ReproTimeout},
		{"C004_playwright_projects_stay_distinct", hcd027ReproPlaywrightProjects},
		{"C005_interrupted_go_is_terminal", hcd027ReproGoInterrupted},
		{"C006_interrupted_playwright_is_not_passed", hcd027ReproPlaywrightInterrupted},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func hcd027ReproMarkedSkill(t *testing.T) {
	home := t.TempDir()
	piDir := filepath.Join(home, ".pi", "agent")
	legacy := filepath.Join(piDir, "skills", "sample")
	content := []byte("# Synthetic skill\n")
	writeFile(t, filepath.Join(legacy, "SKILL.md"), string(content))
	writeFile(t, filepath.Join(legacy, skillMarkerFile), contentHash(content)+"\n")
	extra := filepath.Join(legacy, "scripts", "mine.sh")
	want := []byte("echo synthetic-user-content\n")
	writeFile(t, extra, string(want))
	if err := RegisterSkillRepo(piDir, home); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(extra)
	if err != nil {
		t.Fatalf("C001: user-added script was deleted: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("C001: user-added script was changed: %q", got)
	}
}

func hcd027ReproMCPMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	const config = `{"mcpServers":{"foreign":{"command":"synthetic","env":{"API_KEY":"SYNTH-TOKEN-0001"}}}}`
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	if err := InstallMCPServers(dir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("C002: private MCP permissions widened: got %04o, want 0600", got)
	}
}

func hcd027ReproTimeout(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import assert from "node:assert/strict";
import { streamLines } from "./gomemory-tdd.ts";
const lines = [];
const child = "process.on('SIGTERM', () => process.exit(0)); console.log('ready'); setInterval(() => {}, 1000);";
const result = await streamLines(process.execPath, ["-e", child], { timeoutMs: 2000 }, line => lines.push(line));
assert.ok(lines.includes("ready"), "fixture must install its SIGTERM handler before the deadline");
assert.notEqual(result.code, 0, "C003: timeout was incorrectly accepted as success");
console.log("C003: timeout is unsuccessful");
`)
	t.Log(out)
}

func hcd027ReproPlaywrightProjects(t *testing.T) {
	project, bin := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(project, "playwright.config.ts"), "export default {}")
	writeExec(t, filepath.Join(bin, "npx"), `printf '%s\n' \
  '[1/2] [chromium] › tests/demo.spec.ts:1:1 › example' \
  '[2/2] [firefox] › tests/demo.spec.ts:1:1 › example' \
  '  2 passed (1s)'

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
          "title": "example",
          "file": "tests/demo.spec.ts",
          "line": 1,
          "column": 1,
          "tests": [
            {
              "projectName": "chromium",
              "status": "expected",
              "results": [{"status": "passed", "duration": 10}]
            },
            {
              "projectName": "firefox",
              "status": "expected",
              "results": [{"status": "passed", "duration": 12}]
            }
          ]
        }
      ]
    }
  ],
  "stats": {"expected": 2}
}
EOF
fi
exit 0`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"},
		`process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import assert from "node:assert/strict";
`+tddToolHarness+`
const result = await run("playwright", `+jsStr(project)+`);
const suite = suiteOf(result.details, "playwright");
assert.ok(suite, "playwright suite must exist");
assert.equal(suite.tests.length, 2, "C004: two projects at the same location collapsed into one result");
assert.notEqual(suite.tests[0].id, suite.tests[1].id, "C004: projects must have distinct identities");
assert.ok(suite.tests.every(test => test.status === "passed"), "C004: both completed projects must pass");
console.log("C004: distinct projects retain their results");
`)
	t.Log(out)
}

func hcd027ReproGoInterrupted(t *testing.T) {
	project, bin := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), "module synthetic\n")
	writeExec(t, filepath.Join(bin, "go"), `echo '{"Action":"run","Package":"synthetic","Test":"TestInterrupted"}'
exit 1`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"},
		`process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import assert from "node:assert/strict";
`+tddToolHarness+`
const result = await run("backend", `+jsStr(project)+`);
const state = result.details;
const test = suiteOf(state, "backend")?.tests.find(test => test.name === "TestInterrupted");
assert.ok(test, "started Go test must exist");
assert.equal(test.status, "failed", "C005: interrupted Go test must not remain running");
assert.equal(state.stats.running, 0, "C005: terminated suite must have no running tests");
assert.equal(state.status, "failed", "C005: global state must reflect failure");
assert.ok(test.failure?.message, "C005: interrupted test must have a diagnostic");
console.log("C005: interrupted Go test is terminal");
`)
	t.Log(out)
}

func hcd027ReproPlaywrightInterrupted(t *testing.T) {
	project, bin := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(project, "playwright.config.ts"), "export default {}")
	writeExec(t, filepath.Join(bin, "npx"), `printf '%s\n' \
  '[1/2] tests/demo.spec.ts:1:1 › fails' \
  '[2/2] tests/demo.spec.ts:2:1 › interrupted' \
  '  1) tests/demo.spec.ts:1:1 › fails ─────' \
  '    Error: synthetic failure'
exit 1`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"},
		`process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import assert from "node:assert/strict";
`+tddToolHarness+`
const result = await run("playwright", `+jsStr(project)+`);
const suite = suiteOf(result.details, "playwright");
const interrupted = suite?.tests.find(test => test.name.includes("interrupted"));
assert.ok(interrupted, "interrupted Playwright test must exist");
assert.equal(interrupted.status, "failed", "C006: unfinished Playwright test was falsely approved");
assert.equal(result.details.stats.running, 0, "C006: no tests may remain running");
const failed = suite.tests.find(test => test.name === "fails");
assert.ok(failed?.failure?.message.includes("synthetic failure"), "C006: original failure diagnostic must survive");
console.log("C006: no approval without a terminal result");
`)
	t.Log(out)
}

func TestHCD027SkillCleanupCompatibility(t *testing.T) {
	home := t.TempDir()
	piDir := filepath.Join(home, ".pi", "agent")
	skillsDir := filepath.Join(piDir, "skills")
	repoDir := filepath.Join(home, ".pi-gomemory", "skills")

	// 1. Managed pristine skill with only SKILL.md and .pi-gomemory
	pristineSkill := filepath.Join(skillsDir, "managed_pristine")
	pristineContent := []byte("# Pristine Skill\n")
	writeFile(t, filepath.Join(pristineSkill, "SKILL.md"), string(pristineContent))
	writeFile(t, filepath.Join(pristineSkill, skillMarkerFile), contentHash(pristineContent)+"\n")

	// 2. Managed unmarked skill matching central repo SKILL.md with only SKILL.md
	repoSkill := filepath.Join(repoDir, "managed_unmarked")
	repoContent := []byte("# Repo Skill\n")
	writeFile(t, filepath.Join(repoSkill, "SKILL.md"), string(repoContent))
	unmarkedSkill := filepath.Join(skillsDir, "managed_unmarked")
	writeFile(t, filepath.Join(unmarkedSkill, "SKILL.md"), string(repoContent))

	// 3. User custom skill (not in repo, not marked)
	customSkill := filepath.Join(skillsDir, "user_custom")
	customContent := []byte("# User Custom Skill\n")
	writeFile(t, filepath.Join(customSkill, "SKILL.md"), string(customContent))
	writeFile(t, filepath.Join(customSkill, "custom_file.txt"), "user payload")

	// 4. Host skill symlink pointing to an external directory
	hostTargetDir := filepath.Join(home, ".claude", "skills", "host_skill")
	hostContent := []byte("# Host Skill\n")
	writeFile(t, filepath.Join(hostTargetDir, "SKILL.md"), string(hostContent))
	writeFile(t, filepath.Join(hostTargetDir, "tool.sh"), "#!/bin/sh\n")
	symlinkPath := filepath.Join(skillsDir, "host_link")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(hostTargetDir, symlinkPath); err != nil {
		t.Fatal(err)
	}

	// 5. User custom folder in repoDir (not symlink)
	userRepoDir := filepath.Join(repoDir, "my_custom_repo_skill")
	writeFile(t, filepath.Join(userRepoDir, "SKILL.md"), "# Repo Custom\n")

	// Run RegisterSkillRepo
	if err := RegisterSkillRepo(piDir, home); err != nil {
		t.Fatal(err)
	}

	// 1: Pristine managed copy must be pruned
	if _, err := os.Stat(pristineSkill); !os.IsNotExist(err) {
		t.Errorf("managed pristine skill was not cleaned up: %v", err)
	}

	// 2: Unmarked matching legacy copy must be pruned
	if _, err := os.Stat(unmarkedSkill); !os.IsNotExist(err) {
		t.Errorf("unmarked legacy skill was not cleaned up: %v", err)
	}

	// 3: User custom skill must NOT be deleted
	if _, err := os.Stat(customSkill); err != nil {
		t.Errorf("user custom skill was deleted: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(customSkill, "custom_file.txt")); err != nil || string(got) != "user payload" {
		t.Errorf("user custom file was lost: %s, %v", string(got), err)
	}

	// 4: Host target directory MUST NOT be deleted (only legacy symlink is pruned from piDir)
	if _, err := os.Stat(hostTargetDir); err != nil {
		t.Errorf("host skill target directory was deleted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(hostTargetDir, "tool.sh")); err != nil {
		t.Errorf("host skill file was deleted: %v", err)
	}

	// 5: User folder in repoDir must NOT be deleted
	if _, err := os.Stat(userRepoDir); err != nil {
		t.Errorf("user repo directory was deleted: %v", err)
	}
}
