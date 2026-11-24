package installer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestHCD075LauncherCtxBytes(t *testing.T) {
	script := GenerateHarnessScript()
	if !strings.Contains(script, "LC_ALL=C") && !strings.Contains(script, "wc -c") {
		t.Fatalf("GenerateHarnessScript no mide el tamaño de CTX en bytes")
	}

	// Extrae y ejecuta el bloque de truncado de CTX con un texto de ~120 KiB en UTF-8
	// pero menor a 65536 caracteres (40 000 glifos de 3 bytes = 120 000 bytes).
	tmp := t.TempDir()
	outFile := filepath.Join(tmp, "ctx.out")
	testScript := `
set -euo pipefail
CTX="$(python3 -c 'print("●" * 40000, end="")')"
` + extractCtxTruncateBlock(t, script) + `
printf '%s' "$CTX" > ` + outFile + `
`
	cmd := exec.Command("bash", "-c", testScript)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fallo al ejecutar truncado de CTX: %v\n%s", err, out)
	}
	b, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) > 65536 {
		t.Fatalf("CTX truncado supera 65536 bytes: %d", len(b))
	}
	if len(b) == 0 {
		t.Fatal("CTX truncado quedó vacío")
	}
	if !utf8.Valid(b) {
		t.Fatalf("CTX truncado partió un carácter UTF-8 al final")
	}
}

func extractCtxTruncateBlock(t *testing.T, script string) string {
	t.Helper()
	start := strings.Index(script, "# Evitar desbordar el límite de tamaño de argumentos")
	end := strings.Index(script, `GOMEMORY_ARGS+=(--append-system-prompt "${CTX}")`)
	if start < 0 || end < 0 || end <= start {
		t.Fatalf("no se encontró el bloque de truncado de CTX en GenerateHarnessScript")
	}
	return script[start:end]
}

func TestHCD075SkillDriftReadError(t *testing.T) {
	home := t.TempDir()
	// No creamos ~/.pi-gomemory/skills -> os.ReadDir devuelve ErrNotExist
	item := skillDriftItem(home)
	if item.Status != "WARN" {
		t.Fatalf("con repositorio de skills ausente se esperaba WARN, obtuvo %+v", item)
	}
}

func TestHCD075CorruptDepsCache(t *testing.T) {
	p := filepath.Join(t.TempDir(), "pi-gomemory-deps.json")
	if err := os.WriteFile(p, []byte("{corrupt json"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := DepsCacheError(p); err == nil {
		t.Fatalf("se esperaba error al leer caché de dependencias corrupta")
	}
}

func TestHCD075PlanMissingPatterns(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node no disponible")
	}
	dir := t.TempDir()
	src, err := os.ReadFile(filepath.Join("..", "..", "extensions", "gomemory-plan.ts"))
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "gomemory-plan.ts")
	if err := os.WriteFile(dst, src, 0644); err != nil {
		t.Fatal(err)
	}
	// Sin gomemory-plan-patterns.json junto a gomemory-plan.ts, importar el módulo
	// no debe lanzar excepción no capturada y debe seguir bloqueando edit/write.
	cmd := exec.Command("node", "--input-type=module", "-e", `
import { toolBlocked } from "./gomemory-plan.ts";
if (!toolBlocked("edit", {})) throw new Error("debería bloquear edit con respaldo");
if (!toolBlocked("bash", { command: "rm -rf foo" })) throw new Error("debería bloquear rm con respaldo");
`)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gomemory-plan.ts falló sin gomemory-plan-patterns.json: %v\n%s", err, out)
	}
}
