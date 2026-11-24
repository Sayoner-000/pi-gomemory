package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pi-gomemory/pkg/installer"
)

// hcd080EditedHome deja un repositorio central sincronizado con hcd-verify
// editada a mano después de instalarla.
func hcd080EditedHome(t *testing.T) (*installer.Options, string) {
	t.Helper()
	home := t.TempDir()
	opts := &installer.Options{HomeDir: home, PiAgentDir: filepath.Join(home, ".pi", "agent")}
	var out, errOut bytes.Buffer
	if code := runSkillsSync(opts, "skills-sync", nil, &out, &errOut); code != 0 {
		t.Fatalf("sync: %d %s", code, errOut.String())
	}
	edited := filepath.Join(installer.SkillRepoDir(home), "hcd-verify", "SKILL.md")
	if err := os.WriteFile(edited, []byte("apaño local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return opts, edited
}

// HCD-080 tarea 10 (Q1): --check informa de las editadas en una línea aparte,
// con el comando para reponerlas, sale con 10 y no las toca.
func TestHCD080SkillsSyncCheckEdited(t *testing.T) {
	opts, edited := hcd080EditedHome(t)
	var out, errOut bytes.Buffer
	if code := runSkillsSync(opts, "skills-sync", []string{"--check"}, &out, &errOut); code != 10 {
		t.Fatalf("con una editada: código %d, quiero 10\n%s", code, errOut.String())
	}
	msg := errOut.String()
	if strings.Contains(msg, "desfasadas en") || !strings.Contains(msg, "editadas localmente") || !strings.Contains(msg, "skills-sync --restore hcd-verify") {
		t.Fatalf("debe nombrar solo la editada y cómo reponerla: %q", msg)
	}
	if b, _ := os.ReadFile(edited); string(b) != "apaño local\n" {
		t.Fatal("--check no debe tocar la copia editada")
	}
}

// HCD-080 tarea 11 (Q1): --restore repone una skill de gomemory editada; rechaza
// nombres que no son de gomemory, la falta de nombre y carpetas con más
// archivos que SKILL.md y la marca.
func TestHCD080SkillsSyncRestore(t *testing.T) {
	opts, edited := hcd080EditedHome(t)
	var out, errOut bytes.Buffer
	for _, args := range [][]string{{"--restore"}, {"--restore", "no-existe"}, {"--restore", "../hcd-verify"}} {
		if code := runSkillsSync(opts, "skills-sync", args, &out, &errOut); code != 2 {
			t.Fatalf("%v: código %d, quiero 2", args, code)
		}
	}
	extra := filepath.Join(filepath.Dir(edited), "notas.txt")
	if err := os.WriteFile(extra, []byte("mío"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runSkillsSync(opts, "skills-sync", []string{"--restore", "hcd-verify"}, &out, &errOut); code != 1 {
		t.Fatalf("con archivos propios debe negarse: código %d", code)
	}
	if b, _ := os.ReadFile(edited); string(b) != "apaño local\n" {
		t.Fatal("al negarse no debe tocar nada")
	}
	if err := os.Remove(extra); err != nil {
		t.Fatal(err)
	}
	errOut.Reset()
	if code := runSkillsSync(opts, "skills-sync", []string{"--restore", "hcd-verify"}, &out, &errOut); code != 0 {
		t.Fatalf("restore: código %d %s", code, errOut.String())
	}
	if b, _ := os.ReadFile(edited); string(b) == "apaño local\n" {
		t.Fatal("--restore debe reponer la versión vigente")
	}
	if code := runSkillsSync(opts, "skills-sync", []string{"--check"}, &out, &errOut); code != 0 {
		t.Fatalf("tras restore: código %d %s", code, errOut.String())
	}
}

// HCD-080 tarea 5: las skills embebidas en el repositorio coinciden con las
// que instaló el `mem` de esta máquina. Se omite si no hay copias de mem (CI)
// o con PI_GOMEMORY_SKIP_SKILL_DRIFT=1.
func TestHCD080EmbeddedSkillsMatchMem(t *testing.T) {
	if os.Getenv("PI_GOMEMORY_SKIP_SKILL_DRIFT") == "1" {
		t.Skip("PI_GOMEMORY_SKIP_SKILL_DRIFT=1")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	drift, n := installer.EmbeddedSkillDrift(EmbeddedAssets, home)
	if n == 0 {
		t.Skip("mem no instaló skills en ningún agente de esta máquina")
	}
	if len(drift) > 0 {
		t.Fatalf("skills/ embebidas desfasadas respecto a las de mem: %s. Ejecuta `make skills-embed` (o `mem update` si tu mem es el antiguo)", strings.Join(drift, ", "))
	}
}

// HCD-080 tarea 2: `skills-sync --check` no escribe; sale con 10 y nombra las
// skills desfasadas, o con 0 si el repositorio central está al día. Sin
// --check sincroniza; una opción desconocida termina con 2.
func TestHCD080SkillsSyncCheck(t *testing.T) {
	home := t.TempDir()
	opts := &installer.Options{HomeDir: home, PiAgentDir: filepath.Join(home, ".pi", "agent")}
	var out, errOut bytes.Buffer

	if code := runSkillsSync(opts, "skills-sync", []string{"--check"}, &out, &errOut); code != 10 {
		t.Fatalf("repositorio vacío: código %d, quiero 10\n%s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "hcd-verify") || !strings.Contains(errOut.String(), "skills-sync") {
		t.Fatalf("--check debe nombrar las desfasadas y cómo corregirlo: %q", errOut.String())
	}
	if _, err := os.Stat(installer.SkillRepoDir(home)); !os.IsNotExist(err) {
		t.Fatalf("--check no debe crear el repositorio: %v", err)
	}

	errOut.Reset()
	if code := runSkillsSync(opts, "skills-sync", nil, &out, &errOut); code != 0 {
		t.Fatalf("sync: código %d\n%s", code, errOut.String())
	}
	if code := runSkillsSync(opts, "skills-sync", []string{"--check"}, &out, &errOut); code != 0 {
		t.Fatalf("tras sincronizar: código %d\n%s", code, errOut.String())
	}
	if code := runSkillsSync(opts, "skills-sync", []string{"--bogus"}, &out, &errOut); code != 2 {
		t.Fatalf("opción desconocida: código %d", code)
	}
}
