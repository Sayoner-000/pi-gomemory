package installer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// hcd039PinnedProbes devuelve solo los probes reales de los paquetes fijados:
// los demás ejecutan comandos o consultan GitHub.
func hcd039PinnedProbes(piAgentDir string) []DepProbe {
	var out []DepProbe
	for _, p := range DefaultDepProbes("v1.0.0", piAgentDir) {
		if p.Name == "pi-antigravity" || p.Name == "pi-cc-extensions" {
			out = append(out, p)
		}
	}
	return out
}

func hcd039StateOf(t *testing.T, states []DepState, name string) DepState {
	t.Helper()
	for _, s := range states {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("falta %s en %+v", name, states)
	return DepState{}
}

// Tarea 12 (C-17, C-23), guarda sin rojo: con npm por encima del pin,
// deps-status no lo cuenta como actualización (el pie solo mira su código 10)
// y deps-update nunca instala la versión de npm, solo el pin.
func TestHCD039DepsUnchanged(t *testing.T) {
	bin := t.TempDir()
	log := filepath.Join(bin, "pi.log")
	writeExec(t, filepath.Join(bin, "pi"), `echo "$*" >> "`+log+`"`)
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	cache := filepath.Join(t.TempDir(), "deps.json")
	writeFile(t, cache, `{"checked_at":"2026-11-17T09:30:00Z","deps":[`+
		`{"name":"pi-antigravity","current":"`+AntigravityVersion+`","latest":"`+AntigravityVersion+`","published":"99.0.0"},`+
		`{"name":"pi-cc-extensions","current":"0.0.1","latest":"`+CCExtensionsVersion+`","published":"98.0.0"}]}`)

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "npm", "node_modules", "pi-antigravity", "package.json"), `{"version":"`+AntigravityVersion+`"}`)
	writeFile(t, filepath.Join(dir, "npm", "node_modules", "pi-cc-extensions", "package.json"), `{"version":"0.0.1"}`)
	status := DepsStatusFor(cache, hcd039PinnedProbes(dir))
	if len(status) != 1 || status[0].Name != "pi-cc-extensions" || status[0].Latest != CCExtensionsVersion {
		t.Fatalf("C-23: solo cuenta lo que está por debajo del pin, y hacia el pin: %+v", status)
	}

	var out, errOut strings.Builder
	if code := RunDepsUpdate(cache, hcd039PinnedProbes(dir), &out, &errOut); code != 0 {
		t.Fatalf("deps-update: código %d\n%s%s", code, out.String(), errOut.String())
	}
	calls, _ := os.ReadFile(log)
	if got := strings.TrimSpace(string(calls)); got != "install "+CCExtensionsSource {
		t.Errorf("C-17: deps-update solo instala el pin, nunca la versión de npm: %q", got)
	}
}

