package installer

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// HCD-013: doctor avisa cuando la cabecera de pi-cc-extensions tapa el banner.
func TestDoctorWarnsCCStartupHeader(t *testing.T) {
	opts := sandbox(t)
	writeFile(t, filepath.Join(opts.PiAgentDir, "pi-cc-extensions.json"), `{"showStartupHeader":true}`)

	var found *CheckItem
	items := RunDoctor(opts)
	for i := range items {
		if items[i].Name == "Banner gomemory" {
			found = &items[i]
		}
	}
	if found == nil || found.Status != "WARN" || !strings.Contains(found.Details, "showStartupHeader") {
		t.Fatalf("doctor no avisa de showStartupHeader activo: %+v", found)
	}

	writeFile(t, filepath.Join(opts.PiAgentDir, "pi-cc-extensions.json"), `{"showStartupHeader":false}`)
	for _, it := range RunDoctor(opts) {
		if it.Name == "Banner gomemory" && it.Status != "OK" {
			t.Fatalf("con showStartupHeader=false el banner debería estar OK: %+v", it)
		}
	}
}

// HCD-013: reinstalar sobre un tema y un banner modificados (o sobre otro tema
// activo) los deja iguales a los embebidos, sin depender del SO.
func TestReinstallRestoresThemeAndBanner(t *testing.T) {
	assets := repoAssets(t)
	opts := sandbox(t)
	themePath := filepath.Join(opts.PiAgentDir, "themes", "gomemory.json")
	bannerPath := filepath.Join(opts.PiAgentDir, "extensions", "gomemory-banner.ts")
	writeFile(t, themePath, `{"viejo":true}`)
	writeFile(t, bannerPath, "// banner antiguo")
	writeFile(t, filepath.Join(opts.PiAgentDir, "settings.json"), `{"theme":"dark"}`)

	if err := InstallAll(assets, opts); err != nil {
		t.Fatal(err)
	}

	for src, dst := range map[string]string{"themes/gomemory.json": themePath, "extensions/gomemory-banner.ts": bannerPath} {
		want, _ := fs.ReadFile(assets, src)
		if readFile(t, dst) != string(want) {
			t.Errorf("%s no se restauró al embebido", dst)
		}
	}
	if s := readFile(t, filepath.Join(opts.PiAgentDir, "settings.json")); !strings.Contains(s, `"theme": "gomemory"`) {
		t.Errorf("el tema gomemory no quedó activo: %s", s)
	}
}
