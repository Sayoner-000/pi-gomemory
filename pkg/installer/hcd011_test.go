package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func installSandbox(t *testing.T) *Options {
	t.Helper()
	opts := sandbox(t)
	if err := InstallAll(repoAssets(t), opts); err != nil {
		t.Fatal(err)
	}
	return opts
}

// Tarea 1 · C-16, C-17: install deja un manifiesto con SHA256 y BIN_DIR, y retira la marca.
func TestHCD011Manifest(t *testing.T) {
	opts := sandbox(t)
	writeFile(t, UninstalledMarkerPath(opts.HomeDir), "")
	opts.LocalBinDir = filepath.Join(opts.HomeDir, "custom-bin")
	if err := InstallAll(repoAssets(t), opts); err != nil {
		t.Fatal(err)
	}
	m, err := ReadManifest(opts.HomeDir)
	if err != nil || m == nil {
		t.Fatalf("manifiesto ausente: %v", err)
	}
	if m.BinDir != opts.LocalBinDir {
		t.Fatalf("BinDir = %q, quiero %q", m.BinDir, opts.LocalBinDir)
	}
	for _, p := range []string{
		filepath.Join(opts.PiAgentDir, "themes", "gomemory.json"),
		filepath.Join(opts.PiAgentDir, "extensions", "gomemory-status.ts"),
		filepath.Join(opts.LocalBinDir, "pi-gomemory"),
	} {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if m.Files[p] != contentHash(data) {
			t.Fatalf("hash de %s no registrado o distinto", p)
		}
	}
	if _, err := os.Stat(UninstalledMarkerPath(opts.HomeDir)); err == nil {
		t.Fatal("install no retiró la marca de desinstalado")
	}
	raw, _ := os.ReadFile(ManifestPath(opts.HomeDir))
	if !json.Valid(raw) {
		t.Fatal("manifiesto no es JSON válido")
	}
}

// Tarea 2 · C-18: con la marca, skills-sync no recrea repositorio ni registro.
func TestHCD011SkillsSyncRespectsMarker(t *testing.T) {
	opts := sandbox(t)
	writeFile(t, UninstalledMarkerPath(opts.HomeDir), "")
	if _, err := InstallSkills(repoAssets(t), opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(SkillRepoDir(opts.HomeDir), "adversarial-consensus-review")); err == nil {
		t.Fatal("skills-sync recreó skills con la marca de desinstalado")
	}
	if _, err := os.Stat(filepath.Join(opts.PiAgentDir, "settings.json")); err == nil {
		t.Fatal("skills-sync escribió settings.json con la marca de desinstalado")
	}
}
