# Corrección de hallazgos de auditoría ACR/ADM followup

ID: HCD-016
Tipo: fix
Carril: fast
Motivo: Cerrar hallazgos de integridad, robustez y guardarraíl identificados en el informe docs/audits/2026-11-11-acr-adm-followup.md

## Intención
Corregir los hallazgos de integridad, robustez y guardarraíl auditados en docs/audits/2026-11-11-acr-adm-followup.md en el orden de prioridad establecido:
1. H-01 (hostskills.go: pruneLegacyPiSkills no borra con RemoveAll si la carpeta de skill tiene archivos adicionales además de SKILL.md y la marca .pi-gomemory).
2. H-02 (uninstall.go: Uninstall y dry-run devuelven error y muestran el plan si hay acciones ActionFail aunque Pending() == 0).
3. H-03 (doctor.go: pi mcp list con CommandContext y timeout de 10s; si falla o el JSON es inválido emite fila WARN en lugar de omitir silenciosamente).
4. H-04 (gomemory-plan-patterns.json / gomemory-plan.ts: cerrar evasión en modo plan bloqueando bash/sh/zsh -c, python/node/perl/ruby -c|-e, xargs, env, make, find -delete|-exec, dd of=, git apply|clean|pull, patch, curl -o, wget -O, install, rsync, unlink, go build -o).
5. H-05 y H-06 (claude_usage.go: comparar fechas en zona local parseando RFC3339 e iterar cache.Files con claves ordenadas deterministas).
6. H-08 (gomemory-sidebar.ts: mostrar solo el campo error del JSON en stderr, guarda inflight de ejecución, TTL/backoff tras error, validar forma de JSON antes de render).
7. H-09 (hostskills.go SyncSkillRepo borra solo enlaces cuyo destino esté bajo hostSkillSources; uninstall.go planPackages usa CommandContext con timeout y propaga error de pi remove).
8. H-11 (harness.go: TTL de 24h en deps-refresh y límite seguro o truncado al contexto de memoria).
9. H-07 (análisis/documentación de viabilidad de Splash sin proveedor autenticado).

## Reproducción
1. H-01: Carpeta en ~/.pi/agent/skills/foo con SKILL.md idéntico y script adicional scripts/mine.sh es borrada por pruneLegacyPiSkills.
2. H-02: settings.json corrupto sin otros elementos pendientes retorna nil en Uninstall.
3. H-03: pi mcp list con JSON inválido o colgado no emite WARN.
4. H-04: bashBlocked permite bash -c, python3 -c, make, etc.
5. H-05/H-06: claude_usage mezcla UTC/local y orden de proyectos varía con filas duplicadas.

## Alcance
Incluido: corrección de H-01 (hostskills.go con onlySkillFile), H-02 (uninstall.go ActionFail check en dry-run y ejecución), H-03 (doctor.go con CommandContext 10s y fila WARN), H-04 (gomemory-plan-patterns.json / gomemory-plan.ts bloqueando intérpretes y evasiones), H-05 y H-06 (claude_usage.go fechas locales y ordenación determinista), H-08 (gomemory-sidebar.ts stderr limpio, inflight y defensivo), H-09 (hostskills.go destino de symlink y uninstall.go timeout/error), H-11 (harness.go TTL 24h deps-refresh y límite de contexto), y tests unitarios.
Excluido: H-10 (decisiones de proceso/CI/firma) y H-12 (información/higiene sin cambio funcional), commits o releases.

## ¿Qué pasa si no lo hacemos?
El usuario sufre riesgo de pérdida de datos propios en ~/.pi/agent/skills (H-01), reportes de desinstalación falsamente exitosos (H-02), diagnósticos incompletos o colgados (H-03) y evasión trivial del modo plan durante sesiones de planificación (H-04).

## Alternativa más barata
Ignorar los hallazgos o corregirlos de forma no atómica sin verificación reproducible ni pruebas de regresión.

## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->