// Tarea 11 (C-19, C-20, C-21, C-22): doctor muestra, al final y sin red, una
// línea por paquete fijado leída de la caché: WARN si npm publica algo por
// encima del pin, OK si está al día, INFO si lo de encima está rechazado o no
// hay datos.
func TestHCD039Doctor(t *testing.T) {
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "")
	checked := time.Date(2026, 11, 17, 9, 30, 0, 0, time.Local)
	write := func(t *testing.T, states ...DepState) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "deps.json")
		data, _ := json.Marshal(depsCache{CheckedAt: checked, Deps: states})
		writeFile(t, path, string(data))
		return path
	}
	itemOf := func(t *testing.T, items []CheckItem, pkg string) CheckItem {
		t.Helper()
		for _, it := range items {
			if strings.Contains(it.Name, pkg) {
				return it
			}
		}
		t.Fatalf("doctor no tiene línea para %s: %+v", pkg, items)
		return CheckItem{}
	}

	items := pinsDoctorItems(write(t,
		DepState{Name: "pi-antigravity", Latest: AntigravityVersion, Published: "99.0.0"},
		DepState{Name: "pi-cc-extensions", Latest: CCExtensionsVersion, Published: CCExtensionsVersion},
	))
	if len(items) != 2 {
		t.Fatalf("una línea por paquete fijado: %+v", items)
	}
	warn := itemOf(t, items, "pi-antigravity")
	for _, want := range []string{AntigravityVersion, "99.0.0", "no validada por pi-gomemory", "2026-11-17 09:30"} {
		if warn.Status != "WARN" || !strings.Contains(warn.Details, want) {
			t.Errorf("C-19: WARN con pin, versión de npm y fecha de la consulta (falta %q): %+v", want, warn)
		}
	}
	if ok := itemOf(t, items, "pi-cc-extensions"); ok.Status != "OK" || !strings.Contains(ok.Details, "2026-11-17 09:30") {
		t.Errorf("C-21: published igual al pin es OK con la fecha: %+v", ok)
	}

	old := RejectedVersions
	RejectedVersions = []RejectedVersion{{"pi-antigravity", "99.0.0", "rompe A1"}}
	t.Cleanup(func() { RejectedVersions = old })
	items = pinsDoctorItems(write(t, DepState{Name: "pi-antigravity", Published: AntigravityVersion}))
	if it := itemOf(t, items, "pi-antigravity"); it.Status != "INFO" || !strings.Contains(it.Details, "99.0.0 rechazada: rompe A1") {
		t.Errorf("C-20: lo rechazado se informa como INFO con su motivo, sin WARN: %+v", it)
	}
	RejectedVersions = old

	noData := "sin datos de npm; ejecuta pi-gomemory-setup deps-refresh"
	for name, path := range map[string]string{
		"caché ausente":        filepath.Join(t.TempDir(), "no-existe.json"),
		"binario anterior":     write(t, DepState{Name: "pi-antigravity", Latest: AntigravityVersion}),
		"solo published_error": write(t, DepState{Name: "pi-antigravity", PublishedError: "sin red"}),
	} {
		if it := itemOf(t, pinsDoctorItems(path), "pi-antigravity"); it.Status != "INFO" || !strings.Contains(it.Details, noData) {
			t.Errorf("C-22: %s → INFO «%s»: %+v", name, noData, it)
		}
	}
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "1")
	if it := itemOf(t, pinsDoctorItems(filepath.Join(t.TempDir(), "x.json")), "pi-antigravity"); !strings.Contains(it.Details, "comprobación desactivada") || strings.Contains(it.Details, "deps-refresh") {
		t.Errorf("C-22: con la comprobación desactivada no recomienda deps-refresh: %+v", it)
	}

	// RunDoctor añade estas líneas al final de la lista.
	t.Setenv("PATH", t.TempDir())
	opts := sandbox(t)
	writeFile(t, DefaultDepsCachePath(opts.PiAgentDir), `{"checked_at":"2026-11-17T09:30:00Z","deps":[{"name":"pi-antigravity","published":"99.0.0"}]}`)
	all := RunDoctor(opts)
	if n := len(all); n < 2 || !strings.Contains(all[n-2].Name, "pi-antigravity") || !strings.Contains(all[n-1].Name, "pi-cc-extensions") {
		t.Errorf("RunDoctor debe terminar con las líneas de los paquetes fijados: %+v", all)
	}
}

