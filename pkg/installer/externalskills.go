package installer

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// HCD-084: skills de terceros que forman parte del combo meta-harness. Se
// instalan y actualizan desde su fuente oficial con el CLI de skills.sh
// (`npx skills`), en el directorio universal ~/.agents/skills; de ahí el
// repositorio central (~/.pi-gomemory/skills) las enlaza como al resto de
// skills del anfitrión. No van en Catalog(): no publican semver.

// ExternalSkill es una skill de skills.sh: Source es el repositorio
// (<owner>/<repo>) y Name, la skill dentro de él.
type ExternalSkill struct {
	Name   string
	Source string
}

// ExternalSkills devuelve las skills externas del meta-harness, en orden.
func ExternalSkills() []ExternalSkill {
	return []ExternalSkill{
		{Name: "impeccable", Source: "pbakaus/impeccable"},
		{Name: "frontend-design", Source: "anthropics/skills"},
		{Name: "playwright-cli", Source: "microsoft/playwright-cli"},
	}
}

// Manual es la instalación a mano de la skill.
func (s ExternalSkill) Manual() string {
	return "npx skills add " + s.Source + " -g -a pi -s " + s.Name + " -y"
}

// RemoveCmd es cómo retirar la skill, que uninstall conserva.
func (s ExternalSkill) RemoveCmd() string {
	return "npx skills remove " + s.Name + " -g -y"
}

// externalSkillAddArgs son los argumentos de npx para instalar una skill. Con
// `-a pi` solo se escribe ~/.agents/skills: sin -a, skills enlaza la skill en
// los directorios de decenas de agentes.
func externalSkillAddArgs(s ExternalSkill) []string {
	return []string{"-y", "skills", "add", s.Source, "-g", "-a", "pi", "-s", s.Name, "-y"}
}

// externalSkillUpdateArgs son los argumentos de npx para actualizar skills.
func externalSkillUpdateArgs(names []string) []string {
	args := append([]string{"-y", "skills", "update"}, names...)
	return append(args, "-g", "-y")
}

// ExternalSkillNames devuelve los nombres de las skills externas, en orden.
func ExternalSkillNames() []string { return externalSkillNames() }

func externalSkillNames() []string {
	var out []string
	for _, s := range ExternalSkills() {
		out = append(out, s.Name)
	}
	return out
}

// externalSkillDir es donde `npx skills add -g` deja la skill.
func externalSkillDir(home, name string) string {
	return filepath.Join(home, ".agents", "skills", name)
}

// ExternalSkillPresent indica si la skill está instalada en ~/.agents/skills.
func ExternalSkillPresent(home, name string) bool {
	_, err := os.Stat(filepath.Join(externalSkillDir(home, name), "SKILL.md"))
	return err == nil
}

type skillLockEntry struct {
	Source          string `json:"source"`
	SkillFolderHash string `json:"skillFolderHash"`
}

// skillLockEntries lee fuente y hash de ~/.agents/.skill-lock.json, para saber
// qué gestiona skills desde la fuente oficial y qué cambió con `skills update`.
func skillLockEntries(home string) map[string]skillLockEntry {
	out := map[string]skillLockEntry{}
	data, err := os.ReadFile(filepath.Join(home, ".agents", ".skill-lock.json"))
	if err != nil {
		return out
	}
	var lock struct {
		Skills map[string]skillLockEntry `json:"skills"`
	}
	if json.Unmarshal(data, &lock) != nil || lock.Skills == nil {
		return out
	}
	return lock.Skills
}

// ExternalSkillsResult resume una pasada de UpdateExternalSkills.
type ExternalSkillsResult struct {
	Installed []string
	Updated   []string
	// Changed son las instaladas y las que `skills update` cambió.
	Changed []string
	Failed  []string
	// Skipped, si no está vacío, dice por qué no se hizo nada.
	Skipped string
}

// externalSkillsTimeout acota cada llamada a npx.
var externalSkillsTimeout = 2 * time.Minute

// externalSkillsTotalTimeout acota una pasada completa de UpdateExternalSkills
// para terminar antes del límite del arranque de Pi (10 min en gomemory-status.ts).
var externalSkillsTotalTimeout = 8 * time.Minute

