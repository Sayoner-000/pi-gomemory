package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// HCD-084: skills externas (impeccable, frontend-design) desde skills.sh y
// gitlab-release embebida.

// fakeNpx pone en el PATH un npx que registra sus argumentos en $HOME/npx.log
// y, con `skills add … -s <skill>`, crea ~/.agents/skills/<skill>/SKILL.md.
// NPX_FAIL hace que falle; NPX_BUMP cambia el hash del lockfile en update.
func fakeNpx(t *testing.T, opts *Options) string {
	t.Helper()
	dir := t.TempDir()
	script := `#!/bin/sh
echo "$*" >> "$HOME/npx.log"
[ -n "$NPX_FAIL" ] && exit 3
if [ "$2" = skills ] && [ "$3" = add ]; then
  n=""; prev=""
  for a in "$@"; do [ "$prev" = "-s" ] && n="$a"; prev="$a"; done
  mkdir -p "$HOME/.agents/skills/$n"
  printf -- "---\nname: %s\ndescription: x\n---\n" "$n" > "$HOME/.agents/skills/$n/SKILL.md"
fi
if [ "$2" = skills ] && [ "$3" = update ] && [ -n "$NPX_BUMP" ]; then
  printf '{"version":3,"skills":{"impeccable":{"source":"pbakaus/impeccable","skillFolderHash":"%s"},"frontend-design":{"source":"anthropics/skills","skillFolderHash":"h0"},"playwright-cli":{"source":"microsoft/playwright-cli","skillFolderHash":"h0"}}}' "$NPX_BUMP" > "$HOME/.agents/.skill-lock.json"
fi
exit 0
`
	writeFile(t, filepath.Join(dir, "npx"), script)
	if err := os.Chmod(filepath.Join(dir, "npx"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+"/usr/bin:/bin")
	t.Setenv(skipDepsEnv, "")
	t.Setenv("NPX_FAIL", "")
	t.Setenv("NPX_BUMP", "")
	return filepath.Join(opts.HomeDir, "npx.log")
}

func npxLog(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func hostSkill(t *testing.T, opts *Options, name string) {
	t.Helper()
	writeFile(t, filepath.Join(opts.HomeDir, ".agents", "skills", name, "SKILL.md"), "---\nname: "+name+"\ndescription: x\n---\n")
}

// lockSkills registra las skills en ~/.agents/.skill-lock.json con su fuente
// oficial del catálogo, como hace `npx skills add`.
func lockSkills(t *testing.T, opts *Options, names ...string) {
	t.Helper()
	sources := map[string]string{}
	for _, s := range ExternalSkills() {
		sources[s.Name] = s.Source
	}
	var parts []string
	for _, n := range names {
		parts = append(parts, `"`+n+`":{"source":"`+sources[n]+`","skillFolderHash":"h0"}`)
	}
	writeFile(t, filepath.Join(opts.HomeDir, ".agents", ".skill-lock.json"), `{"version":3,"skills":{`+strings.Join(parts, ",")+`}}`)
}

// T1: catálogo de skills externas con su fuente oficial y comandos.
func TestHCD084Catalog(t *testing.T) {
	got := ExternalSkills()
	want := []ExternalSkill{
		{Name: "impeccable", Source: "pbakaus/impeccable"},
		{Name: "frontend-design", Source: "anthropics/skills"},
		{Name: "playwright-cli", Source: "microsoft/playwright-cli"},
	}
	if len(got) != len(want) {
		t.Fatalf("ExternalSkills = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ExternalSkills[%d] = %+v, quiero %+v", i, got[i], want[i])
		}
	}
	s := want[0]
	if a := strings.Join(externalSkillAddArgs(s), " "); a != "-y skills add pbakaus/impeccable -g -a pi -s impeccable -y" {
		t.Fatalf("add args = %q", a)
	}
	if a := strings.Join(externalSkillUpdateArgs([]string{"impeccable", "frontend-design"}), " "); a != "-y skills update impeccable frontend-design -g -y" {
		t.Fatalf("update args = %q", a)
	}
	if m := s.Manual(); m != "npx skills add pbakaus/impeccable -g -a pi -s impeccable -y" {
		t.Fatalf("manual = %q", m)
	}
	if r := s.RemoveCmd(); r != "npx skills remove impeccable -g -y" {
		t.Fatalf("remove = %q", r)
	}
}

// T2: instala las ausentes (y adopta las presentes que skills no gestiona) y
// actualiza siempre las gestionadas.
func TestHCD084InstallAndUpdate(t *testing.T) {
	opts := sandbox(t)
	log := fakeNpx(t, opts)
	hostSkill(t, opts, "frontend-design")
	lockSkills(t, opts, "frontend-design")
	// Copia manual sin entrada en el lockfile: `skills update` no la tocaría.
	hostSkill(t, opts, "playwright-cli")

	var out bytes.Buffer
	res := UpdateExternalSkills(opts, &out)
	lines := npxLog(t, log)
	wantLines := []string{
		"-y skills add pbakaus/impeccable -g -a pi -s impeccable -y",
		"-y skills add microsoft/playwright-cli -g -a pi -s playwright-cli -y",
		"-y skills update frontend-design -g -y",
	}
	if strings.Join(lines, "|") != strings.Join(wantLines, "|") {
		t.Fatalf("npx llamado con:\n%v\nquiero:\n%v", lines, wantLines)
	}
	if strings.Join(res.Installed, ",") != "impeccable,playwright-cli" || len(res.Failed) != 0 {
		t.Fatalf("resultado = %+v\n%s", res, out.String())
	}
	if !ExternalSkillPresent(opts.HomeDir, "impeccable") {
		t.Fatal("impeccable no quedó en ~/.agents/skills")
	}

	// Segunda vez: todas gestionadas; se actualizan siempre, sin add.
	_ = os.Remove(log)
	lockSkills(t, opts, "impeccable", "frontend-design", "playwright-cli")
	t.Setenv("NPX_BUMP", "nuevo")
	res = UpdateExternalSkills(opts, &out)
	lines = npxLog(t, log)
	if len(lines) != 1 || lines[0] != "-y skills update impeccable frontend-design playwright-cli -g -y" {
		t.Fatalf("segunda pasada: %v", lines)
	}
	if strings.Join(res.Changed, ",") != "impeccable" {
		t.Fatalf("cambio del lockfile no detectado: %+v", res)
	}

	// C-001: si el lockfile registra una skill homónima desde otro repositorio,
	// se adopta desde la fuente oficial con add en lugar de actualizar el fork.
	_ = os.Remove(log)
	writeFile(t, filepath.Join(opts.HomeDir, ".agents", ".skill-lock.json"),
		`{"version":3,"skills":{"impeccable":{"source":"someone/fork","skillFolderHash":"h0"},"frontend-design":{"source":"anthropics/skills","skillFolderHash":"h0"},"playwright-cli":{"source":"microsoft/playwright-cli","skillFolderHash":"h0"}}}`)
	t.Setenv("NPX_BUMP", "")
	res = UpdateExternalSkills(opts, &out)
	lines = npxLog(t, log)
	wantForkLines := []string{
		"-y skills add pbakaus/impeccable -g -a pi -s impeccable -y",
		"-y skills update frontend-design playwright-cli -g -y",
	}
	if strings.Join(lines, "|") != strings.Join(wantForkLines, "|") {
		t.Fatalf("lockfile con otro origen:\n%v\nquiero:\n%v", lines, wantForkLines)
	}
	if strings.Join(res.Installed, ",") != "impeccable" {
		t.Fatalf("impeccable debía reinstalarse desde fuente oficial: %+v", res)
	}
}

// T3: sin npx, con PI_GOMEMORY_SKIP_DEPS, desinstalado o con fallo: avisa y sigue.
func TestHCD084Degrade(t *testing.T) {
	t.Run("sin npx", func(t *testing.T) {
		opts := sandbox(t)
		t.Setenv("PATH", t.TempDir())
		var out bytes.Buffer
		res := UpdateExternalSkills(opts, &out)
		if res.Skipped == "" || !strings.Contains(out.String(), "npx skills add pbakaus/impeccable -g -a pi -s impeccable -y") {
			t.Fatalf("sin npx debe avisar con el comando manual: %+v\n%s", res, out.String())
		}
	})
	t.Run("skip deps", func(t *testing.T) {
		opts := sandbox(t)
		log := fakeNpx(t, opts)
		t.Setenv(skipDepsEnv, "1")
		res := UpdateExternalSkills(opts, &bytes.Buffer{})
		if res.Skipped == "" || len(npxLog(t, log)) != 0 {
			t.Fatalf("PI_GOMEMORY_SKIP_DEPS no omitió npx: %+v", res)
		}
	})
	t.Run("desinstalado", func(t *testing.T) {
		opts := sandbox(t)
		log := fakeNpx(t, opts)
		writeFile(t, UninstalledMarkerPath(opts.HomeDir), "v\n")
		res := UpdateExternalSkills(opts, &bytes.Buffer{})
		if res.Skipped == "" || len(npxLog(t, log)) != 0 {
			t.Fatalf("desinstalado no debe llamar a npx: %+v", res)
		}
	})
	t.Run("fallo", func(t *testing.T) {
		opts := sandbox(t)
		log := fakeNpx(t, opts)
		t.Setenv("NPX_FAIL", "1")
		var out bytes.Buffer
		res := UpdateExternalSkills(opts, &out)
		if len(npxLog(t, log)) != 3 || strings.Join(res.Failed, ",") != "impeccable,frontend-design,playwright-cli" {
			t.Fatalf("un fallo debe intentar todas y seguir: %+v\n%s", res, out.String())
		}
		if !strings.Contains(out.String(), "⚠") || !strings.Contains(out.String(), "npx skills add anthropics/skills") {
			t.Fatalf("falta el aviso con el comando manual:\n%s", out.String())
		}
	})
}

// T4: --if-due ejecuta como mucho una vez cada 24 h y respeta
// PI_GOMEMORY_NO_UPDATE_CHECK.
func TestHCD084IfDue(t *testing.T) {
	opts := sandbox(t)
	log := fakeNpx(t, opts)
	t.Setenv(updateCheckEnv, "")
	now := time.Now()
	if !ExternalSkillsDue(opts.PiAgentDir, now) {
		t.Fatal("sin marca debe tocar")
	}
	var out, errOut bytes.Buffer
	if code := RunSkillsUpdate(repoAssets(t), opts, []string{"--if-due"}, &out, &errOut); code != 0 {
		t.Fatalf("código %d: %s", code, errOut.String())
	}
	if len(npxLog(t, log)) == 0 {
		t.Fatal("la primera ejecución con --if-due debe llamar a npx")
	}
	if ExternalSkillsDue(opts.PiAgentDir, now.Add(time.Hour)) {
		t.Fatal("una hora después no debe tocar")
	}
	if !ExternalSkillsDue(opts.PiAgentDir, now.Add(25*time.Hour)) {
		t.Fatal("25 h después debe tocar")
	}
	_ = os.Remove(log)
	RunSkillsUpdate(repoAssets(t), opts, []string{"--if-due"}, &out, &errOut)
	if len(npxLog(t, log)) != 0 {
		t.Fatal("--if-due repitió dentro de las 24 h")
	}
	// Sin --if-due se ejecuta siempre.
	RunSkillsUpdate(repoAssets(t), opts, nil, &out, &errOut)
	if len(npxLog(t, log)) == 0 {
		t.Fatal("skills-update sin --if-due debe ejecutarse siempre")
	}
	// La comprobación pasiva respeta PI_GOMEMORY_NO_UPDATE_CHECK.
	_ = os.Remove(log)
	_ = os.Remove(filepath.Join(opts.PiAgentDir, externalSkillsStampFile))
	t.Setenv(updateCheckEnv, "1")
	RunSkillsUpdate(repoAssets(t), opts, []string{"--if-due"}, &out, &errOut)
	if len(npxLog(t, log)) != 0 {
		t.Fatal("--if-due ignoró PI_GOMEMORY_NO_UPDATE_CHECK")
	}
	if code := RunSkillsUpdate(repoAssets(t), opts, []string{"--raro"}, &out, &errOut); code != 2 {
		t.Fatalf("opción desconocida = %d, quiero 2", code)
	}
	// C-002: si falla con --if-due, no suprime el reintento durante 24 h.
	_ = os.Remove(log)
	_ = os.Remove(filepath.Join(opts.PiAgentDir, externalSkillsStampFile))
	t.Setenv(updateCheckEnv, "")
	t.Setenv("NPX_FAIL", "1")
	if code := RunSkillsUpdate(repoAssets(t), opts, []string{"--if-due"}, &out, &errOut); code != 0 {
		t.Fatalf("código con fallo = %d: %s", code, errOut.String())
	}
	if !ExternalSkillsDue(opts.PiAgentDir, now.Add(time.Minute)) {
		t.Fatal("tras un fallo con --if-due, la marca no debe suprimir el próximo reintento")
	}
}

// C-002: el presupuesto total de UpdateExternalSkills queda por debajo del
// timeout del arranque de Pi (10 min) y corta limpiamente al agotarse.
func TestHCD084TotalTimeoutBudget(t *testing.T) {
	if externalSkillsTotalTimeout >= 10*time.Minute {
		t.Fatalf("externalSkillsTotalTimeout (%v) debe ser menor que el timeout de 10 min en gomemory-status.ts", externalSkillsTotalTimeout)
	}
	opts := sandbox(t)
	log := fakeNpx(t, opts)
	prev := externalSkillsTotalTimeout
	externalSkillsTotalTimeout = -time.Millisecond
	t.Cleanup(func() { externalSkillsTotalTimeout = prev })

	var out bytes.Buffer
	res := UpdateExternalSkills(opts, &out)
	if len(npxLog(t, log)) != 0 {
		t.Fatalf("con presupuesto agotado no debe invocar npx: %v", npxLog(t, log))
	}
	if len(res.Failed) != len(ExternalSkills()) {
		t.Fatalf("debía marcar todas como fallidas por tiempo agotado: %+v", res)
	}
}

// T5: el paso de skills de install instala las externas y las enlaza en el
// repositorio central.
func TestHCD084InstallStep(t *testing.T) {
	opts := sandbox(t)
	fakeNpx(t, opts)
	var out bytes.Buffer
	res, err := installSkillsStep(repoAssets(t), opts, &out)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range ExternalSkills() {
		target, err := os.Readlink(filepath.Join(SkillRepoDir(opts.HomeDir), s.Name))
		if err != nil || target != filepath.Join(opts.HomeDir, ".agents", "skills", s.Name) {
			t.Fatalf("%s no enlazada en el repositorio central: %q %v", s.Name, target, err)
		}
	}
	if res.Linked < 2 {
		t.Fatalf("enlazadas = %d", res.Linked)
	}
	src, err := os.ReadFile("installer.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "installSkillsStep(assets, opts, os.Stdout)") {
		t.Fatal("InstallAll no usa installSkillsStep en el paso [6/9]")
	}
}

// T7: Pi revisa las skills externas al arrancar, en segundo plano.
func TestHCD084StartupHook(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "extensions", "gomemory-status.ts"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, `["skills-update", "--if-due"]`) || !strings.Contains(s, "updateExternalSkills(ctx)") {
		t.Fatal("gomemory-status.ts no ejecuta skills-update --if-due al arrancar")
	}
}

