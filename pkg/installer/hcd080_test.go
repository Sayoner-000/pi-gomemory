package installer

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// hcd080StatusHarness carga gomemory-status.ts con un pi falso: `skills-sync
// --check` sale con DRIFT (10 + lista) la primera vez y con 0 tras sincronizar.
const hcd080StatusHarness = `
import ext from "./gomemory-status.ts";
const handlers = {}, execs = [], notes = [];
let drifted = true;
const pi = {
	on: (name, fn) => { handlers[name] = fn; },
	registerCommand: () => {}, registerShortcut: () => {},
	exec: async (cmd, args) => {
		const line = cmd + " " + args.join(" ");
		execs.push(line);
		if (line === "pi-gomemory-setup skills-sync --check") {
			return drifted
				? { code: 10, stdout: "", stderr: "skills de gomemory desfasadas en ~/.pi-gomemory/skills: hcd-implement, hcd-verify. Ejecuta pi-gomemory-setup skills-sync\n" }
				: { code: 0, stdout: "", stderr: "" };
		}
		if (line === "pi-gomemory-setup skills-sync") { drifted = false; return { code: 0, stdout: "", stderr: "" }; }
		return { code: 0, stdout: "", stderr: "" };
	},
};
const ctx = { hasUI: true, ui: { notify: (m, l) => notes.push([l, m]), setStatus() {}, setFooter() {}, setWorkingIndicator() {}, setHiddenThinkingLabel() {} } };
delete process.env.PI_GOMEMORY_NO_UPDATE_CHECK;
ext(pi);
const settle = () => new Promise((r) => setTimeout(r, 50));
const skillExecs = () => execs.filter((e) => e.includes("skills-sync"));
`

func hcd080Lines(out string) map[string]string {
	lines := map[string]string{}
	for _, l := range strings.Split(out, "\n") {
		if k, v, ok := strings.Cut(l, "="); ok {
			lines[k] = v
		}
	}
	return lines
}

// HCD-080 tarea 3: al iniciar la sesión, si `skills-sync --check` sale con 10,
// la extensión sincroniza y avisa (con los nombres) de que /reload las carga;
// si está al día, no sincroniza ni avisa.
func TestHCD080StatusSessionStart(t *testing.T) {
	t.Parallel()
	out := runNodeExt(t, []string{"extensions/gomemory-status.ts"}, hcd080StatusHarness+`
await handlers.session_start({}, ctx); await settle();
console.log("FIRST=" + JSON.stringify(skillExecs()));
console.log("NOTES=" + JSON.stringify(notes));
execs.length = 0; notes.length = 0;
await handlers.session_start({}, ctx); await settle();
console.log("SECOND=" + JSON.stringify(skillExecs()));
console.log("NOTES2=" + JSON.stringify(notes));
// HCD-037 C-02: con PI_GOMEMORY_NO_UPDATE_CHECK=1 no se consulta nada.
execs.length = 0; drifted = true; process.env.PI_GOMEMORY_NO_UPDATE_CHECK = "1";
await handlers.session_start({}, ctx); await handlers.tool_result({ toolName: "bash", input: { command: "mem update" } }, ctx); await settle();
console.log("DISABLED=" + JSON.stringify(skillExecs()));
`)
	l := hcd080Lines(out)
	if l["FIRST"] != `["pi-gomemory-setup skills-sync --check","pi-gomemory-setup skills-sync"]` {
		t.Fatalf("con deriva debe comprobar y sincronizar: %s\n%s", l["FIRST"], out)
	}
	if !strings.Contains(l["NOTES"], "hcd-verify") || !strings.Contains(l["NOTES"], "/reload") {
		t.Fatalf("debe avisar con los nombres y /reload: %s", l["NOTES"])
	}
	if l["SECOND"] != `["pi-gomemory-setup skills-sync --check"]` || l["NOTES2"] != "[]" {
		t.Fatalf("al día solo comprueba y calla: %s %s", l["SECOND"], l["NOTES2"])
	}
	if l["DISABLED"] != "[]" {
		t.Fatalf("con PI_GOMEMORY_NO_UPDATE_CHECK=1 no debe comprobar: %s", l["DISABLED"])
	}
}

