# ACR y ADM — ronda 4 (verificación post-cierre R3-01..R3-04 y HCD-042)

- **Fecha:** 2026-11-17.
- **Revisión:** HEAD `5183867a86a800a88638b4234051810e47d4beb0` + árbol sucio actual.
- **Digest de producto:** `9cf38a850777de21e7316189943e42941c3d2ecd7fe8bb0a044bfb50269ab522`, comprobado antes y después de la revisión. Excluye `docs/audits/**`, `hcd/**`, `.codegraph/**` y `.codebase-memory/**`.
- **Alcance dirigido:** `main.go`, `pkg/installer` y `scripts/build-release.sh`, con prioridad en los cierres `R3-01..R3-04` (`manifest.go`, `deps.go`, `uninstall.go`, `doctor.go`, `acradm_round3_test.go`) y módulos críticos (`pins.go`, `pins_candidate.go`, `jsonfile.go`, `plugin.go`, `update.go`, `installer.go`). No es auditoría exhaustiva de extensiones TypeScript ni UI HTML/CSS.
- **Modalidad:** solo lectura. Sin modificaciones de código, cambios de pins, instalaciones en HOME real, commits ni release.
- [Target y contratos](2026-11-17-acr-adm-round4-target.md).

## 1. Resumen de salud del código

- **Lenguajes / tecnologías detectadas:** Go en CLI/instalador (`main.go`, `pkg/installer/*.go`) y pruebas (`pkg/installer/*_test.go`); Shell en gates y scripts (`install.sh`, `scripts/build-release.sh`). Extensiones TypeScript y documentación HTML/CSS fuera de inspección profunda.
- **Nivel de riesgo general:** **Bajo** dentro del alcance auditado.
- **Evaluación breve:** Los cuatro hallazgos `MEDIUM` de la ronda 3 (`R3-01` a `R3-04`) quedaron corregidos con el mínimo cambio necesario y con pruebas permanentes de regresión (`TestRound3*`) que cubren FIFO en `ReadManifest`, sufijos prerelease en `versionFromCmd`/`latestFromCmd`, revalidación y retención de descriptor en `--self`, y `WaitDelay` ante descendientes en `RunDoctor`. Las protecciones previas (`HCD-035`, `HCD-040`, `HCD-041`, `HCD-042`) se mantienen intactas bajo la suite completa con detector de carreras y análisis estático.
- **Resultado:** **0 hallazgos** (0 críticos, 0 altos, 0 medios, 0 bajos) en la inspección estática dual (ACR) y comprobaciones dinámicas (ADM). Límites declarados: análisis estático limitado a las herramientas Go/Shell disponibles en `PATH`; sin ejecución de `govulncheck`, `staticcheck`, `shellcheck`, `semgrep`, `gitleaks` ni `tsc`, ni pruebas funcionales sobre extensiones TS o runtime Linux.

### Cobertura y validación

| Comprobación | Resultado |
|---|---|
| `go test -race ./... -count=1` | Pasa (`pi-gomemory` 1,205 s–1,227 s; `pkg/installer` 73,388 s–74,630 s; incluye `TestRound3*`, `TestFollowup*` y `TestHCD042*`) |
| `go vet ./... && go vet -tags pins ./...` | Pasa (código 0) |
| `bash -n install.sh scripts/build-release.sh` | Pasa (solo sintaxis) |
| `git diff --check` | Pasa (sin errores de formato/espacios) |
| `go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsCheck$'` | Falla según contrato del gate: `pi-antigravity` al día (`0.14.0`), `pi-cc-extensions` fijado `1.0.20` mientras npm publica `1.0.21` |
| Dos revisores `splash` independientes (A y B) | Completados sin hallazgos (`findings: []`); lectura estática independiente del target congelado |
| `govulncheck`, `staticcheck`, `shellcheck`, `semgrep`, `gitleaks`, `tsc` | No disponibles en `PATH`; no instalados |
| Extensiones TS, UI HTML/CSS y runtime Linux | Fuera del alcance dirigido de esta ronda |

## 2. Hallazgos y fugas detectadas

| ID | Severidad | Estado ADM | Consenso ACR | Problema |
|---|---|---|---|---|
| — | — | — | — | Sin hallazgos nuevos ni regresiones detectadas en el alcance |

**Verificación de cierres anteriores (`R3-01` a `R3-04`):**

- `R3-01` (`pkg/installer/manifest.go:67-95`): `ReadManifest` abre con `os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK`, valida `st.Mode().IsRegular()` y propaga errores de `Close`. Cubierto por `TestRound3ManifestFIFO`.
- `R3-02` (`pkg/installer/deps.go:77,235,241`): `semverRe` y `latestFromCmd` de `mem`/`codegraph` conservan sufijos `[0-9A-Za-z.+-]*` para que `parseSemver` rechace prereleases antes de anunciar «al día». Cubierto por `TestRound3PrereleaseThroughCommand`.
- `R3-03` (`pkg/installer/uninstall.go:965`): `planSelf` reutiliza `planRegularRemoval`, reteniendo el descriptor original y preservando sustitutos y entradas no regulares. Cubierto por `TestRound3SelfReplacement`.
- `R3-04` (`pkg/installer/doctor.go:91`): `RunDoctor` configura `mcpListCmd.WaitDelay = time.Second` antes de `Output()`. Cubierto por `TestRound3DoctorDescendant`.

## 3. ACR: consenso y veredicto

- **Review ID:** `acr_5c35efd6-2c0d-40be-b825-8f59e901deb6`
- **Revisores:** dos ejecuciones independientes de `splash` (A y B) sobre [2026-11-17-acr-adm-round4-target.md](2026-11-17-acr-adm-round4-target.md) sin exposición cruzada de hallazgos ni historial. Nivel de independencia declarado por el motor: `degraded / same-model`.
- **Consenso:** `findings_total: 0`, `findings_confirmed: 0`, `findings_suspect: 0`, `contradictions: 0`, `fix_rounds: 0`.
- **Veredicto terminal del motor:** **`APPROVED`**. (Nota: la aprobación ACR no autoriza commit, push, tag ni release; además, `TestPinsCheck` sigue bloqueando release por `pi-cc-extensions@1.0.21` en npm frente al pin `1.0.20`).

## 4. Recomendaciones de resiliencia

1. Tratar por separado el candidato `pi-cc-extensions@1.0.21` (y la observación humana pendiente de `HCD-040`) antes de intentar una release; no usar `PI_GOMEMORY_ALLOW_STALE_PINS=1` sin validación explícita.
2. Incorporar `govulncheck`, `staticcheck`, `shellcheck`, `gitleaks` y `tsc` en CI para cubrir las comprobaciones no disponibles localmente en `PATH`.
