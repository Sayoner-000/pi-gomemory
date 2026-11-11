package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// C-02: ningún probe ejecuta un script descargado por tubería.
func TestHCD012DepsNoPipe(t *testing.T) {
	for _, p := range DefaultDepProbes("v1.0.0", t.TempDir()) {
		cmd := strings.Join(p.Update, " ")
		for _, bad := range []string{"curl", "| bash", "| sh", "|bash", "|sh"} {
			if strings.Contains(cmd, bad) {
				t.Errorf("%s: comando inseguro %q", p.Name, cmd)
			}
		}
		if p.Name == "codebase-memory-mcp" && (len(p.Update) != 0 || !strings.Contains(p.Manual, "releases/latest")) {
			t.Errorf("codebase-memory-mcp debe ser manual: %+v", p)
		}
	}
}

func writeDepsCache(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "deps.json")
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// C-09: el comando sale del probe actual, nunca de la caché.
func TestHCD012DepsCacheIgnored(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "pwned")
	cache := writeDepsCache(t, `{"deps":[
 {"name":"x","current":"1.0.0","latest":"2.0.0","update":["sh","-c","touch `+marker+`"]},
 {"name":"ghost","current":"1.0.0","latest":"2.0.0","update":["sh","-c","touch `+marker+`"]}]}`)
	probes := []DepProbe{{Name: "x", Update: []string{"true"}}}
	st := DepsStatusFor(cache, probes)
	if len(st) != 1 || st[0].Name != "x" || strings.Join(st[0].Update, " ") != "true" {
		t.Fatalf("status = %+v", st)
	}
	var out, errb bytes.Buffer
	if code := RunDepsUpdate(cache, probes, &out, &errb); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errb.String())
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("se ejecutó el comando de la caché")
	}

	only := writeDepsCache(t, `{"deps":[{"name":"ghost","current":"1.0.0","latest":"2.0.0","update":["false"]}]}`)
	if s := DepsStatusFor(only, probes); len(s) != 0 {
		t.Fatalf("ghost no debe listarse: %+v", s)
	}
	out.Reset()
	if code := RunDepsUpdate(only, probes, &out, &errb); code != 0 || !strings.Contains(out.String(), "están al día") {
		t.Fatalf("code=%d out=%s", code, out.String())
	}
}

// C-01: un componente manual no se ejecuta ni se declara actualizado.
func TestHCD012DepsManual(t *testing.T) {
	cache := writeDepsCache(t, `{"deps":[{"name":"m","current":"1.0.0","latest":"2.0.0","update":["false"]}]}`)
	probes := []DepProbe{{Name: "m", Manual: "https://example.com/releases/latest"}}
	if n := FormatDepsNotice(DepsStatusFor(cache, probes)); !strings.Contains(n, "releases/latest") {
		t.Fatalf("aviso sin URL: %s", n)
	}
	var out, errb bytes.Buffer
	code := RunDepsUpdate(cache, probes, &out, &errb)
	s := out.String() + errb.String()
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	if strings.Contains(s, "Dependencias actualizadas") {
		t.Fatalf("no debe declarar actualizado: %s", s)
	}
	if !strings.Contains(s, "actualízalo manualmente: https://example.com/releases/latest") {
		t.Fatalf("falta instrucción manual: %s", s)
	}
}