## Checklist
- [x] 1. H-01: pruneLegacyPiSkills no borra carpetas con contenido adicional → go test -v ./pkg/installer -run TestH01PruneLegacyPreservesUserData
- [x] 2. H-02: Uninstall y dry-run fallan con ActionFail aunque Pending sea 0 → go test -v ./pkg/installer -run TestH02UninstallFailsWhenActionFail
- [x] 3. H-03: doctor usa timeout y emite WARN ante fallo de pi mcp list → go test -v ./pkg/installer -run TestH03DoctorTimeoutAndWarn
- [x] 4. H-04: gomemory-plan bloquea intérpretes y comandos de evasión en modo plan → go test -v ./pkg/installer -run TestH04PlanEvasionsBlocked
- [x] 5. H-05 y H-06: claude_usage compara fechas locales y tiene atribución determinista → go test -v ./pkg/installer -run TestH05H06ClaudeUsageDeterministicAndLocal
- [x] 6. H-08: gomemory-sidebar maneja errores limpios, guarda inflight y validación defensiva → go test -v ./pkg/installer -run TestH08SidebarErrorAndInflight
- [x] 7. H-09: SyncSkillRepo solo borra symlinks de fuentes del host y planPackages propaga errores → go test -v ./pkg/installer -run TestH09SyncSkillRepoAndUninstallPackages
- [x] 8. H-11: harness launcher respeta TTL de 24h para deps-refresh y no desborda argumentos → go test -v ./pkg/installer -run TestH11HarnessTTLAndPromptCap

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 11:49 -05:00 · tarea 1 · rojo observado · caducada · `go test -v ./pkg/installer -run TestH01PruneLegacyPreservesUserData` (salida 1)
      === RUN   TestH01PruneLegacyPreservesUserData
          hcd016_test.go:37: PERDIDA DE DATOS: userScript fue borrado por pruneLegacyPiSkills
      --- FAIL: TestH01PruneLegacyPreservesUserData (0.00s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.006s
      FAIL
- 2026-11-11 11:49 -05:00 · tarea 1 · verde · caducada · `go test -v ./pkg/installer -run TestH01PruneLegacyPreservesUserData` (salida 0)
      === RUN   TestH01PruneLegacyPreservesUserData
      --- PASS: TestH01PruneLegacyPreservesUserData (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.005s
- 2026-11-11 11:50 -05:00 · tarea 2 · rojo observado · caducada · `go test -v ./pkg/installer -run TestH02UninstallFailsWhenActionFail` (salida 1)
      === RUN   TestH02UninstallFailsWhenActionFail
          hcd016_test.go:57: Uninstall con dry-run y ActionFail debio retornar error, obtuvo nil
      --- FAIL: TestH02UninstallFailsWhenActionFail (0.00s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.004s
      FAIL
- 2026-11-11 11:50 -05:00 · tarea 2 · verde · caducada · `go test -v ./pkg/installer -run TestH02UninstallFailsWhenActionFail` (salida 0)
      === RUN   TestH02UninstallFailsWhenActionFail
      --- PASS: TestH02UninstallFailsWhenActionFail (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 11:51 -05:00 · tarea 1 · verde · caducada · `go test -v ./pkg/installer -run TestH01PruneLegacyPreservesUserData` (salida 0)
      === RUN   TestH01PruneLegacyPreservesUserData
      --- PASS: TestH01PruneLegacyPreservesUserData (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.004s
- 2026-11-11 11:51 -05:00 · tarea 3 · rojo observado · caducada · `go test -v ./pkg/installer -run TestH03DoctorTimeoutAndWarn` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd016_test.go:85:28: report.Items undefined (type []CheckItem has no field or method Items)
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-11 11:52 -05:00 · tarea 3 · verde · caducada · `go test -v ./pkg/installer -run TestH03DoctorTimeoutAndWarn` (salida 0)
      === RUN   TestH03DoctorTimeoutAndWarn
      --- PASS: TestH03DoctorTimeoutAndWarn (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 11:52 -05:00 · tarea 1 · verde · caducada · `go test -v ./pkg/installer -run TestH01PruneLegacyPreservesUserData` (salida 0)
      === RUN   TestH01PruneLegacyPreservesUserData
      --- PASS: TestH01PruneLegacyPreservesUserData (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.005s
- 2026-11-11 11:52 -05:00 · tarea 2 · verde · caducada · `go test -v ./pkg/installer -run TestH02UninstallFailsWhenActionFail` (salida 0)
      === RUN   TestH02UninstallFailsWhenActionFail
      --- PASS: TestH02UninstallFailsWhenActionFail (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.006s
- 2026-11-11 11:52 -05:00 · tarea 4 · rojo observado · caducada · `go test -v ./pkg/installer -run TestH04PlanEvasionsBlocked` (salida 1)
      === RUN   TestH04PlanEvasionsBlocked
          hcd016_test.go:139: FAIL bash -c 'rm -rf build' | sh -c "touch f" | zsh -c 'rm f' | python3 -c "open('f','w')" | python -c 'import os; os.remove(1)' | node -e "require('fs').writeFileSync('f','')" | perl -pi -e 's/a/b/' file | ruby -e 'File.write("f", "")' | xargs rm | env rm f | make build | find . -delete | find . -exec rm {} \; | dd of=f if=/dev/zero | git apply patch.diff | git clean -fd | git pull origin main | patch -p1 < file.patch | curl -o output.bin http://example.com | wget -O output.bin http://example.com | install -m 755 src dst | rsync -av src/ dst/ | unlink file.txt | go build -o /tmp/bin .
      --- FAIL: TestH04PlanEvasionsBlocked (0.06s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.065s
      FAIL
- 2026-11-11 11:52 -05:00 · tarea 4 · verde · caducada · `go test -v ./pkg/installer -run TestH04PlanEvasionsBlocked` (salida 0)
      === RUN   TestH04PlanEvasionsBlocked
      --- PASS: TestH04PlanEvasionsBlocked (0.05s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 11:53 -05:00 · tarea 5 · rojo observado · caducada · `go test -v ./pkg/installer -run TestH05H06ClaudeUsageDeterministicAndLocal` (salida 1)
      === RUN   TestH05H06ClaudeUsageDeterministicAndLocal
          hcd016_test.go:189: iteracion 0: no determinista, se esperaba proj-a pero obtuvo: [{Name:proj-z ClaudeTokenBucket:{Input:100 Output:20 CacheCreate:0 CacheRead:0 Messages:1}} {Name:proj-a ClaudeTokenBucket:{Input:0 Output:0 CacheCreate:0 CacheRead:0 Messages:0}}]
      --- FAIL: TestH05H06ClaudeUsageDeterministicAndLocal (0.00s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.004s
      FAIL
- 2026-11-11 11:53 -05:00 · tarea 5 · verde · caducada · `go test -v ./pkg/installer -run TestH05H06ClaudeUsageDeterministicAndLocal` (salida 0)
      === RUN   TestH05H06ClaudeUsageDeterministicAndLocal
      --- PASS: TestH05H06ClaudeUsageDeterministicAndLocal (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 11:54 -05:00 · tarea 6 · verde · caducada · `go test -v ./pkg/installer -run TestH08SidebarErrorAndInflight` (salida 0)
      === RUN   TestH08SidebarErrorAndInflight
      --- PASS: TestH08SidebarErrorAndInflight (0.07s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 11:55 -05:00 · tarea 7 · verde · caducada · `go test -v ./pkg/installer -run TestH09SyncSkillRepoAndUninstallPackages` (salida 0)
      === RUN   TestH09SyncSkillRepoAndUninstallPackages
      --- PASS: TestH09SyncSkillRepoAndUninstallPackages (0.11s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 11:56 -05:00 · tarea 8 · verde · vigente · `go test -v ./pkg/installer -run TestH11HarnessTTLAndPromptCap` (salida 0)
      === RUN   TestH11HarnessTTLAndPromptCap
      --- PASS: TestH11HarnessTTLAndPromptCap (0.01s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 11:56 -05:00 · tarea 1 · verde · vigente · `go test -v ./pkg/installer -run TestH01PruneLegacyPreservesUserData` (salida 0)
      === RUN   TestH01PruneLegacyPreservesUserData
      --- PASS: TestH01PruneLegacyPreservesUserData (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.005s
- 2026-11-11 11:56 -05:00 · tarea 2 · verde · vigente · `go test -v ./pkg/installer -run TestH02UninstallFailsWhenActionFail` (salida 0)
      === RUN   TestH02UninstallFailsWhenActionFail
      --- PASS: TestH02UninstallFailsWhenActionFail (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.007s
- 2026-11-11 11:56 -05:00 · tarea 3 · verde · vigente · `go test -v ./pkg/installer -run TestH03DoctorTimeoutAndWarn` (salida 0)
      === RUN   TestH03DoctorTimeoutAndWarn
      --- PASS: TestH03DoctorTimeoutAndWarn (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.006s
- 2026-11-11 11:56 -05:00 · tarea 4 · verde · vigente · `go test -v ./pkg/installer -run TestH04PlanEvasionsBlocked` (salida 0)
      === RUN   TestH04PlanEvasionsBlocked
      --- PASS: TestH04PlanEvasionsBlocked (0.06s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.066s
- 2026-11-11 11:56 -05:00 · tarea 5 · verde · vigente · `go test -v ./pkg/installer -run TestH05H06ClaudeUsageDeterministicAndLocal` (salida 0)
      === RUN   TestH05H06ClaudeUsageDeterministicAndLocal
      --- PASS: TestH05H06ClaudeUsageDeterministicAndLocal (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.004s
- 2026-11-11 11:56 -05:00 · tarea 6 · verde · vigente · `go test -v ./pkg/installer -run TestH08SidebarErrorAndInflight` (salida 0)
      === RUN   TestH08SidebarErrorAndInflight
      --- PASS: TestH08SidebarErrorAndInflight (0.07s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.071s
- 2026-11-11 11:56 -05:00 · tarea 7 · verde · vigente · `go test -v ./pkg/installer -run TestH09SyncSkillRepoAndUninstallPackages` (salida 0)
      === RUN   TestH09SyncSkillRepoAndUninstallPackages
      --- PASS: TestH09SyncSkillRepoAndUninstallPackages (0.12s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.126s

## Siguiente
pendiente
