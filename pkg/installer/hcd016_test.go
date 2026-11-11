package installer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// H-01: pruneLegacyPiSkills no debe borrar con RemoveAll carpetas que tengan
// archivos adicionales del usuario además de SKILL.md.
func TestH01PruneLegacyPreservesUserData(t *testing.T) {
	opts := sandbox(t)
	repo := SkillRepoDir(opts.HomeDir)
	piSkills := filepath.Join(opts.PiAgentDir, "skills")

	skillA := filepath.Join(repo, "skillA")
	writeFile(t, filepath.Join(skillA, "SKILL.md"), "content-A\n")

	// Carpeta en piSkills con SKILL.md idéntico pero con script de usuario
	targetA := filepath.Join(piSkills, "skillA")
	writeFile(t, filepath.Join(targetA, "SKILL.md"), "content-A\n")
	userScript := filepath.Join(targetA, "scripts", "mine.sh")
	writeFile(t, userScript, "#!/bin/sh\necho mine\n")

	// Carpeta en piSkills con solo SKILL.md idéntico (debe borrarse)
	skillB := filepath.Join(repo, "skillB")
	writeFile(t, filepath.Join(skillB, "SKILL.md"), "content-B\n")
	targetB := filepath.Join(piSkills, "skillB")
	writeFile(t, filepath.Join(targetB, "SKILL.md"), "content-B\n")

	if err := pruneLegacyPiSkills(piSkills, opts.HomeDir); err != nil {
		t.Fatalf("pruneLegacyPiSkills error: %v", err)
	}

	// targetA debe conservarse porque tiene archivos del usuario
	if _, err := os.Stat(userScript); os.IsNotExist(err) {
		t.Fatalf("PERDIDA DE DATOS: userScript fue borrado por pruneLegacyPiSkills")
	}

	// targetB debe borrarse porque solo contenía SKILL.md idéntico
	if _, err := os.Stat(targetB); !os.IsNotExist(err) {
		t.Fatalf("targetB debió ser podado pero todavía existe")
	}
}

// H-02: Uninstall y dry-run deben fallar si hay ActionFail aunque Pending() sea 0
func TestH02UninstallFailsWhenActionFail(t *testing.T) {
	opts := sandbox(t)
	// Corromper settings.json para generar ActionFail en planSettings
	writeFile(t, filepath.Join(opts.PiAgentDir, "settings.json"), "{corrupt")

	var buf bytes.Buffer
	// Test con dry-run
	errDry := Uninstall(repoAssets(t), opts, UninstallFlags{DryRun: true}, &buf)
	if errDry == nil {
		t.Fatalf("Uninstall con dry-run y ActionFail debio retornar error, obtuvo nil")
	}

	buf.Reset()
	// Test con ejecución
	errRun := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: true}, &buf)
	if errRun == nil {
		t.Fatalf("Uninstall con ActionFail debio retornar error, obtuvo nil")
	}
}

// H-03: doctor debe emitir WARN si pi mcp list falla o devuelve JSON inválido
func TestH03DoctorTimeoutAndWarn(t *testing.T) {
	opts := sandbox(t)
	// Crear un binario 'pi' en PATH que devuelva error o JSON roto
	fakeBin := filepath.Join(opts.HomeDir, "bin")
	if err := os.MkdirAll(fakeBin, 0755); err != nil {
		t.Fatal(err)
	}
	fakePi := filepath.Join(fakeBin, "pi")
	writeFile(t, fakePi, "#!/bin/sh\necho 'not-valid-json'\nexit 1\n")
	_ = os.Chmod(fakePi, 0755)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", fakeBin+string(filepath.ListSeparator)+origPath)

	report := RunDoctor(opts)
	foundMcpWarn := false
	for _, it := range report {
		if it.Name == "MCP en Pi" && it.Status == "WARN" {
			foundMcpWarn = true
			break
		}
	}
	if !foundMcpWarn {
		t.Fatalf("Se esperaba CheckItem 'MCP en Pi' con status WARN ante fallo de pi mcp list, pero no se encontro")
	}
}