// UpdateExternalSkills instala las skills externas que faltan y actualiza
// siempre las presentes. Nunca falla: avisa en out con el comando manual y
// sigue. Se omite desinstalado, en modo lite y con PI_GOMEMORY_SKIP_DEPS.
func UpdateExternalSkills(opts *Options, out io.Writer) ExternalSkillsResult {
	var res ExternalSkillsResult
	names := strings.Join(externalSkillNames(), " y ")
	switch lite, _ := GetLiteStatus(opts.HomeDir); {
	case IsUninstalled(opts.HomeDir):
		res.Skipped = "pi-gomemory está desinstalado"
	case SkipDepsRequested():
		res.Skipped = skipDepsEnv
	case lite:
		res.Skipped = "modo lite"
	}
	if res.Skipped != "" {
		fmt.Fprintf(out, "%s    · %s: no se comprueban las skills externas (%s)%s\n", ColorMuted, res.Skipped, names, ColorReset)
		return res
	}
	if _, err := exec.LookPath("npx"); err != nil {
		res.Skipped = "npx no está en el PATH"
		fmt.Fprintf(out, "%s    ⚠ npx no está en el PATH: no se instalan ni actualizan %s. Instala Node.js y ejecuta:%s\n", ColorWarning, names, ColorReset)
		for _, s := range ExternalSkills() {
			fmt.Fprintf(out, "%s      %s%s\n", ColorWarning, s.Manual(), ColorReset)
		}
		return res
	}

	before := skillLockEntries(opts.HomeDir)
	deadline := time.Now().Add(externalSkillsTotalTimeout)
	stepTimeout := func() (time.Duration, bool) {
		rem := time.Until(deadline)
		if rem <= 0 {
			return 0, false
		}
		if rem > externalSkillsTimeout {
			return externalSkillsTimeout, true
		}
		return rem, true
	}

	var present []string
	for _, s := range ExternalSkills() {
		// Solo se actualiza lo que skills gestiona desde la fuente oficial del
		// catálogo; una copia manual o de otro repositorio se adopta con add,
		// que la reemplaza por la oficial y la registra en el lockfile.
		if entry, managed := before[s.Name]; managed && strings.EqualFold(strings.TrimSpace(entry.Source), s.Source) && ExternalSkillPresent(opts.HomeDir, s.Name) {
			present = append(present, s.Name)
			continue
		}
		timeout, ok := stepTimeout()
		if !ok {
			err := fmt.Errorf("tiempo máximo agotado (%v)", externalSkillsTotalTimeout)
			res.Failed = append(res.Failed, s.Name)
			fmt.Fprintf(out, "%s    ⚠ no se pudo instalar la skill %s (%v). Instálala a mano con: %s%s\n", ColorWarning, s.Name, err, s.Manual(), ColorReset)
			continue
		}
		fmt.Fprintf(out, "%s    → instalando la skill %s desde skills.sh (%s)…%s\n", ColorMuted, s.Name, s.Source, ColorReset)
		err := runInstaller("npx", externalSkillAddArgs(s), nil, opts.HomeDir, io.Discard, timeout)
		if err == nil && !ExternalSkillPresent(opts.HomeDir, s.Name) {
			err = fmt.Errorf("npx terminó pero %s no aparece en %s", s.Name, externalSkillDir(opts.HomeDir, s.Name))
		}
		if err != nil {
			res.Failed = append(res.Failed, s.Name)
			fmt.Fprintf(out, "%s    ⚠ no se pudo instalar la skill %s (%v). Instálala a mano con: %s%s\n", ColorWarning, s.Name, err, s.Manual(), ColorReset)
			continue
		}
		res.Installed = append(res.Installed, s.Name)
		res.Changed = append(res.Changed, s.Name)
		fmt.Fprintf(out, "%s    ✓ skill %s instalada%s\n", ColorMuted, s.Name, ColorReset)
	}
	if len(present) > 0 {
		timeout, ok := stepTimeout()
		if !ok {
			err := fmt.Errorf("tiempo máximo agotado (%v)", externalSkillsTotalTimeout)
			res.Failed = append(res.Failed, present...)
			fmt.Fprintf(out, "%s    ⚠ no se pudieron actualizar las skills %s (%v). Actualízalas a mano con: npx skills update %s -g -y%s\n",
				ColorWarning, strings.Join(present, ", "), err, strings.Join(present, " "), ColorReset)
		} else {
			fmt.Fprintf(out, "%s    → actualizando las skills %s desde skills.sh…%s\n", ColorMuted, strings.Join(present, ", "), ColorReset)
			if err := runInstaller("npx", externalSkillUpdateArgs(present), nil, opts.HomeDir, io.Discard, timeout); err != nil {
				res.Failed = append(res.Failed, present...)
				fmt.Fprintf(out, "%s    ⚠ no se pudieron actualizar las skills %s (%v). Actualízalas a mano con: npx skills update %s -g -y%s\n",
					ColorWarning, strings.Join(present, ", "), err, strings.Join(present, " "), ColorReset)
			} else {
				res.Updated = present
				after := skillLockEntries(opts.HomeDir)
				for _, n := range present {
					if after[n].SkillFolderHash != before[n].SkillFolderHash {
						res.Changed = append(res.Changed, n)
					}
				}
				fmt.Fprintf(out, "%s    ✓ skills %s al día%s\n", ColorMuted, strings.Join(present, ", "), ColorReset)
			}
		}
	}
	return res
}