// T8: doctor informa las skills externas.
func TestHCD084Doctor(t *testing.T) {
	opts := sandbox(t)
	find := func() CheckItem {
		for _, it := range RunDoctor(opts) {
			if it.Name == "Skills externas" {
				return it
			}
		}
		t.Fatal("falta «Skills externas» en doctor")
		return CheckItem{}
	}
	if it := find(); it.Status != "WARN" || !strings.Contains(it.Details, "impeccable") || !strings.Contains(it.Details, "skills-update") {
		t.Fatalf("ausentes: %+v", it)
	}
	for _, s := range ExternalSkills() {
		hostSkill(t, opts, s.Name)
	}
	if it := find(); it.Status != "OK" {
		t.Fatalf("presentes: %+v", it)
	}
}

// T9: uninstall conserva las skills externas y dice cómo retirarlas.
func TestHCD084Uninstall(t *testing.T) {
	var out bytes.Buffer
	printToolsKept(&out)
	for _, s := range ExternalSkills() {
		if !strings.Contains(out.String(), s.RemoveCmd()) {
			t.Fatalf("falta %q:\n%s", s.RemoveCmd(), out.String())
		}
	}
}

// T10: gitlab-release viaja embebida.
func TestHCD084GitlabRelease(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "skills", "gitlab-release", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "name: gitlab-release") {
		t.Fatal("frontmatter de gitlab-release")
	}
	opts := sandbox(t)
	if _, err := InstallSkills(repoAssets(t), opts); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(SkillRepoDir(opts.HomeDir), "gitlab-release")
	if fi, err := os.Lstat(dir); err != nil || fi.Mode()&os.ModeSymlink != 0 || !isPristineEmbeddedCopy(dir) {
		t.Fatalf("gitlab-release no es copia embebida: %v", err)
	}
}
