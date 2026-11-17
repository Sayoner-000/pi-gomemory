# Target ACR/ADM — ronda adicional posterior a Q2

## Congelado y alcance

HEAD `5183867a86a800a88638b4234051810e47d4beb0` + árbol sucio actual. Digest SHA256 inventario `0ed83c20464858d16b97d14c536ea4da84ef74f07c23b50c82c6f4a58332343b`.

Inventario del coordinador: `git ls-files -z --cached --others --exclude-standard -- . ':!:docs/audits/**' ':!:hcd/**' ':!:.codegraph/**' ':!:.codebase-memory/**' | LC_ALL=C sort -zu | xargs -0 shasum -a 256 | shasum -a 256`.

Alcance explícito: `main.go`, `pkg/installer` y `scripts/build-release.sh`. Prioridad en `deps.go`, `uninstall.go`, `pins.go`, `pins_candidate.go`, `jsonfile.go`, `plugin.go`, `doctor.go`, pruebas HCD-011/030/034/035/039/040/041/042 y de ciclo de vida. No es un diff Git mínimo ni auditoría exhaustiva del repositorio. No revisar extensiones o UI completas.

## Encargo independiente

Revisa fuentes actuales desde primeros principios, no solo este manifiesto. Solo lectura: sin modificar producto ni memoria, sin instalar dependencias, sin ejecutar install/uninstall contra HOME real. No recibes resultados de otro revisor ni historial de conversación. No infieras consenso ni inventes defectos. Cada defecto requiere conducta incorrecta alcanzable, ubicación, evidencia y severidad. Una colección vacía es válida.

Devuelve JSON con `findings` [{local_id, location, severity: CRITICAL/HIGH/MEDIUM/LOW/INFO, category, claim, evidence_class, evidence, confidence}], `inspected`, `limitations`. Explicita si no puedes ejecutar comandos ni verificar digest.

## Contratos aplicables

- `deps-update` solo anuncia al día versiones efectivas comparables. `dev` y formatos no admitidos por el comparador existente son desconocidos: salida 1, aviso y petición deps-refresh. Se actualiza lo conocido aunque haya desconocidos.
- Resolver pin compilado y versión local antes de clasificar comparabilidad. Precedencia del pin sobre latest antigua en caché. Fuentes de actualización solo de probes compilados, no de caché. Sin red implícita en el diagnóstico local. Se conserva el fallback histórico a Current de caché cuando el lector local falla; no convertir ese contrato previo, por sí solo, en una regresión nueva.
- Cachés y manifiesto: conservar entradas no regulares preexistentes y sobrevenidas y archivos sustituidos después del plan. Retener original mediante descriptor hasta aplicar o descartar para que su inode no se reutilice durante ese intervalo. Validación de tipo/identidad, sin promesa de retirada atómica frente a sustitución concurrente durante la propia acción. Si no se puede retener con seguridad, informar fallo y no borrar.
- Plan dueño de recursos, Close explícito/idempotente, acciones con recursos de un solo uso. Cierre de retenidos incluso si acción falla/cambia/desaparece; al descartar y al salir por simulación, cancelación, falta de terminal o error. Quien llama directamente BuildUninstallPlan debe cerrar. Manifiesto al final, condicionado a ausencia de fallos previos, sin romper reintentos.
- Pins pi-antigravity 0.14.0 y pi-cc-extensions 1.0.20; retirada solo actuales/históricos conocidos. Preservar configuración ajena. Fallo/cancelación de proceso nunca da versión. Doctor sin red.
- Extracción limitada a 10.000 entradas, 256 MiB de flujo y 200 MiB de contenido; cancelación entre entradas, limpieza tras fallo, rechazo de escape/enlaces.
- Gate de release rechaza consulta fallida o nuevas estables no rechazadas, salvo escape explícito `PI_GOMEMORY_ALLOW_STALE_PINS=1`.

## Orientación y límites de grafo

Se consultó codegraph para Close/planRegularRemoval/Uninstall y effectiveDepState/RunDepsUpdate; fuente actual leída. Codebase-memory generación 2026-11-14T12:00:58Z, sin watcher y archivos modificados/nuevos: solo orienta navegación, mandan fuentes actuales. Si no dispones de MCP, usa lectura directa y no afirmes tenerlo. El coordinador comprobará el digest antes y después. No hay autorización para correcciones, commits o release.
