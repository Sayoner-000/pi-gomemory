package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Reproductor de H-01 de la ACR/ADM 2026-11-17 (docs/audits/2026-11-17-acr-adm-repro_test.go.txt).
// Desde HCD-085 el paquete fijado de la prueba es pi-cc-extensions.
func TestACRADM01NewPinOverridesOldCache(t *testing.T) {
	t.Parallel()

	p := filepath.Join(t.TempDir(), "cache.json")
	b, _ := json.Marshal(depsCache{CheckedAt: time.Now(), Deps: []DepState{{Name: "pi-cc-extensions", Current: "1.0.16", Latest: "1.0.16"}}})
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	probes := []DepProbe{{Name: "pi-cc-extensions", Current: fixedVersion("1.0.16"), Latest: fixedVersion(CCExtensionsVersion), Update: []string{"pi", "install", CCExtensionsSource}}}
	got := DepsStatusFor(p, probes)
	if len(got) != 1 || got[0].Latest != CCExtensionsVersion {
		t.Fatalf("old cache hides new approved pin %s: got %+v", CCExtensionsVersion, got)
	}
}

// Tarea 10 (H-01, Q2): con una caché anterior al cambio de pin, DepsStatusFor
// ofrece exactamente el pin compilado, no lo que diga la caché, y los demás
// componentes siguen usando la latest guardada.
func TestHCD040StaleCacheOffersNewPin(t *testing.T) {
	t.Parallel()

	probe := func(name, current, source string) DepProbe {
		return DepProbe{Name: name, Current: fixedVersion(current), Update: []string{"pi", "install", source}}
	}
	cache := func(t *testing.T, checked time.Time, states ...DepState) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "cache.json")
		b, _ := json.Marshal(depsCache{CheckedAt: checked, Deps: states})
		writeFile(t, p, string(b))
		return p
	}
	names := func(ds []DepState) string {
		var out []string
		for _, d := range ds {
			out = append(out, d.Name+"→"+d.Latest)
		}
		return strings.Join(out, ",")
	}
	for name, checked := range map[string]time.Time{"vigente": time.Now(), "caducada": time.Now().Add(-48 * time.Hour)} {
		t.Run(name, func(t *testing.T) {
			p := cache(t, checked,
				DepState{Name: "pi-gomemory", Current: "v1.0.0", Latest: "v1.1.0"},
				DepState{Name: "mem", Current: "1.0.0", Latest: "2.0.0"},
				DepState{Name: "pi-cc-extensions", Current: "1.0.16", Latest: "1.0.16"},
			)
			got := DepsStatusFor(p, []DepProbe{
				{Name: "pi-gomemory", Current: fixedVersion("v1.0.0"), Update: []string{"pi-gomemory-setup", "update"}},
				{Name: "mem", Current: fixedVersion("1.0.0"), Update: []string{"mem", "update", "-y"}},
				probe("pi-cc-extensions", "1.0.16", CCExtensionsSource),
			})
			want := "pi-gomemory→v1.1.0,mem→2.0.0,pi-cc-extensions→" + CCExtensionsVersion
			if names(got) != want {
				t.Fatalf("se ofrece el pin compilado y el resto no cambia:\n got %s\nwant %s", names(got), want)
			}
			for _, d := range got[2:] {
				if src := d.Update[len(d.Update)-1]; !strings.HasSuffix(src, "@"+d.Latest) {
					t.Errorf("%s: se ofrece la fuente compilada exacta, no %q", d.Name, src)
				}
			}
		})
	}

	// Instalado ya en el pin, o caché con una latest superior a la aprobada:
	// no se ofrece nada por encima del pin.
	p := cache(t, time.Now(),
		DepState{Name: "pi-cc-extensions", Current: "1.0.16", Latest: "0.99.0"},
	)
	got := DepsStatusFor(p, []DepProbe{
		probe("pi-cc-extensions", CCExtensionsVersion, CCExtensionsSource),
	})
	if len(got) != 0 {
		t.Errorf("con el pin instalado no se ofrece nada, aunque la caché diga 0.99.0: %s", names(got))
	}
}