// HCD-080 tarea 4: un bash que ejecutó `mem update` vuelve a comprobar las
// skills; otros comandos o un `mem update` fallido no.
func TestHCD080StatusAfterMemUpdate(t *testing.T) {
	t.Parallel()
	out := runNodeExt(t, []string{"extensions/gomemory-status.ts"}, hcd080StatusHarness+`
const bash = (command, isError = false) => handlers.tool_result?.({ toolName: "bash", input: { command }, isError }, ctx);
await bash("go test ./..."); await settle();
await bash("mem update -y", true); await settle();
console.log("OTHER=" + JSON.stringify(skillExecs()));
await bash("cd /tmp && mem update -y"); await settle();
console.log("UPDATE=" + JSON.stringify(skillExecs()));
console.log("NOTES=" + JSON.stringify(notes));
`)
	l := hcd080Lines(out)
	if l["OTHER"] != "[]" {
		t.Fatalf("otros comandos o un mem update fallido no comprueban: %s\n%s", l["OTHER"], out)
	}
	if l["UPDATE"] != `["pi-gomemory-setup skills-sync --check","pi-gomemory-setup skills-sync"]` || !strings.Contains(l["NOTES"], "/reload") {
		t.Fatalf("tras mem update debe comprobar, sincronizar y avisar: %s %s", l["UPDATE"], l["NOTES"])
	}
}

// HCD-080 tarea 12 (Q1): con solo copias editadas, la extensión no sincroniza
// (no las tocaría) y avisa en amarillo con el comando --restore; con deriva y
// editadas, sincroniza, avisa de /reload y también de las editadas.
func TestHCD080StatusEditedWarning(t *testing.T) {
	t.Parallel()
	out := runNodeExt(t, []string{"extensions/gomemory-status.ts"}, `
import ext from "./gomemory-status.ts";
const handlers = {}, execs = [], notes = [];
const EDITED = "skills de gomemory editadas localmente y desfasadas respecto a mem: hcd-verify. Ejecuta pi-gomemory-setup skills-sync --restore hcd-verify\n";
const DRIFT = "skills de gomemory desfasadas en ~/.pi-gomemory/skills: hcd-close. Ejecuta pi-gomemory-setup skills-sync\n";
let stderr = EDITED;
const pi = {
	on: (n, fn) => { handlers[n] = fn; }, registerCommand() {}, registerShortcut() {},
	exec: async (cmd, args) => {
		const line = cmd + " " + args.join(" ");
		if (line.includes("skills-sync")) execs.push(line);
		if (line.endsWith("skills-sync --check")) return { code: 10, stdout: "", stderr };
		return { code: 0, stdout: "", stderr: "" };
	},
};
const ctx = { hasUI: true, ui: { notify: (m, l) => notes.push([l, m]), setStatus() {}, setFooter() {}, setWorkingIndicator() {}, setHiddenThinkingLabel() {} } };
delete process.env.PI_GOMEMORY_NO_UPDATE_CHECK;
ext(pi);
const settle = () => new Promise((r) => setTimeout(r, 50));
await handlers.session_start({}, ctx); await settle();
console.log("ONLY_EXECS=" + JSON.stringify(execs));
console.log("ONLY_NOTES=" + JSON.stringify(notes));
execs.length = 0; notes.length = 0; stderr = DRIFT + EDITED;
await handlers.session_start({}, ctx); await settle();
console.log("MIX_EXECS=" + JSON.stringify(execs));
console.log("MIX_NOTES=" + JSON.stringify(notes));
`)
	l := hcd080Lines(out)
	if l["ONLY_EXECS"] != `["pi-gomemory-setup skills-sync --check"]` {
		t.Fatalf("solo editadas: no debe sincronizar: %s\n%s", l["ONLY_EXECS"], out)
	}
	if !strings.Contains(l["ONLY_NOTES"], `"warning"`) || !strings.Contains(l["ONLY_NOTES"], "skills-sync --restore hcd-verify") || strings.Contains(l["ONLY_NOTES"], "/reload") {
		t.Fatalf("solo editadas: aviso warning con --restore y sin /reload: %s", l["ONLY_NOTES"])
	}
	if l["MIX_EXECS"] != `["pi-gomemory-setup skills-sync --check","pi-gomemory-setup skills-sync"]` {
		t.Fatalf("con deriva debe sincronizar: %s", l["MIX_EXECS"])
	}
	if !strings.Contains(l["MIX_NOTES"], "hcd-close") || !strings.Contains(l["MIX_NOTES"], "/reload") || !strings.Contains(l["MIX_NOTES"], "--restore hcd-verify") || strings.Contains(l["MIX_NOTES"], "actualizadas desde mem (hcd-close, hcd-verify") {
		t.Fatalf("mixto: info de hcd-close con /reload y aviso de hcd-verify: %s", l["MIX_NOTES"])
	}
}

