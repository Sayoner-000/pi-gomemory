package installer

import "testing"

// C-03/C-04: splash_review no lee fuera de ctx.cwd (.., absoluta, enlace) y
// acepta nombres internos que empiezan por "..".
func TestHCD012Splash(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-splash.ts"}, `
import { mkdtempSync, mkdirSync, writeFileSync, symlinkSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { calls } from "@earendil-works/pi-coding-agent";
import ext from "./gomemory-splash.ts";

let tool;
ext({ registerTool: (t) => { tool = t; } });
const root = mkdtempSync(join(tmpdir(), "splash-"));
const repo = join(root, "repo"); mkdirSync(repo);
writeFileSync(join(root, "secret.txt"), "s");
writeFileSync(join(repo, "..notas.md"), "ok");
writeFileSync(join(repo, "doc.md"), "ok");
symlinkSync(join(root, "secret.txt"), join(repo, "link.md"));
const ctx = { cwd: repo, model: undefined };

async function run(document) {
  try { await tool.execute("id", { document }, undefined, undefined, ctx); return "ok"; }
  catch (e) { return String(e.message); }
}
const fails = [];
for (const d of ["../secret.txt", join(root, "secret.txt"), "link.md"]) {
  const r = await run(d);
  if (!/fuera del repositorio/.test(r)) fails.push(d + " => " + r);
}
for (const d of ["..notas.md", "doc.md"]) {
  const r = await run(d);
  if (!/selected model/.test(r)) fails.push(d + " => " + r);
}
if (calls.length) fails.push("se contactó al proveedor: " + calls);
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if out != "PASS" {
		t.Fatal(out)
	}
}