// externalSkillsItem informa en doctor si las skills externas están en
// ~/.agents/skills. Solo lee.
func externalSkillsItem(home string) CheckItem {
	const name = "Skills externas"
	var missing []string
	for _, s := range ExternalSkills() {
		if !ExternalSkillPresent(home, s.Name) {
			missing = append(missing, s.Name)
		}
	}
	if len(missing) > 0 {
		return CheckItem{Name: name, Status: "WARN", Details: fmt.Sprintf("faltan %s en ~/.agents/skills; ejecuta pi-gomemory-setup skills-update (requiere npx)", strings.Join(missing, ", "))}
	}
	return CheckItem{Name: name, Status: "OK", Details: strings.Join(externalSkillNames(), ", ") + " desde skills.sh en ~/.agents/skills"}
}

// externalSkillsStampFile guarda cuándo se revisaron por última vez.
const externalSkillsStampFile = "pi-gomemory-skills-update.json"

// externalSkillsTTL espacia la revisión del arranque de Pi.
const externalSkillsTTL = 24 * time.Hour

type externalSkillsStamp struct {
	CheckedAt time.Time `json:"checked_at"`
}

// ExternalSkillsDue indica si toca revisar las skills externas: sin marca,
// con una marca de hace 24 h o más, o con una marca del futuro.
func ExternalSkillsDue(piAgentDir string, now time.Time) bool {
	data, err := os.ReadFile(filepath.Join(piAgentDir, externalSkillsStampFile))
	if err != nil {
		return true
	}
	var st externalSkillsStamp
	if json.Unmarshal(data, &st) != nil || st.CheckedAt.IsZero() {
		return true
	}
	return now.Sub(st.CheckedAt) >= externalSkillsTTL || now.Before(st.CheckedAt)
}

func writeExternalSkillsStamp(piAgentDir string, now time.Time) {
	if data, err := json.Marshal(externalSkillsStamp{CheckedAt: now}); err == nil {
		_ = os.MkdirAll(piAgentDir, 0o755)
		_ = writeFileAtomic(filepath.Join(piAgentDir, externalSkillsStampFile), data, 0o644)
	}
}

// installSkillsStep es el paso [6/9] de install: primero las skills externas
// (que quedan en ~/.agents/skills) y después el repositorio central, que las
// enlaza junto a las embebidas.
func installSkillsStep(assets fs.FS, opts *Options, out io.Writer) (SkillRepoResult, error) {
	UpdateExternalSkills(opts, out)
	writeExternalSkillsStamp(opts.PiAgentDir, time.Now())
	return InstallSkills(assets, opts)
}

// RunSkillsUpdate implementa `pi-gomemory-setup skills-update [--if-due]`:
// instala o actualiza las skills externas y vuelve a reunir el repositorio
// central. Con --if-due (arranque de Pi) solo actúa una vez cada 24 h y
// respeta PI_GOMEMORY_NO_UPDATE_CHECK. Si algo cambió lo dice en stdout; el
// progreso va a stderr.
func RunSkillsUpdate(assets fs.FS, opts *Options, args []string, stdout, stderr io.Writer) int {
	ifDue := false
	for _, a := range args {
		if a != "--if-due" {
			fmt.Fprintf(stderr, "opción desconocida para skills-update: %s\nUso: pi-gomemory-setup skills-update [--if-due]\n", a)
			return 2
		}
		ifDue = true
	}
	now := time.Now()
	if ifDue {
		if UpdateCheckDisabled() || !ExternalSkillsDue(opts.PiAgentDir, now) {
			return 0
		}
		// Se marca antes de salir a la red: dos arranques seguidos no repiten.
		writeExternalSkillsStamp(opts.PiAgentDir, now)
	}
	res := UpdateExternalSkills(opts, stderr)
	if ifDue && len(res.Failed) > 0 {
		// Si falló o agotó el tiempo, borra la marca para reintentar en el próximo arranque.
		_ = os.Remove(filepath.Join(opts.PiAgentDir, externalSkillsStampFile))
	} else if !ifDue {
		writeExternalSkillsStamp(opts.PiAgentDir, now)
	}
	if res.Skipped != "" {
		return 0
	}
	if _, err := InstallSkills(assets, opts); err != nil {
		fmt.Fprintf(stderr, "Error sincronizando skills: %v\n", err)
		return 1
	}
	if len(res.Changed) > 0 {
		fmt.Fprintf(stdout, "Skills externas actualizadas: %s. Usa /reload en Pi para cargarlas.\n", strings.Join(res.Changed, ", "))
	}
	return 0
}