// Tarea 10 (C-24): una caché reciente sin published ni published_error en un
// paquete con Published (escrita por un binario anterior) se trata como
// caducada; con PI_GOMEMORY_NO_UPDATE_CHECK no se consulta ni se reescribe.
func TestHCD039StaleCacheRefresh(t *testing.T) {
	calls := 0
	probe := DepProbe{
		Name:      "pi-antigravity",
		Current:   func(context.Context) (string, error) { return AntigravityVersion, nil },
		Latest:    fixedVersion(AntigravityVersion),
		Published: func(context.Context) (string, error) { calls++; return "99.0.0", nil },
	}
	now := time.Now()
	write := func(t *testing.T, states ...DepState) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "deps.json")
		data, _ := json.Marshal(depsCache{CheckedAt: now.Add(-time.Minute), Deps: states})
		writeFile(t, path, string(data))
		return path
	}
	old := DepState{Name: "pi-antigravity", Current: AntigravityVersion, Latest: AntigravityVersion}

	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "")
	path := write(t, old)
	states := RefreshDeps([]DepProbe{probe}, path, now)
	if calls != 1 || hcd039StateOf(t, states, "pi-antigravity").Published != "99.0.0" {
		t.Errorf("C-24: la caché antigua debe refrescarse antes de la hora: llamadas=%d %+v", calls, states)
	}

	calls = 0
	withPub := old
	withPub.Published = AntigravityVersion
	RefreshDeps([]DepProbe{probe}, write(t, withPub), now)
	withErr := old
	withErr.PublishedError = "sin red"
	RefreshDeps([]DepProbe{probe}, write(t, withErr), now)
	if calls != 0 {
		t.Errorf("una caché reciente con published o published_error no debe volver a consultar: llamadas=%d", calls)
	}

	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "1")
	path = write(t, old)
	before, _ := os.ReadFile(path)
	RefreshDeps([]DepProbe{probe}, path, now)
	after, _ := os.ReadFile(path)
	if calls != 0 || string(before) != string(after) {
		t.Errorf("C-24: con la comprobación desactivada no consulta ni reescribe: llamadas=%d", calls)
	}
}

// Tarea 9 (C-18, C-25): la caché guarda published y published_error sin tocar
// Error ni Latest, también sin el paquete instalado; un probe sin Published
// se tolera.
func TestHCD039DepsPublished(t *testing.T) {
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "")
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "npm", "node_modules", "pi-antigravity", "package.json"), `{"version":"0.0.1"}`)
	probes := append(hcd039PinnedProbes(dir), DepProbe{
		Name:    "otro",
		Current: func(context.Context) (string, error) { return "1.0.0", nil },
		Latest:  func(context.Context) (string, error) { return "1.0.0", nil },
	})

	hcd039Registry(t, map[string]string{
		"pi-antigravity":   hcd039Doc(AntigravityVersion, "99.0.0"),
		"pi-cc-extensions": hcd039Doc(CCExtensionsVersion, "98.0.0"),
	})
	cache := filepath.Join(t.TempDir(), "deps.json")
	states := RefreshDeps(probes, cache, time.Now())
	ag := hcd039StateOf(t, states, "pi-antigravity")
	if ag.Published != "99.0.0" || ag.Latest != AntigravityVersion || ag.Error != "" || !ag.Outdated() {
		t.Errorf("pi-antigravity: published=99.0.0, latest=pin y sigue ofreciendo el pin: %+v", ag)
	}
	cc := hcd039StateOf(t, states, "pi-cc-extensions")
	if cc.Published != "98.0.0" || cc.Error == "" {
		t.Errorf("C-25: sin el paquete instalado la caché guarda igualmente published: %+v", cc)
	}
	if o := hcd039StateOf(t, states, "otro"); o.Published != "" || o.PublishedError != "" || o.Error != "" {
		t.Errorf("un probe sin Published no debe ganar campos: %+v", o)
	}
	raw, _ := os.ReadFile(cache)
	var onDisk depsCache
	if err := json.Unmarshal(raw, &onDisk); err != nil || hcd039StateOf(t, onDisk.Deps, "pi-antigravity").Published != "99.0.0" {
		t.Errorf("la caché en disco debe llevar published: %s", raw)
	}

	// Sin registro (bloqueado por TestMain): el error va a published_error y el
	// pin se sigue ofreciendo a una instalación por debajo de él (C-18).
	NPMRegistryURL = blockedNPMRegistry
	states = RefreshDeps(probes, "", time.Now())
	ag = hcd039StateOf(t, states, "pi-antigravity")
	if ag.PublishedError == "" || ag.Published != "" || ag.Error != "" || !ag.Outdated() {
		t.Errorf("C-18: sin npm, published_error y el pin se sigue ofreciendo: %+v", ag)
	}
}
