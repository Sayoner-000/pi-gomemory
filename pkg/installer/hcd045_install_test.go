package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hcd045Block es el bloque que escribían las instalaciones anteriores a HCD-045.
func hcd045Block() string {
	return instructionsStart + "\n" + legacyPiInstructions + instructionsEnd + "\n"
}

// C-05: install deja AGENTS.md exactamente igual a stripInstructions(original),
// con el bloque en medio, al final o en el formato antiguo sin marcadores,
// conservando permisos y enlace; y nunca vuelve a escribir el protocolo.
func TestHCD045InstallStrips(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"bloque al final":    "# Mis reglas\n\nNo tocar.\n\n" + hcd045Block(),
		"bloque en medio":    "# Mis reglas\n\n" + hcd045Block() + "\nPie propio.\n",
		"formato sin marcas": "# Mis reglas\n\n" + legacyPiInstructions + "\nPie propio.\n",
	}
	for name, original := range cases {
		want, changed := stripInstructions(original)
		if !changed {
			t.Fatalf("%s: el caso no contiene el bloque", name)
		}
		for _, linked := range []bool{false, true} {
			t.Run(name+map[bool]string{false: "", true: " (enlace)"}[linked], func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "AGENTS.md")
				target := path
				if linked {
					target = linkConfig(t, path, original, 0600)
				} else {
					writeFile(t, path, original)
					if err := os.Chmod(path, 0600); err != nil {
						t.Fatal(err)
					}
				}
				retired, err := RetireLegacyInstructions(dir)
				if err != nil || !retired {
					t.Fatalf("RetireLegacyInstructions = %v, %v", retired, err)
				}
				if got := readFile(t, target); got != want {
					t.Fatalf("contenido:\n got %q\nwant %q", got, want)
				}
				assertMode(t, target, 0600)
				if linked {
					assertLink(t, path, target)
				}
				// Idempotente y sin volver a escribir el protocolo.
				retired, err = RetireLegacyInstructions(dir)
				if err != nil || retired {
					t.Fatalf("segunda pasada = %v, %v", retired, err)
				}
				if got := readFile(t, target); got != want || strings.Contains(got, "Protocolo para Pi") {
					t.Fatalf("la segunda pasada cambió el archivo: %q", got)
				}
			})
		}
	}

	t.Run("sin bloque no se toca", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "AGENTS.md")
		writeFile(t, path, "# Mis reglas\n\n\n")
		if retired, err := RetireLegacyInstructions(dir); err != nil || retired {
			t.Fatalf("= %v, %v", retired, err)
		}
		if got := readFile(t, path); got != "# Mis reglas\n\n\n" {
			t.Fatalf("cambió un AGENTS.md sin bloque: %q", got)
		}
	})

	t.Run("sin AGENTS.md no se crea", func(t *testing.T) {
		dir := t.TempDir()
		if retired, err := RetireLegacyInstructions(dir); err != nil || retired {
			t.Fatalf("= %v, %v", retired, err)
		}
		if _, err := os.Lstat(filepath.Join(dir, "AGENTS.md")); !os.IsNotExist(err) {
			t.Fatalf("se creó AGENTS.md: %v", err)
		}
	})
}

// C-05: el formato sin marcadores se reconoce en cada versión publicada del
// protocolo (extraídas de git a legacy/), no solo en la última. Caso real de
// la verificación con Pi: la versión de 2026-11-09 seguida de bloques ajenos.
func TestHCD045InstallStripsHistoricVersions(t *testing.T) {
	t.Parallel()

	foreign := "<!-- codebase-memory-mcp:start -->\n# Codebase Memory\n<!-- codebase-memory-mcp:end -->\n"
	for _, v := range []string{"protocol-1.txt", "protocol-2.txt"} {
		t.Run(v, func(t *testing.T) {
			old := readFile(t, filepath.Join("legacy", v))
			dir := t.TempDir()
			path := filepath.Join(dir, "AGENTS.md")
			writeFile(t, path, old+"\n"+foreign)
			if retired, err := RetireLegacyInstructions(dir); err != nil || !retired {
				t.Fatalf("no reconoce %s: %v, %v", v, retired, err)
			}
			if got := readFile(t, path); got != foreign {
				t.Fatalf("contenido:\n got %q\nwant %q", got, foreign)
			}
			if item := legacyAgentsItem(dir); item != nil {
				t.Fatalf("doctor sigue avisando tras retirar %s", v)
			}
		})
	}
}

