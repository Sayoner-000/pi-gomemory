# Target ACR/ADM — ronda 4 (verificación post-cierre R3-01..R3-04 y HCD-042)

## Congelado y alcance

HEAD `5183867a86a800a88638b4234051810e47d4beb0` + árbol sucio actual. Digest SHA256 inventario `9cf38a850777de21e7316189943e42941c3d2ecd7fe8bb0a044bfb50269ab522`.

Inventario del coordinador: `git ls-files -z --cached --others --exclude-standard -- . ':!:docs/audits/**' ':!:hcd/**' ':!:.codegraph/**' ':!:.codebase-memory/**' | LC_ALL=C sort -zu | xargs -0 shasum -a 256 | shasum -a 256`.

Alcance explícito: `main.go`, `pkg/installer` y `scripts/build-release.sh`. Prioridad en los cierres recién aplicados (`manifest.go`, `deps.go`, `uninstall.go`, `doctor.go`, `acradm_round3_test.go`) y en los módulos críticos (`pins.go`, `pins_candidate.go`, `jsonfile.go`, `plugin.go`, pruebas HCD-011/030/034/035/039/040/041/042). No es una auditoría exhaustiva de extensiones TS ni UI HTML/CSS.

## Encargo independiente

Revisa fuentes actuales desde primeros principios, no solo este manifiesto. Solo lectura: sin modificar producto ni memoria, sin instalar dependencias, sin ejecutar install/uninstall contra HOME real. No recibes resultados de otro revisor ni historial de conversación. No infieras consenso ni inventes defectos. Cada defecto requiere conducta incorrecta alcanzable, ubicación, evidencia y severidad. Una colección vacía es válida.

Devuelve JSON con `findings` [{local_id, location, severity: CRITICAL/HIGH/MEDIUM/LOW/INFO, category, claim, evidence_class, evidence, confidence}], `inspected`, `limitations`. Explicita si no puedes ejecutar comandos ni verificar digest.

## Contratos aplicables

- `deps-update` solo anuncia al día versiones efectivas comparables. `dev`, prereleases (`1.0.0-rc.1`) y formatos no admitidos por `parseSemver` son desconocidos: salida 1, aviso y petición `deps-refresh`. Se actualiza lo conocido aunque haya desconocidos. Los lectores de comandos (`versionFromCmd`, `latestFromCmd`) deben conservar el token completo sin truncar sufijos.
- Resolver pin compilado y versión local antes de clasificar comparabilidad. Precedencia del pin sobre latest antigua en caché. Fuentes de actualización solo de probes compilados, no de caché. Sin red implícita en el diagnóstico local. Se conserva el fallback histórico a Current de caché cuando el lector local falla.
- Cachés, manifiesto y `--self`: conservar entradas no regulares preexistentes y sobrevenidas (FIFO, enlaces, directorios) y archivos sustituidos después del plan. `ReadManifest` no debe bloquear ante FIFO ni seguir enlaces al abrir. Retener original mediante descriptor hasta aplicar o descartar para que su inode no se reutilice durante ese intervalo.
- Plan dueño de recursos, `Close` explícito/idempotente, acciones con recursos de un solo uso. Cierre de retenidos incluso si acción falla/cambia/desaparece; al descartar y al salir por simulación, cancelación, falta de terminal o error. Manifiesto al final, condicionado a ausencia de fallos previos (`ifClean`).
- Pins `pi-antigravity` `0.14.0` y `pi-cc-extensions` `1.0.20`; retirada solo actuales/históricos conocidos. Preservar configuración ajena. Fallo/cancelación de proceso nunca da versión. `RunDoctor` sin red y con `WaitDelay` acotado ante descendientes con tuberías heredadas.
- Extracción limitada a 10.000 entradas, 256 MiB de flujo y 200 MiB de contenido; cancelación entre entradas, limpieza tras fallo, rechazo de escape/enlaces.
- Gate de release rechaza consulta fallida o nuevas estables no rechazadas, salvo escape explícito `PI_GOMEMORY_ALLOW_STALE_PINS=1`.
