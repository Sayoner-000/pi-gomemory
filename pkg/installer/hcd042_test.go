package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Reproductores F-01/F-02 del seguimiento ACR/ADM 2026-11-17.
func TestFollowupInvalidVersionNotHealthy(t *testing.T) {
	t.Parallel()

	for _, versions := range [][2]string{{"1.0.0", "desconocida"}, {"dev", "1.0.0"}} {
		cache := hcd042Cache(t, DepState{Name: "x", Current: versions[0], Latest: versions[1]})
		probes := []DepProbe{{Name: "x", Current: fixedVersion(versions[0])}}
		var out, errs bytes.Buffer
		rc := RunDepsUpdate(cache, probes, &out, &errs)
		if rc == 0 || strings.Contains(out.String(), "están al día") {
			t.Errorf("uncomparable %v: rc=%d stdout=%q stderr=%q", versions, rc, out.String(), errs.String())
		}
	}
}

func TestFollowupLateEntryPreserved(t *testing.T) {
	t.Parallel()

	for _, kind := range []string{"cache", "manifest"} {
		for _, replacement := range []string{"symlink", "directory"} {
			t.Run(kind+"/"+replacement, func(t *testing.T) {
				dir := t.TempDir()
				file, target := filepath.Join(dir, "owned.json"), filepath.Join(dir, "personal.json")
				writeFile(t, file, "{}")
				writeFile(t, target, "SYNTH-PERSONAL")
				p := hcd042RemovalPlan(t, file, kind)
				if len(p.Actions) != 1 || p.Actions[0].apply == nil {
					t.Fatalf("unexpected plan: %+v", p.Actions)
				}
				if err := os.Remove(file); err != nil {
					t.Fatal(err)
				}
				if replacement == "symlink" {
					if err := os.Symlink(target, file); err != nil {
						t.Fatal(err)
					}
				} else if err := os.Mkdir(file, 0700); err != nil {
					t.Fatal(err)
				}
				err := p.Actions[0].apply()
				if _, statErr := os.Lstat(file); statErr != nil {
					t.Errorf("replacement removed: apply=%v lstat=%v", err, statErr)
				}
				if data, err := os.ReadFile(target); err != nil || string(data) != "SYNTH-PERSONAL" {
					t.Fatalf("target changed: %q %v", data, err)
				}
			})
		}
	}
}

func hcd042Cache(t *testing.T, states ...DepState) string {
	t.Helper()
	data, err := json.Marshal(depsCache{CheckedAt: time.Date(2026, 11, 17, 9, 30, 0, 0, time.UTC), Deps: states})
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "deps.json")
	writeFile(t, p, string(data))
	return p
}

func hcd042RemovalPlan(t *testing.T, file, kind string) *UninstallPlan {
	t.Helper()
	p := &UninstallPlan{}
	if kind == "manifest" {
		p.planManifest(file)
	} else {
		p.planPlainFile(file, "caché sintética")
	}
	hcd042RequireCloser(t, p)
	return p
}

func TestHCD042DepsComparable(t *testing.T) {
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "")
	for _, tc := range []struct {
		name, cachedCurrent, latest, live string
		wantUnknown                       bool
	}{
		{"latest inválida", "1.0.0", "desconocida", "1.0.0", true},
		{"current dev", "dev", "1.0.0", "dev", true},
		{"local dev con caché sana", "1.0.0", "1.0.0", "dev", true},
		{"caché inválida con local sana", "dev", "1.0.0", "1.0.0", false},
		{"estable sana", "1.0.0", "1.0.0", "1.0.0", false},
		{"prerelease no comparable", "1.0.0-rc.1", "1.0.0", "1.0.0-rc.1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cache := hcd042Cache(t, DepState{Name: "x", Current: tc.cachedCurrent, Latest: tc.latest})
			var out, errs bytes.Buffer
			rc := RunDepsUpdate(cache, []DepProbe{{Name: "x", Current: fixedVersion(tc.live)}}, &out, &errs)
			if tc.wantUnknown {
				if rc != 1 || strings.Contains(out.String(), "al día") || !strings.Contains(errs.String(), "x: estado desconocido") || !strings.Contains(errs.String(), "deps-refresh") {
					t.Fatalf("rc=%d stdout=%q stderr=%q", rc, out.String(), errs.String())
				}
			} else if rc != 0 || !strings.Contains(out.String(), "están al día") {
				t.Fatalf("rc=%d stdout=%q stderr=%q", rc, out.String(), errs.String())
			}
		})
	}

	// Un desconocido no bloquea la actualización conocida; no se ejecutan
	// comandos que vengan de caché ni se vuelve a consultar Latest en red.
	t.Run("actualización parcial", func(t *testing.T) {
		marker := filepath.Join(t.TempDir(), "updated")
		cache := hcd042Cache(t,
			DepState{Name: "unknown", Current: "dev", Latest: "2.0.0", Update: []string{"false"}},
			DepState{Name: "known", Current: "1.0.0", Latest: "2.0.0", Update: []string{"false"}},
		)
		latest := func(context.Context) (string, error) { t.Error("Latest no debe consultarse"); return "", nil }
		probes := []DepProbe{
			{Name: "unknown", Current: fixedVersion("dev"), Latest: latest, Update: []string{"false"}},
			{Name: "known", Current: fixedVersion("1.0.0"), Latest: latest, Update: []string{"sh", "-c", `printf updated > "$1"`, "sh", marker}},
		}
		var out, errs bytes.Buffer
		if rc := RunDepsUpdate(cache, probes, &out, &errs); rc != 1 || strings.Contains(out.String(), "al día") || !strings.Contains(errs.String(), "unknown: estado desconocido") {
			t.Fatalf("rc=%d stdout=%q stderr=%q", rc, out.String(), errs.String())
		}
		if data, err := os.ReadFile(marker); err != nil || string(data) != "updated" {
			t.Fatalf("conocido no actualizado: %q %v", data, err)
		}
	})

	for _, pin := range PinnedPackages() {
		t.Run("pin efectivo "+pin.Name, func(t *testing.T) {
			cache := hcd042Cache(t, DepState{Name: pin.Name, Current: "dev", Latest: "desconocida"})
			var out, errs bytes.Buffer
			probes := []DepProbe{{Name: pin.Name, Current: fixedVersion(pin.Version)}}
			if rc := RunDepsUpdate(cache, probes, &out, &errs); rc != 0 || !strings.Contains(out.String(), "están al día") {
				t.Fatalf("pin/local deben prevalecer sobre caché: rc=%d stdout=%q stderr=%q", rc, out.String(), errs.String())
			}
		})
	}
}

