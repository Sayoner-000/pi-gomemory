package installer

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// tuiResources son las extensiones embebidas y la definición neutral de Splash.
var tuiResources = []struct{ source, destination string }{
	{"extensions/gomemory-status.ts", "gomemory-status.ts"},
	{"extensions/gomemory-splash.ts", "gomemory-splash.ts"},
	{"extensions/gomemory-banner.ts", "gomemory-banner.ts"},
	{"extensions/gomemory-plan.ts", "gomemory-plan.ts"},
	{"extensions/gomemory-plan-patterns.json", "gomemory-plan-patterns.json"},
	{"extensions/gomemory-tasks.ts", "gomemory-tasks.ts"},
	{"extensions/gomemory-sidebar.ts", "gomemory-sidebar.ts"},
	{"extensions/gomemory-context.ts", "gomemory-context.ts"}, // contexto vivo de gomemory (HCD-045)
	{"extensions/gomemory-suggest.ts", "gomemory-suggest.ts"}, // sugerencia del siguiente comando (HCD-057)
	{"config/splash.md", "gomemory-splash-prompt.md"},
}

// retiredResources son extensiones que instalaron versiones anteriores y ya no
// se distribuyen (HCD-085): /tdd y el workaround de pi-antigravity. Install y
// uninstall las retiran solo si siguen intactas según el manifiesto.
var retiredResources = []string{
	"gomemory-tdd.ts",
	"antigravity-prompt-guard.js",
}

// InstallTUIExtension instala las extensiones embebidas y la definición neutral de Splash.
func InstallTUIExtension(assets fs.FS, piAgentDir string) error {
	for _, resource := range tuiResources {
		data, err := fs.ReadFile(assets, resource.source)
		if err != nil {
			return fmt.Errorf("leer extensión embebida: %w", err)
		}

		extDir := filepath.Join(piAgentDir, "extensions")
		if err := os.MkdirAll(extDir, 0755); err != nil {
			return fmt.Errorf("crear dir extensions en pi: %w", err)
		}

		dstFile := filepath.Join(extDir, resource.destination)
		if err := writeFileAtomic(dstFile, data, 0644); err != nil {
			return fmt.Errorf("escribir extensión %s: %w", dstFile, err)
		}
	}
	return nil
}

// RetireResources borra las extensiones retiradas cuyo contenido coincide con
// la huella del manifiesto anterior. Una copia editada, o sin huella que pruebe
// que la instaló pi-gomemory, se conserva con un aviso. Debe ejecutarse antes
// de WriteManifest, que ya no registra estos archivos.
func RetireResources(opts *Options, out io.Writer) error {
	m, err := ReadManifest(opts.HomeDir)
	if err != nil {
		fmt.Fprintf(out, "%s    ⚠ manifiesto ilegible; no se retiran extensiones antiguas: %v%s\n", ColorWarning, err, ColorReset)
		return nil
	}
	for _, name := range retiredResources {
		path := filepath.Join(opts.PiAgentDir, "extensions", name)
		data, err := os.ReadFile(path)
		if isAbsent(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspeccionar %s: %w", path, err)
		}
		want := ownedHash(m, path, nil)
		if want == "" || contentHash(data) != want {
			fmt.Fprintf(out, "%s    ⚠ %s ya no se distribuye y se editó o no consta en el manifiesto: se conserva; bórralo y haz /reload si no lo usas%s\n", ColorWarning, path, ColorReset)
			continue
		}
		if err := ownedFileUnchanged(path, want); err != nil {
			continue
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("retirar %s: %w", path, err)
		}
		fmt.Fprintf(out, "%s    ✓ retirada %s (ya no se distribuye)%s\n", ColorMuted, path, ColorReset)
	}
	return nil
}
