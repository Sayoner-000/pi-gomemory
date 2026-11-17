# Target ACR / ADM — 2026-11-17

Este documento es un manifiesto de alcance, no un informe de hallazgos.

## Identidad congelada

- Repositorio: /Users/josegomezj/home/rcw/pi-gomemory
- HEAD: 5183867a86a800a88638b4234051810e47d4beb0
- Árbol sucio: cambios HCD-039 y HCD-040 incluidos, incluidos archivos nuevos no versionados.
- Digest SHA256 del inventario ordenado (ruta + SHA256 por archivo): c607cab1488ca83608df2d8c593a42928cd8969298a022ed922869b55df7f98c.
- Inventario: git ls-files --cached --others --exclude-standard, excluyendo docs/audits/*, hcd/*, .codegraph/* y .codebase-memory/*. No se revisan esas exclusiones como producción.

## Encargo al revisor independiente

Revisa el CÓDIGO ACTUAL de este repositorio desde primeros principios, no solo este manifiesto. Solo lectura; no modifiques archivos ni memorias, no ejecutes install/uninstall contra el HOME real. No recibes hallazgos de otros revisores. No inventes defectos. Devuelve hallazgos estructurados con local_id, location archivo:línea, severity CRITICAL/HIGH/MEDIUM/LOW/INFO, category, claim, evidence_class, evidence, confidence y lista de archivos inspeccionados. Una lista vacía es válida. Declara fallos de acceso o límites.

## Alcance dirigido

Prioridad: pkg/installer/pins.go, pins_candidate.go, deps.go, doctor.go, plugin.go, uninstall.go y scripts/build-release.sh, con sus pruebas y rutas de entrada. Contrastar preservación de configuración ajena, pins actuales/históricos, caché, red, descarga/extracción tar, verificación de integridad, errores, timeouts, éxito falso y gate de release. Contexto adicional si es necesario: main.go, jsonfile.go y extensiones TypeScript. No es una auditoría exhaustiva de todo el repositorio.

## Contratos de proyecto

Pins aprobados: pi-antigravity 0.14.0 y pi-cc-extensions 1.0.20. deps-update instala solo el pin aprobado; npm publicado es informativo en doctor. doctor no consulta red. pins-check debe detener releases con versiones estables superiores no rechazadas o consulta fallida, salvo escape explícito PI_GOMEMORY_ALLOW_STALE_PINS=1. pins-validate valida contratos textuales, no equivalencia semántica total. uninstall --packages retira pins actuales e históricos conocidos, preserva configuración ajena y revalida antes de ejecutar pi remove. Configuraciones enlazadas deben seguir enlazadas. PI_GOMEMORY_NO_UPDATE_CHECK desactiva comprobaciones automáticas. No se autorizan correcciones, commits ni releases.

## Evidencia de grafo disponible y límites

codegraph_explore consultado sobre CheckPins, validateCandidate, planPinnedPackages, DefaultDepProbes y pinsDoctorItems: CheckPins llama a registryVersions/publishedVersion; validateCandidate llama a fetchRegistryDoc/fetchCandidate/checkContracts; RunDoctor añade pinsDoctorItems; planPackages llama a planPinnedPackages que revalida con pinnedEntryUnchanged antes de pi remove.

Codebase-memory proyecto Users-josegomezj-home-rcw-pi-gomemory, generación 2026-11-14T12:00:58Z, sin watcher; cobertura comprobada de los archivos anteriores: nuevos pins.go/pins_candidate.go no tracked, restantes modificados salvo main.go. Por tanto fuente actual manda, no fiarse del snapshot antiguo. Si no tienes MCP, usa lectura directa del código actual; no afirmes disponer del grafo.
