package installer

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// HCD-038: cierra H-01 y H-02 de la ADM 2026-11-17. Los dos primeros tests son
// el reproductor de la auditoría (docs/audits/2026-11-17-adm-repro_test.go.txt)
// sin cambios de contenido.

func TestADM20261117StreamOutputBound(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import assert from "node:assert/strict";
import { streamLines } from "./gomemory-tdd.ts";
const result = await streamLines(process.execPath,
 ["-e", "process.stdout.write('x'.repeat(8 * 1024 * 1024))"],
 { timeoutMs: 10000 }, () => {});
const retained = result.tail.reduce((n, s) => n + Buffer.byteLength(s), 0);
console.log(JSON.stringify({code: result.code, lines: result.tail.length, retained}));
assert.ok(retained <= 65536, 'diagnóstico sin cota de bytes: ' + retained);
`)
	t.Log(out)
}

func TestADM20261117QuotaRejectsMissingData(t *testing.T) {
	for _, body := range []string{"null", "{}"} {
		t.Run(body, func(t *testing.T) {
			requests := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			}))
			defer srv.Close()
			old := endpoints
			endpoints = []string{srv.URL, srv.URL}
			defer func() { endpoints = old }()
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "antigravity-accounts.json"), `{"activeAccountId":"synthetic","accounts":{"synthetic":{"access":"SYNTH-TOKEN-0001"}}}`)
			summary, err := FetchQuotaSummary(dir)
			t.Logf("summary=%+v error=%v requests=%d", summary, err, requests)
			if err == nil {
				t.Error("respuesta sin cuotas aceptada por FetchQuotaSummary")
			}
			requests = 0
			text, err := FetchAndFormatUsage(dir)
			t.Logf("text=%q error=%v requests=%d", text, err, requests)
			if err == nil {
				t.Error("respuesta sin cuotas aceptada por FetchAndFormatUsage")
			}
		})
	}
}

// Tarea 2 (Q1, Q2): una línea de más de 64 Ki caracteres termina el proceso,
// deja el aviso en el diagnóstico y devuelve código 1; una de 64 Ki exactos se
// acepta entera. Vale tanto si la línea nunca termina como si termina tarde.
func TestHCD038LongLineFails(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { streamLines } from "./gomemory-tdd.ts";
const fails = [];
const LIMIT = 64 * 1024;
const run = async (script) => {
	const lines = [];
	const t0 = Date.now();
	const res = await streamLines(process.execPath, ["-e", script], { timeoutMs: 20000 }, (l) => lines.push(l));
	return { res, lines, ms: Date.now() - t0 };
};
// Sin salto de línea y el proceso sigue vivo: debe cortarse sin esperar al timeout.
const open = await run("process.stdout.write('x'.repeat(" + (LIMIT + 10) + ")); setTimeout(() => {}, 30000)");
if (open.res.code !== 1) fails.push("línea abierta: código " + open.res.code + ", se esperaba 1");
if (open.res.timedOut) fails.push("línea abierta: debió cortarse por el límite, no por timeout");
if (open.ms > 10000) fails.push("línea abierta: tardó " + open.ms + " ms");
if (!open.res.tail.some((l) => l.includes("Salida rechazada"))) fails.push("línea abierta: falta el aviso en tail: " + JSON.stringify(open.res.tail.map((l) => l.slice(0, 80))));
// Línea excesiva terminada con salto de línea en un solo chunk.
const closed = await run("process.stdout.write('y'.repeat(" + (LIMIT + 1) + ") + '\\n')");
if (closed.res.code !== 1) fails.push("línea cerrada: código " + closed.res.code + ", se esperaba 1");
if (closed.lines.some((l) => l.length > LIMIT)) fails.push("línea cerrada: el parser recibió una línea de más de 64 Ki");
// Exactamente en el límite: se admite y el parser la recibe entera.
const exact = await run("process.stdout.write('z'.repeat(" + LIMIT + ") + '\\nfin\\n')");
if (exact.res.code !== 0) fails.push("límite exacto: código " + exact.res.code + ", se esperaba 0");
if (exact.lines[0]?.length !== LIMIT || exact.lines[1] !== "fin") fails.push("límite exacto: el parser no recibió las líneas completas");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD038LongLineFails: %s", out)
	}
}

// Tarea 3 (Q2): tail guarda como mucho 40 extractos de 1 Ki caracteres y 64 KiB
// en bytes UTF-8, con muchas líneas grandes multibyte en stdout y stderr a la vez;
// el parser sigue recibiendo cada línea completa y tail conserva la última.
func TestHCD038TailBudget(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { streamLines } from "./gomemory-tdd.ts";
const fails = [];
// 100 líneas de 5000 «€» (3 bytes cada uno) por stream, intercaladas, y una marca final.
const script = "const l = '€'.repeat(5000) + '\\n';" +
	"for (let i = 0; i < 100; i++) { process.stdout.write(l); process.stderr.write(l); }" +
	"process.stdout.write('FIN\\n');";
const lines = [];
const res = await streamLines(process.execPath, ["-e", script], { timeoutMs: 20000 }, (l) => lines.push(l));
const bytes = res.tail.reduce((n, s) => n + Buffer.byteLength(s), 0);
if (res.code !== 0) fails.push("código " + res.code + ", se esperaba 0");
if (lines.length !== 201) fails.push("el parser recibió " + lines.length + " líneas, se esperaban 201");
// stdout y stderr son tuberías distintas: FIN puede llegar antes que las últimas
// líneas de stderr (Constitución §10, sin depender del orden). Se comprueba el
// contenido recibido y que tail conserve la última línea que llegó.
if (lines.filter((l) => l.length === 5000).length !== 200 || lines.filter((l) => l === "FIN").length !== 1) fails.push("el parser debe recibir cada línea completa");
if (res.tail.length > 40) fails.push("tail tiene " + res.tail.length + " líneas, máximo 40");
if (res.tail.some((l) => l.length > 1024)) fails.push("tail guarda extractos de más de 1 Ki caracteres");
if (bytes > 65536) fails.push("tail retiene " + bytes + " bytes, máximo 65536");
const last = lines.at(-1) ?? "";
if (res.tail.at(-1) !== last.slice(-1024)) fails.push("tail debe conservar la última línea recibida, termina en " + JSON.stringify(res.tail.at(-1)?.slice(0, 20)));
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD038TailBudget: %s", out)
	}
}

// Tarea 4: el check hcd lee la unidad del JSON completo de una línea de
// `mem hcd show --json`, aunque supere el extracto de 1 Ki que guarda tail.
func TestHCD038HcdCheckLongJSON(t *testing.T) {
	t.Parallel()

	proj := t.TempDir()
	bin := t.TempDir()
	writeExec(t, filepath.Join(bin, "mem"), `printf '{"ok":true,"unit_id":"HCD-999","pad":"%s"}\n' "$(printf 'x%.0s' $(seq 1 3000))"`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
`+tddToolHarness+`
const res = await run("data", `+jsStr(proj)+`);
const names = (suiteOf(res.details, "data")?.tests || []).map((x) => x.name + "=" + x.status);
if (!names.includes("Unidad HCD activa: HCD-999=passed")) fails.push("el check hcd debe mostrar la unidad HCD-999: " + JSON.stringify(names));
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD038HcdCheckLongJSON: %s", out)
	}
}

// Tarea 5: parseQuotaResponse solo acepta respuestas con al menos un bucket y
// fracciones en [0,1]; 0 y 1 son valores legítimos.
func TestHCD038QuotaParse(t *testing.T) {
	t.Parallel()

	bucket := func(frac string) string {
		return `{"groups":[{"displayName":"G","buckets":[{"displayName":"B","remainingFraction":` + frac + `}]}]}`
	}
	for name, body := range map[string]string{
		"null":             `null`,
		"vacío":            `{}`,
		"sin grupos":       `{"groups":[]}`,
		"grupo sin bucket": `{"groups":[{"displayName":"G","buckets":[]}]}`,
		"fracción > 1":     bucket("1.5"),
		"fracción < 0":     bucket("-0.1"),
		"ilegible":         `{"groups":`,
	} {
		if q, err := parseQuotaResponse([]byte(body)); err == nil {
			t.Errorf("%s: debe rechazarse, devolvió %+v", name, q)
		}
	}
	for _, frac := range []string{"0", "0.5", "1"} {
		q, err := parseQuotaResponse([]byte(bucket(frac)))
		if err != nil || len(q.Groups) != 1 || len(q.Groups[0].Buckets) != 1 {
			t.Errorf("fracción %s: debe aceptarse, err=%v q=%+v", frac, err, q)
		}
	}
}

// hcd038QuotaServers sirve primero y segundo cuerpo en dos endpoints y cuenta
// las peticiones a cada uno; deja una cuenta sintética en el directorio devuelto.
func hcd038QuotaServers(t *testing.T, first, second string) (dir string, hits *[2]int) {
	t.Helper()
	hits = &[2]int{}
	serve := func(i int, body string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits[i]++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		return srv
	}
	a, b := serve(0, first), serve(1, second)
	old := endpoints
	endpoints = []string{a.URL, b.URL}
	t.Cleanup(func() { endpoints = old })
	dir = t.TempDir()
	writeFile(t, filepath.Join(dir, "antigravity-accounts.json"), `{"activeAccountId":"synthetic","accounts":{"synthetic":{"access":"SYNTH-TOKEN-0001","email":"synthetic@example.invalid"}}}`)
	return dir, hits
}

const hcd038ValidQuota = `{"planLabel":"Plan Sintético","groups":[{"displayName":"Modelos","buckets":[{"displayName":"Bucket-A","remainingFraction":0.25,"resetTime":""}]}]}`

// Tarea 6: si el primer endpoint responde {} se usa el segundo, en usage y usage --json.
func TestHCD038QuotaFallback(t *testing.T) {
	dir, hits := hcd038QuotaServers(t, `{}`, hcd038ValidQuota)
	summary, err := FetchQuotaSummary(dir)
	if err != nil || summary.PlanLabel != "Plan Sintético" || len(summary.Groups) != 1 || summary.Groups[0].Buckets[0].RemainingPercent != 25 {
		t.Errorf("FetchQuotaSummary debe devolver la cuota del segundo endpoint: %+v err=%v", summary, err)
	}
	text, err := FetchAndFormatUsage(dir)
	if err != nil || !strings.Contains(text, "Bucket-A") || !strings.Contains(text, "Plan Sintético") {
		t.Errorf("FetchAndFormatUsage debe mostrar la cuota del segundo endpoint: %q err=%v", text, err)
	}
	if hits[0] != 2 || hits[1] != 2 {
		t.Errorf("cada función debe consultar los dos endpoints: peticiones=%v", *hits)
	}
}

// Tarea 7 (Q3): sin planLabel no se atribuye un plan: «Plan no informado».
func TestHCD038PlanLabelMissing(t *testing.T) {
	noLabel := strings.Replace(hcd038ValidQuota, `"planLabel":"Plan Sintético",`, "", 1)
	dir, _ := hcd038QuotaServers(t, noLabel, noLabel)
	summary, err := FetchQuotaSummary(dir)
	if err != nil || summary.PlanLabel != "Plan no informado" {
		t.Errorf("usage --json debe decir «Plan no informado»: %+v err=%v", summary, err)
	}
	text, err := FetchAndFormatUsage(dir)
	if err != nil || !strings.Contains(text, "Plan no informado") || strings.Contains(text, "g1-pro-tier") {
		t.Errorf("usage debe decir «Plan no informado» sin inventar el plan: %q err=%v", text, err)
	}
}
