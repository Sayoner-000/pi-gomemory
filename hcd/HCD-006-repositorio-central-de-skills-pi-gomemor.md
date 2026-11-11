# Repositorio central de skills ~/.pi-gomemory/skills

ID: HCD-006
Tipo: evolution
Carril: fast
Motivo: decisiones resueltas por la persona: ubicación, enlaces, prioridad y frecuencia

## Intención
Escanear las skills de los agentes del anfitrión y reunirlas, junto a las embebidas, en ~/.pi-gomemory/skills, que Pi carga como única fuente

## Reproducción
Hoy (HCD-002/003) Pi registra `~/.config/opencode/skills` entera, enlaza en `~/.pi/agent/skills` solo las de Claude que faltan e ignora Codex. Las 21 skills de gomemory que Pi usa son las de OpenCode, no las adaptadas para Pi. En este sistema hay skills en `~/.agents/skills` (23), `~/.config/opencode/skills` (21), `~/.claude/skills` (44) y `~/.codex/skills` (24).

## Alcance
Incluido: `~/.pi-gomemory/skills` con la estructura de `~/.agents/skills`. Las skills embebidas se copian con su marca y siempre ganan. Las del anfitrión se enlazan por orden fijo: `~/.agents/skills` → `~/.config/opencode/skills` → `~/.claude/skills` → `~/.codex/skills` → `~/.gemini/skills`. Solo cuentan las subcarpetas directas que tienen `SKILL.md`. Los enlaces rotos o que apuntan a otro sitio se rehacen. Nunca se tocan las carpetas reales de la persona dentro del repositorio ni las del anfitrión. `settings.json.skills` registra `~/.pi-gomemory/skills` y retira las rutas del anfitrión que añadió HCD-002; las demás se conservan. De `~/.pi/agent/skills` salen las copias embebidas intactas y los enlaces a carpetas del anfitrión; las skills propias se quedan. Nuevo comando `pi-gomemory-setup skills-sync`, que se ejecuta al instalar y en cada arranque del launcher. `doctor` muestra el total y las colisiones resueltas.
Excluido: copiar skills del anfitrión; modificar las carpetas de otros agentes; que gomemory escriba en `~/.agents/skills`.
Decisión previa que cambia: HCD-002 registraba las carpetas del anfitrión en `settings.json`, y HCD-003 retiraba de Pi las copias embebidas cuando el anfitrión ya ofrecía la misma skill. A partir de ahora gana la embebida (decisión de la persona) y las carpetas del anfitrión solo entran mediante el repositorio.

## ¿Qué pasa si no lo hacemos?
Pi sigue cargando la versión de OpenCode de las skills de gomemory, no ve las de Codex y cada agente nuevo exige tocar el código.

## Alternativa más barata
Exclusiones `-ruta/<n>` en settings.json (memoria 39): resuelve la prioridad pero no reúne las skills ni añade Codex/Gemini.

## Aclaraciones
- [x] Q1. ¿Ubicación? → ~/.pi-gomemory/skills
- [x] Q2. ¿Copiar o enlazar? → enlace simbólico
- [x] Q3. ¿Colisiones? → embebida de pi-gomemory, luego orden fijo de agentes
- [x] Q4. ¿Cuándo? → al instalar y al iniciar pi-gomemory

## Checklist
- [x] 1. SyncSkillRepo reúne embebidas y enlaces del anfitrión con la prioridad acordada y limpia enlaces rotos → `go test ./pkg/installer -run TestSkillRepoScan`
- [x] 2. El repositorio no pisa contenido propio de la persona → `go test ./pkg/installer -run TestSkillRepoKeepsUserContent`
- [x] 3. RegisterSkillRepo deja settings.json y ~/.pi/agent/skills migrados sin perder lo propio → `go test ./pkg/installer -run TestSkillRepoPiSettings`
- [x] 4. El launcher sincroniza antes de lanzar pi → `go test ./pkg/installer -run TestHarnessSyncsSkills`
- [x] 5. Suite completa → `go test ./... && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 04:59 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestSkillRepoScan` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd006_test.go:22:10: undefined: SkillRepoDir
      pkg/installer/hcd006_test.go:24:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:56:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:69:10: undefined: SkillRepoDir
      pkg/installer/hcd006_test.go:72:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:93:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:96:12: undefined: RegisterSkillRepo
      pkg/installer/hcd006_test.go:100:46: undefined: SkillRepoSetting
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-11 04:59 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestSkillRepoKeepsUserContent` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd006_test.go:22:10: undefined: SkillRepoDir
      pkg/installer/hcd006_test.go:24:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:56:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:69:10: undefined: SkillRepoDir
      pkg/installer/hcd006_test.go:72:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:93:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:96:12: undefined: RegisterSkillRepo
      pkg/installer/hcd006_test.go:100:46: undefined: SkillRepoSetting
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-11 04:59 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestSkillRepoPiSettings` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd006_test.go:22:10: undefined: SkillRepoDir
      pkg/installer/hcd006_test.go:24:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:56:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:69:10: undefined: SkillRepoDir
      pkg/installer/hcd006_test.go:72:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:93:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:96:12: undefined: RegisterSkillRepo
      pkg/installer/hcd006_test.go:100:46: undefined: SkillRepoSetting
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-11 04:59 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run TestHarnessSyncsSkills` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd006_test.go:22:10: undefined: SkillRepoDir
      pkg/installer/hcd006_test.go:24:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:56:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:69:10: undefined: SkillRepoDir
      pkg/installer/hcd006_test.go:72:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:93:15: undefined: SyncSkillRepo
      pkg/installer/hcd006_test.go:96:12: undefined: RegisterSkillRepo
      pkg/installer/hcd006_test.go:100:46: undefined: SkillRepoSetting
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-11 05:04 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestSkillRepoScan` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.182s
- 2026-11-11 05:04 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestSkillRepoKeepsUserContent` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.158s
- 2026-11-11 05:04 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestSkillRepoPiSettings` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.179s
- 2026-11-11 05:04 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run TestHarnessSyncsSkills` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.165s
- 2026-11-11 05:04 -05:00 · tarea 5 · verde · vigente · `go test ./... && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	(cached)

## Siguiente
Cerrada con aprobación sobre la revisión cdf72adf9529.
