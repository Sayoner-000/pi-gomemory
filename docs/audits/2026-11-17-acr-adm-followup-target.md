# Target de seguimiento ACR/ADM: cierres y regresiones

## Identidad

HEAD `5183867a86a800a88638b4234051810e47d4beb0`, árbol sucio actual tras HCD-041. Digest SHA256 del inventario: `8a361933af5b91945ea15ed7d4a316dfd4cb0e61a8ef6b46d33c44550174c607`.

Comando del coordinador: `git ls-files -z --cached --others --exclude-standard -- . ':!:docs/audits/**' ':!:hcd/**' ':!:.codegraph/**' ':!:.codebase-memory/**' | LC_ALL=C sort -zu | xargs -0 shasum -a 256 | shasum -a 256`.

## Encargo independiente

Revisa fuentes actuales desde primeros principios, no solo este manifiesto. Solo lectura, sin modificar código, memoria ni configuración personal. No recibes resultados de otros revisores. No ejecutes instalación/desinstalación contra HOME real. No inventes hallazgos. Devuelve JSON con findings [{local_id, location, severity, category, claim, evidence_class, evidence, confidence}], inspected y limitations; colección vacía válida. Severidades CRITICAL/HIGH/MEDIUM/LOW/INFO. No confundas hipótesis con pruebas. Informa si no puedes ejecutar comandos.

## Alcance acotado

Delta por alcance explícito, no por diff de commits: `pkg/installer`, `scripts/build-release.sh` y rutas de entrada `main.go`; prioridad `deps.go`, `uninstall.go`, `pins.go`, `pins_candidate.go`, `plugin.go`, `doctor.go`, `jsonfile.go`, pruebas HCD-039/040/041 y gate Shell. No auditar extensiones ni UI completas.

Contratos a verificar: pin compilado prevalece sobre latest de caché antigua; fuentes de actualización solo vienen de probes compilados. Paquetes actuales pi-antigravity 0.14.0 y pi-cc-extensions 1.0.20; uninstall retira pins actuales e históricos conocidos, preservando versiones ajenas y configuración propia. Un enlace/directorio sobrevenido después del plan se preserva antes de iniciar acción; no se promete retirada atómica frente a actor local concurrente durante ella. Un comando que falla o se cancela nunca da versión. Sin caché válida deps-update no actualiza y sale 1 pidiendo deps-refresh; con componentes desconocidos actualiza lo conocido y sale 1. Solo declara al día con estado conocido. Opcionales no instalados no impiden actualizar el resto. Extracción: máximo 10.000 entradas, 256 MiB de flujo tar descomprimido y 200 MiB de contenido; cancelación entre entradas, limpieza tras error, rechazo de escape de rutas/enlaces. Doctor no consulta red. Release gate rechaza novedades estables no rechazadas y fallos de consulta salvo escape explícito PI_GOMEMORY_ALLOW_STALE_PINS=1.

Evalúa funcionamiento, regresiones y robustez defensiva (recursos, fallos parciales, entradas sintéticas, límites, integridad). Distingue incumplimiento real de ampliación del contrato. No hay autorización para corregir, instalar dependencias, hacer commit ni release.

## Navegación y límites

Grafo codegraph consultado: RunDepsUpdate → DepsStatusFor → cachedDeps/liveVersion; versionFromCmd/latestFromCmd → runOutput; planPackages → planPinnedPackages/pinnedEntryUnchanged y planCCExtensions; fetchCandidate → extractCandidate. Codebase-memory generación 2026-11-14T12:00:58Z sin watcher, fuentes nuevas/modificadas no fiables en snapshot: leer código actual. Si no tienes herramientas de grafo, usa lectura directa y no afirmes tener MCP.