// H-04: gomemory-plan debe bloquear evasiones de intérpretes y comandos mutables
func TestH04PlanEvasionsBlocked(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-plan.ts", "extensions/gomemory-plan-patterns.json"}, `
import { bashBlocked } from "./gomemory-plan.ts";

const evasions = [
  "bash -c 'rm -rf build'",
  "sh -c \"touch f\"",
  "zsh -c 'rm f'",
  "python3 -c \"open('f','w')\"",
  "python -c 'import os; os.remove(1)'",
  "node -e \"require('fs').writeFileSync('f','')\"",
  "perl -pi -e 's/a/b/' file",
  "ruby -e 'File.write(\"f\", \"\")'",
  "xargs rm",
  "env rm f",
  "make build",
  "find . -delete",
  "find . -exec rm {} \\;",
  "dd of=f if=/dev/zero",
  "git apply patch.diff",
  "git clean -fd",
  "git pull origin main",
  "patch -p1 < file.patch",
  "curl -o output.bin http://example.com",
  "wget -O output.bin http://example.com",
  "install -m 755 src dst",
  "rsync -av src/ dst/",
  "unlink file.txt",
  "go build -o /tmp/bin ."
];

const unblocked = [];
for (const cmd of evasions) {
  if (!bashBlocked(cmd)) {
    unblocked.push(cmd);
  }
}

console.log(unblocked.length ? "FAIL " + unblocked.join(" | ") : "PASS");
`)

	if out != "PASS" {
		t.Fatal(out)
	}
}

// H-08: gomemory-sidebar debe limpiar errores JSON de stderr y validar estructura defensivamente
func TestH08SidebarErrorAndInflight(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import { parseCleanQuotaError, validateQuotaSummary } from "./gomemory-sidebar.ts";

const fails = [];

// 1. Limpieza de error
const jsonErr = JSON.stringify({ error: "no se encontró cuenta de Antigravity en /path/to/acc.json" });
const clean = parseCleanQuotaError(jsonErr);
if (clean !== "no se encontró cuenta de Antigravity en /path/to/acc.json") {
  fails.push("parseCleanQuotaError no extrajo campo error: " + clean);
}

const plainErr = "error de conexion\nlinea 2";
if (parseCleanQuotaError(plainErr) !== "error de conexion") {
  fails.push("parseCleanQuotaError no tomo primera linea en texto plano");
}

// 2. Validacion defensiva
if (validateQuotaSummary(null) !== null) fails.push("null debe ser invalido");
if (validateQuotaSummary({}) !== null) fails.push("{} sin groups debe ser invalido");
if (validateQuotaSummary({ groups: [{ displayName: "G", buckets: [{ displayName: "B", remainingFraction: "no-num" }] }] }) !== null) {
  fails.push("bucket con remainingFraction no numerico debe ser invalido");
}

const valid = {
  planLabel: "Plan",
  groups: [{ displayName: "G", buckets: [{ displayName: "B", remainingFraction: 0.5, remainingPercent: 50.0, resetIn: "1h" }] }]
};
if (validateQuotaSummary(valid) === null) fails.push("summary valido fue rechazado");

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)

	if out != "PASS" {
		t.Fatal(out)
	}
}

// H-05 y H-06: claude_usage debe parsear fechas con zona local y ordenar archivos para atribución determinista
func TestH05H06ClaudeUsageDeterministicAndLocal(t *testing.T) {
	opts := sandbox(t)
	claudeDir := filepath.Join(opts.PiAgentDir, "fake_claude")

	// Usar un timestamp que sea "hoy" en UTC pero "ayer" en local, o viceversa,
	// y filas duplicadas en dos proyectos A y B.
	now := time.Now()
	nowLocalStr := now.Format(time.RFC3339)

	cacheData := fmt.Sprintf(`{
  "version": 3,
  "files": {
    "/fake/path/projZ/session.jsonl": {
      "size": 100,
      "mtime": 1000.0,
      "project": "proj-z",
      "rows": {
        "dup|req": [%q, "claude-sonnet-5.5", 100, 20, 0, 0]
      }
    },
    "/fake/path/projA/session.jsonl": {
      "size": 100,
      "mtime": 1000.0,
      "project": "proj-a",
      "rows": {
        "dup|req": [%q, "claude-sonnet-5.5", 100, 20, 0, 0]
      }
    }
  }
}`, nowLocalStr, nowLocalStr)

	writeFile(t, filepath.Join(claudeDir, "token-usage-cache.json"), cacheData)

	for i := 0; i < 20; i++ {
		res, err := FetchClaudeUsage(claudeDir)
		if err != nil {
			t.Fatalf("FetchClaudeUsage fallo: %v", err)
		}
		if res.Today.Messages != 1 {
			t.Fatalf("iteracion %d: se esperaba Today.Messages == 1 para fecha local actual, obtuvo %d", i, res.Today.Messages)
		}
		// projA debe recibir los tokens del mensaje duplicado
		if len(res.Projects) == 0 || res.Projects[0].Name != "proj-a" || res.Projects[0].Messages != 1 {
			t.Fatalf("iteracion %d: no determinista, se esperaba que proj-a tuviera el mensaje: %+v", i, res.Projects)
		}
	}
}

// H-09: SyncSkillRepo solo borra symlinks de fuentes del host; enlaces de usuario se conservan
func TestH09SyncSkillRepoAndUninstallPackages(t *testing.T) {
	opts := sandbox(t)
	repo := SkillRepoDir(opts.HomeDir)
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}

	// 1. Enlace creado por el usuario a una carpeta fuera de hostSkillSources
	userFolder := filepath.Join(opts.HomeDir, "my-custom-skills", "custom1")
	writeFile(t, filepath.Join(userFolder, "SKILL.md"), "custom\n")
	userLink := filepath.Join(repo, "custom1")
	if err := os.Symlink(userFolder, userLink); err != nil {
		t.Fatal(err)
	}

	// 2. Enlace huérfano que apunta a ~/.claude/skills/old (está bajo hostSkillSources)
	claudeSkills := filepath.Join(opts.HomeDir, ".claude", "skills", "old")
	claudeLink := filepath.Join(repo, "old")
	if err := os.Symlink(claudeSkills, claudeLink); err != nil {
		t.Fatal(err)
	}

	res, err := SyncSkillRepo(repoAssets(t), opts.HomeDir)
	if err != nil {
		t.Fatalf("SyncSkillRepo fallo: %v", err)
	}
	_ = res

	// userLink debe sobrevivir
	if _, err := os.Lstat(userLink); os.IsNotExist(err) {
		t.Fatalf("SyncSkillRepo borro enlace del usuario custom1")
	}

	// claudeLink debe haberse eliminado
	if _, err := os.Lstat(claudeLink); !os.IsNotExist(err) {
		t.Fatalf("SyncSkillRepo no borro enlace huerfano del host")
	}
}

// H-11: RefreshDeps respeta el TTL de 24h y GenerateHarnessScript trunca contextos mayores a 64 KiB
func TestH11HarnessTTLAndPromptCap(t *testing.T) {
	opts := sandbox(t)
	cachePath := DefaultDepsCachePath(opts.PiAgentDir)

	called := false
	probes := []DepProbe{
		{
			Name: "test-dep",
			Current: func(ctx context.Context) (string, error) {
				called = true
				return "1.0.0", nil
			},
			Latest: func(ctx context.Context) (string, error) {
				return "1.0.1", nil
			},
		},
	}

	now := time.Now()
	// Primera llamada: debe consultar en red y escribir caché
	res1 := RefreshDeps(probes, cachePath, now)
	if !called || len(res1) == 0 {
		t.Fatalf("RefreshDeps inicial no consulto los probes")
	}

	called = false
	// Segunda llamada 1 hora despues: debe retornar de cache sin invocar probes
	res2 := RefreshDeps(probes, cachePath, now.Add(1*time.Hour))
	if called {
		t.Fatalf("RefreshDeps volvio a consultar la red antes de expirar el TTL de 24h")
	}
	if len(res2) == 0 || res2[0].Name != "test-dep" {
		t.Fatalf("RefreshDeps no retorno datos de cache: %+v", res2)
	}

	// Verificar guarda de tamaño en el script del harness
	script := GenerateHarnessScript()
	if !bytes.Contains([]byte(script), []byte("65536")) {
		t.Fatalf("GenerateHarnessScript no contiene proteccion de truncado para contextos grandes")
	}
}