// Tarea 2 (Q1): uninstall --packages retira también las entradas de pins
// anteriores de pi-gomemory, con pi remove de esa fuente; una versión que
// pi-gomemory nunca fijó se conserva.
func TestHCD040HistoricPins(t *testing.T) {
	run := func(t *testing.T, packages ...any) (settings, piCalls, out string) {
		t.Helper()
		opts := installSandbox(t)
		path := filepath.Join(opts.PiAgentDir, "settings.json")
		editJSONFile(t, path, func(m map[string]any) { m["packages"] = packages })
		bin := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))[0]
		log := filepath.Join(opts.HomeDir, "pi.log")
		writeExec(t, filepath.Join(bin, "pi"), `echo "$*" >> "`+log+`"`)
		fakeTerminal(t, false, "")
		out, err := runUninstall(t, opts, UninstallFlags{Packages: true, Yes: true})
		if err != nil {
			t.Fatalf("uninstall --packages: %v\n%s", err, out)
		}
		calls, _ := os.ReadFile(log)
		return readFile(t, path), string(calls), out
	}

	settings, calls, out := run(t, "npm:pi-antigravity@0.12.0", "npm:pi-cc-extensions@1.0.16", "npm:otro@1.0.0")
	if strings.Contains(settings, "pi-antigravity@0.12.0") || strings.Contains(settings, "pi-cc-extensions@1.0.16") || !strings.Contains(settings, "npm:otro@1.0.0") {
		t.Errorf("las entradas de pins anteriores se retiran y el resto se conserva:\n%s\n%s", settings, out)
	}
	for _, want := range []string{"remove npm:pi-antigravity@0.12.0", "remove npm:pi-cc-extensions@1.0.16"} {
		if !strings.Contains(calls, want) {
			t.Errorf("falta «pi %s»; llamadas:\n%s", want, calls)
		}
	}

	settings, calls, _ = run(t, "npm:pi-antigravity@0.13.0")
	if !strings.Contains(settings, "npm:pi-antigravity@0.13.0") || strings.Contains(calls, "remove") {
		t.Errorf("una versión que pi-gomemory nunca fijó se conserva sin pi remove:\n%s\n%s", settings, calls)
	}
	for _, src := range historicPinnedSources() {
		if src == CCExtensionsSource {
			t.Errorf("el pin actual no es histórico: %s", src)
		}
	}
}

// HCD-040: subida de los pins a pi-antigravity 0.14.0 y pi-cc-extensions 1.0.20.

// Tarea 1: los pins y sus fuentes apuntan a las versiones validadas. HCD-063
// sube pi-antigravity a 0.15.0 tras probarlo con la cuenta real; HCD-078 sube
// pi-cc-extensions a 1.0.22 tras validarla con pins-validate.
func TestHCD040PinsBumped(t *testing.T) {
	t.Parallel()

	// HCD-085: todos los pins de pi-antigravity son históricos, para que install
	// y uninstall --packages retiren las entradas que escribió pi-gomemory.
	for _, v := range []string{"0.12.0", "0.14.0", "0.15.0"} {
		if !isHistoricPin("pi-antigravity", "npm:pi-antigravity@"+v) {
			t.Errorf("pi-antigravity %s debe quedar como pin histórico", v)
		}
	}
	if CCExtensionsVersion != "1.0.22" || CCExtensionsSource != "npm:pi-cc-extensions@1.0.22" {
		t.Errorf("pi-cc-extensions: %s / %s, se espera 1.0.22", CCExtensionsVersion, CCExtensionsSource)
	}
	if !isHistoricPin("pi-cc-extensions", "npm:pi-cc-extensions@1.0.21") {
		t.Error("1.0.21 debe quedar como pin histórico para que uninstall --packages lo retire")
	}
}
