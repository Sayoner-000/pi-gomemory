package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func hcd073Assets() fstest.MapFS {
	return fstest.MapFS{
		"skills/hcd-verify/SKILL.md": {Data: []byte("---\nname: hcd-verify\ndescription: embebida\n---\nembebida\n")},
		"skills/hcd-close/SKILL.md":  {Data: []byte("---\nname: hcd-close\ndescription: embebida\n---\nembebida\n")},
	}
}

func hcd073Write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hcd073Read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestHCD073Protocolo: codegraph es el grafo principal y codebase-memory solo
// se consulta ante huecos; el ajuste propio de Pi de hcd-verify vive en el
// protocolo y no en una copia de la skill (HCD-073, Q1=a).
func TestHCD073Protocolo(t *testing.T) {
	t.Parallel()
	out := runNodeExt(t, []string{"extensions/gomemory-context.ts", "extensions/gomemory-plan.ts", "extensions/gomemory-plan-patterns.json"}, `
import { piBlock } from "./gomemory-context.ts";
import { IMPACT_PROTOCOL } from "./gomemory-plan.ts";

const fails = [];
const lines = piBlock([]).split("\n");
const find = (re) => lines.find((l) => re.test(l)) || "";
const principal = (where, text) => {
	if (!/codegraph/i.test(text)) fails.push(where + ": falta codegraph");
	if (/codebase-memory/i.test(text)) fails.push(where + ": no debe exigir codebase-memory");
};
principal("impacto", find(/impacto obligatorio/i));
principal("plan", find(/^- Modo plan/));
principal("IMPACT_PROTOCOL", IMPACT_PROTOCOL);
const hcd = find(/^- HCD:/);
if (!/una verificación por llamada/i.test(hcd) || !/panel de tareas/i.test(hcd)) fails.push("HCD: falta una verificación por llamada Bash para el panel de tareas");
if (/codegraph y codebase-memory/.test(hcd)) fails.push("HCD: sigue exigiendo los dos grafos");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD073Protocolo: %s", out)
	}
}

// TestHCD073GeminiConfig: mem instala las skills de agy en ~/.gemini/config/skills;
// una skill que solo está ahí también llega a Pi.
func TestHCD073GeminiConfig(t *testing.T) {
	home := t.TempDir()
	src := filepath.Join(home, ".gemini", "config", "skills", "solo-gemini")
	hcd073Write(t, filepath.Join(src, "SKILL.md"), "---\nname: solo-gemini\ndescription: x\n---\n")
	if _, err := SyncSkillRepo(hcd073Assets(), home); err != nil {
		t.Fatal(err)
	}
	if target, err := os.Readlink(filepath.Join(SkillRepoDir(home), "solo-gemini")); err != nil || target != src {
		t.Fatalf("solo-gemini no se enlazó desde ~/.gemini/config/skills: %q %v", target, err)
	}
}

// TestHCD073DoctorDesfase: doctor avisa cuando una copia intacta de Pi difiere
// de la que instaló mem; una copia personalizada no se reporta.
func TestHCD073DoctorDesfase(t *testing.T) {
	home := t.TempDir()
	memPath := filepath.Join(home, ".claude", "skills", "hcd-verify", "SKILL.md")
	hcd073Write(t, memPath, "---\nname: hcd-verify\ndescription: v1\n---\nv1\n")
	hcd073Write(t, filepath.Join(home, ".claude", "skills", "hcd-close", "SKILL.md"), "---\nname: hcd-close\ndescription: v1\n---\nv1\n")
	if _, err := SyncSkillRepo(hcd073Assets(), home); err != nil {
		t.Fatal(err)
	}
	if item := skillDriftItem(home); item.Status != "OK" {
		t.Fatalf("recién sincronizado se espera OK: %+v", item)
	}

	hcd073Write(t, memPath, "---\nname: hcd-verify\ndescription: v2\n---\nv2\n")
	hcd073Write(t, filepath.Join(SkillRepoDir(home), "hcd-close", "SKILL.md"), "personalizada")
	item := skillDriftItem(home)
	if item.Status != "WARN" || !strings.Contains(item.Details, "hcd-verify") || strings.Contains(item.Details, "hcd-close") || !strings.Contains(item.Details, "skills-sync") {
		t.Fatalf("desfase mal reportado: %+v", item)
	}
}

// TestHCD073SkillsDesdeMem: la copia que mem instala en los agentes del
// anfitrión gana a la embebida; sin copia de mem se usa la embebida y una
// copia personalizada en el repositorio nunca se pisa (HCD-073, Q1=a).
func TestHCD073SkillsDesdeMem(t *testing.T) {
	home := t.TempDir()
	mem := "---\nname: hcd-verify\ndescription: de mem\n---\nversión de mem\n"
	hcd073Write(t, filepath.Join(home, ".claude", "skills", "hcd-verify", "SKILL.md"), mem)

	if _, err := SyncSkillRepo(hcd073Assets(), home); err != nil {
		t.Fatal(err)
	}
	repo := SkillRepoDir(home)
	if got := hcd073Read(t, filepath.Join(repo, "hcd-verify", "SKILL.md")); got != mem {
		t.Fatalf("hcd-verify no viene de mem:\n%s", got)
	}
	if !isPristineEmbeddedCopy(filepath.Join(repo, "hcd-verify")) {
		t.Fatal("la copia tomada de mem debe llevar marca para actualizarse después")
	}
	if got := hcd073Read(t, filepath.Join(repo, "hcd-close", "SKILL.md")); !strings.Contains(got, "embebida") {
		t.Fatalf("sin copia de mem se espera la embebida:\n%s", got)
	}

	// mem publica una versión nueva: la siguiente sincronización la recoge.
	mem2 := strings.Replace(mem, "versión de mem", "versión nueva de mem", 1)
	hcd073Write(t, filepath.Join(home, ".claude", "skills", "hcd-verify", "SKILL.md"), mem2)
	if _, err := SyncSkillRepo(hcd073Assets(), home); err != nil {
		t.Fatal(err)
	}
	if got := hcd073Read(t, filepath.Join(repo, "hcd-verify", "SKILL.md")); got != mem2 {
		t.Fatalf("no se actualizó con la versión nueva de mem:\n%s", got)
	}

	// Una copia editada por la persona se conserva.
	hcd073Write(t, filepath.Join(repo, "hcd-verify", "SKILL.md"), "personalizada")
	if _, err := SyncSkillRepo(hcd073Assets(), home); err != nil {
		t.Fatal(err)
	}
	if got := hcd073Read(t, filepath.Join(repo, "hcd-verify", "SKILL.md")); got != "personalizada" {
		t.Fatalf("se pisó la copia personalizada:\n%s", got)
	}
}