// C-06 (Q4): si AGENTS.md solo tenía el bloque, install hace lo mismo que
// uninstall: borra el archivo normal y vacía el destino de un enlace.
func TestHCD045InstallOnlyBlock(t *testing.T) {
	t.Parallel()

	t.Run("archivo normal se borra", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "AGENTS.md")
		writeFile(t, path, hcd045Block())
		if _, err := RetireLegacyInstructions(dir); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("AGENTS.md con solo el bloque sigue existiendo: %v", err)
		}
	})
	t.Run("enlace se conserva vacío", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "AGENTS.md")
		target := linkConfig(t, path, hcd045Block(), 0640)
		if _, err := RetireLegacyInstructions(dir); err != nil {
			t.Fatal(err)
		}
		assertLink(t, path, target)
		if got := readFile(t, target); got != "" {
			t.Fatalf("el destino no quedó vacío: %q", got)
		}
		assertMode(t, target, 0640)
	})
}

// C-07: si no se puede escribir AGENTS.md, el paso de install informa y no
// aborta la instalación.
func TestHCD045InstallReadOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	writeFile(t, path, "mías\n\n"+hcd045Block())
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })
	if _, err := RetireLegacyInstructions(dir); err == nil {
		t.Skip("el sistema permite escribir en un directorio 0555 (¿root?)")
	}
	var out bytes.Buffer
	if err := retireLegacyStep(dir, &out); err != nil {
		t.Fatalf("un fallo de escritura no debe abortar InstallAll: %v", err)
	}
	if !strings.Contains(out.String(), "AGENTS.md") || !strings.Contains(out.String(), "pi-gomemory-setup install") {
		t.Fatalf("el aviso no informa del fallo ni de cómo reintentar: %q", out.String())
	}
}

// C-11: gomemory-context.ts se instala con las demás extensiones y figura en el
// manifiesto, que es lo que uninstall retira.
func TestHCD045ContextExtensionInstalled(t *testing.T) {
	opts := sandbox(t)
	if err := InstallTUIExtension(repoAssets(t), opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(opts.PiAgentDir, "extensions", "gomemory-context.ts")
	if readFile(t, dst) != readAsset(t, "extensions/gomemory-context.ts") {
		t.Fatal("gomemory-context.ts no se instala o difiere del embebido")
	}
	found := false
	for _, f := range installedFiles(opts) {
		found = found || f == dst
	}
	if !found {
		t.Fatal("gomemory-context.ts no figura en el manifiesto: uninstall no lo retiraría")
	}
}

// C-07: doctor avisa del bloque heredado con la orden que lo retira, y calla
// cuando no está.
func TestHCD045DoctorLegacyBlock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if item := legacyAgentsItem(dir); item != nil {
		t.Fatalf("sin AGENTS.md no debe avisar: %+v", item)
	}
	writeFile(t, path, "mías\n")
	if item := legacyAgentsItem(dir); item != nil {
		t.Fatalf("sin bloque no debe avisar: %+v", item)
	}
	writeFile(t, path, "mías\n\n"+hcd045Block())
	item := legacyAgentsItem(dir)
	if item == nil || item.Status != "WARN" ||
		!strings.Contains(item.Name+item.Details, "AGENTS.md conserva el protocolo heredado de pi-gomemory") ||
		!strings.Contains(item.Details, "pi-gomemory-setup install") {
		t.Fatalf("aviso incorrecto: %+v", item)
	}
	if !strings.Contains(readAsset(t, "pkg/installer/doctor.go"), "legacyAgentsItem(") {
		t.Fatal("RunDoctor no incluye la comprobación del bloque heredado")
	}
}