func TestHCD042Removal(t *testing.T) {
	t.Parallel()

	for _, kind := range []string{"cache", "manifest"} {
		for _, typ := range []string{"symlink", "broken symlink", "directory"} {
			t.Run(kind+"/preexisting/"+typ, func(t *testing.T) {
				dir := t.TempDir()
				file, target := filepath.Join(dir, "owned"), filepath.Join(dir, "personal")
				if typ == "directory" {
					if err := os.Mkdir(file, 0700); err != nil {
						t.Fatal(err)
					}
				} else {
					if typ == "symlink" {
						writeFile(t, target, "SYNTH-PERSONAL")
					}
					if err := os.Symlink(target, file); err != nil {
						t.Fatal(err)
					}
				}
				p := hcd042RemovalPlan(t, file, kind)
				if p.Pending() != 0 {
					t.Fatalf("no debe planificar retirada: %+v", p.Actions)
				}
				for _, a := range p.Actions {
					if a.apply != nil {
						t.Fatal("entrada ajena no debe tener acción destructiva")
					}
				}
				if _, err := os.Lstat(file); err != nil {
					t.Fatal(err)
				}
			})
		}
		for _, mutation := range []string{"unchanged", "replaced regular", "broken symlink", "gone"} {
			t.Run(kind+"/after/"+mutation, func(t *testing.T) {
				dir := t.TempDir()
				file := filepath.Join(dir, "owned")
				writeFile(t, file, "original")
				p := hcd042RemovalPlan(t, file, kind)
				if len(p.Actions) != 1 || p.Actions[0].apply == nil {
					t.Fatalf("plan: %+v", p.Actions)
				}
				if (kind == "manifest") != p.Actions[0].ifClean {
					t.Fatalf("ifClean incorrecto: %+v", p.Actions[0])
				}
				switch mutation {
				case "replaced regular":
					// Q2: el plan retiene el original, no hace falta renombrarlo.
					if err := os.Remove(file); err != nil {
						t.Fatal(err)
					}
					writeFile(t, file, "SYNTH-NEW")
				case "broken symlink":
					if err := os.Remove(file); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(filepath.Join(dir, "missing"), file); err != nil {
						t.Fatal(err)
					}
				case "gone":
					if err := os.Remove(file); err != nil {
						t.Fatal(err)
					}
				}
				err := p.Actions[0].apply()
				switch mutation {
				case "unchanged":
					if err != nil {
						t.Fatal(err)
					}
					if _, err := os.Lstat(file); !os.IsNotExist(err) {
						t.Fatalf("no se retiró: %v", err)
					}
				case "gone":
					if !errors.Is(err, errGoneAfterPlan) {
						t.Fatalf("esperado gone: %v", err)
					}
				default:
					if !errors.Is(err, errChangedAfterPlan) {
						t.Fatalf("esperado changed: %v", err)
					}
					if _, err := os.Lstat(file); err != nil {
						t.Fatalf("se perdió entrada nueva: %v", err)
					}
					if mutation == "replaced regular" {
						if data, err := os.ReadFile(file); err != nil || string(data) != "SYNTH-NEW" {
							t.Fatalf("contenido nuevo: %q %v", data, err)
						}
					}
				}
			})
		}
	}
}