// HCD-080 tarea 9 (Q1): SkillRepoEdited lista las copias que instaló
// pi-gomemory (con marca) y luego se editaron, si difieren de la vigente. Una
// editada que ya coincide, o una carpeta propia sin marca, no cuentan.
func TestHCD080SkillRepoEdited(t *testing.T) {
	home := t.TempDir()
	assets := hcd073Assets()
	assets["skills/hcd-define/SKILL.md"] = assets["skills/hcd-close/SKILL.md"]
	memVerify := filepath.Join(home, ".claude", "skills", "hcd-verify", "SKILL.md")
	hcd073Write(t, memVerify, "---\nname: hcd-verify\ndescription: mem 2.59\n---\nv2\n")
	if _, err := SyncSkillRepo(assets, home); err != nil {
		t.Fatal(err)
	}
	repo := SkillRepoDir(home)
	if ed, err := SkillRepoEdited(assets, home); err != nil || len(ed) != 0 {
		t.Fatalf("sin ediciones: %v %v", ed, err)
	}
	// hcd-verify: editada y distinta. hcd-close: marca que no cuadra pero
	// contenido igual al vigente. hcd-define: carpeta propia sin marca.
	hcd073Write(t, filepath.Join(repo, "hcd-verify", "SKILL.md"), "apaño local\n")
	want, err := wantedSkillContent(assets, repo, home, "hcd-close")
	if err != nil {
		t.Fatal(err)
	}
	hcd073Write(t, filepath.Join(repo, "hcd-close", skillMarkerFile), "otra-marca\n")
	hcd073Write(t, filepath.Join(repo, "hcd-close", "SKILL.md"), string(want))
	if err := os.Remove(filepath.Join(repo, "hcd-define", skillMarkerFile)); err != nil {
		t.Fatal(err)
	}
	hcd073Write(t, filepath.Join(repo, "hcd-define", "SKILL.md"), "mía\n")

	ed, err := SkillRepoEdited(assets, home)
	if err != nil || !slices.Equal(ed, []string{"hcd-verify"}) {
		t.Fatalf("editadas = %v (%v), quiero solo hcd-verify", ed, err)
	}
	if drift, _ := SkillRepoDrift(assets, home); len(drift) != 0 {
		t.Fatalf("las editadas no son deriva sincronizable: %v", drift)
	}
}

// HCD-080 tarea 5: EmbeddedSkillDrift compara las skills embebidas en el
// binario con las copias que instaló `mem` y cuenta cuántas pudo comparar;
// sin copias de `mem` no hay nada que comparar.
func TestHCD080EmbeddedSkillDrift(t *testing.T) {
	assets := hcd073Assets()
	if drift, n := EmbeddedSkillDrift(assets, t.TempDir()); n != 0 || len(drift) != 0 {
		t.Fatalf("sin copias de mem: n=%d deriva=%v", n, drift)
	}
	home := t.TempDir()
	hcd073Write(t, filepath.Join(home, ".codex", "skills", "hcd-close", "SKILL.md"), string(assets["skills/hcd-close/SKILL.md"].Data))
	hcd073Write(t, filepath.Join(home, ".codex", "skills", "hcd-verify", "SKILL.md"), "---\nname: hcd-verify\ndescription: mem 2.59\n---\nnueva\n")
	// Una copia que resuelve dentro del repositorio central no es de mem.
	hcd073Write(t, filepath.Join(SkillRepoDir(home), "otra", "SKILL.md"), "x\n")
	drift, n := EmbeddedSkillDrift(assets, home)
	if n != 2 || !slices.Equal(drift, []string{"hcd-verify"}) {
		t.Fatalf("n=%d deriva=%v, quiero 2 comparadas y solo hcd-verify desfasada", n, drift)
	}
}

// HCD-080 tarea 1: SkillRepoDrift detecta las skills de gomemory cuya copia
// intacta en el repositorio central ya no coincide con lo que escribiría una
// sincronización (la copia de `mem` manda), o que faltan. Ignora las copias
// editadas por la persona y no escribe nada.
func TestHCD080SkillRepoDrift(t *testing.T) {
	home := t.TempDir()
	assets := hcd073Assets()
	memVerify := filepath.Join(home, ".claude", "skills", "hcd-verify", "SKILL.md")
	memClose := filepath.Join(home, ".claude", "skills", "hcd-close", "SKILL.md")
	hcd073Write(t, memVerify, "---\nname: hcd-verify\ndescription: mem 2.58\n---\nv1\n")
	hcd073Write(t, memClose, "---\nname: hcd-close\ndescription: mem 2.58\n---\nv1\n")
	if _, err := SyncSkillRepo(assets, home); err != nil {
		t.Fatal(err)
	}

	drift, err := SkillRepoDrift(assets, home)
	if err != nil || len(drift) != 0 {
		t.Fatalf("recién sincronizado no debe haber deriva: %v %v", drift, err)
	}

	// `mem update` cambia sus copias; la persona editó su hcd-close.
	repo := SkillRepoDir(home)
	hcd073Write(t, memVerify, "---\nname: hcd-verify\ndescription: mem 2.59\n---\nv2\n")
	hcd073Write(t, memClose, "---\nname: hcd-close\ndescription: mem 2.59\n---\nv2\n")
	repoClose := filepath.Join(repo, "hcd-close", "SKILL.md")
	hcd073Write(t, repoClose, "mi versión\n")
	before := hcd073Read(t, filepath.Join(repo, "hcd-verify", "SKILL.md"))

	drift, err = SkillRepoDrift(assets, home)
	if err != nil || !slices.Equal(drift, []string{"hcd-verify"}) {
		t.Fatalf("deriva = %v (%v), quiero solo hcd-verify (hcd-close está editada)", drift, err)
	}
	if hcd073Read(t, filepath.Join(repo, "hcd-verify", "SKILL.md")) != before || hcd073Read(t, repoClose) != "mi versión\n" {
		t.Fatal("SkillRepoDrift no debe escribir")
	}

	// Una skill ausente del repositorio también es deriva.
	if err := os.RemoveAll(filepath.Join(repo, "hcd-verify")); err != nil {
		t.Fatal(err)
	}
	if drift, _ = SkillRepoDrift(assets, home); !slices.Equal(drift, []string{"hcd-verify"}) {
		t.Fatalf("ausente: deriva = %v", drift)
	}

	// Tras sincronizar vuelve a estar al día y la copia editada se conserva.
	if _, err := SyncSkillRepo(assets, home); err != nil {
		t.Fatal(err)
	}
	if drift, _ = SkillRepoDrift(assets, home); len(drift) != 0 {
		t.Fatalf("tras sincronizar: deriva = %v", drift)
	}
	if hcd073Read(t, repoClose) != "mi versión\n" {
		t.Fatal("la sincronización pisó la copia editada")
	}
}
