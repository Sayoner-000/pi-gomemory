# Documentación de pi-gomemory

Esta carpeta contiene la documentación técnica de pi-gomemory, el instalador y lanzador en Go para configurar Pi Coding Agent dentro de Herdr.

## Contenido

[Guía del harness pi-gomemory (HTML)](Guia-PI-GOMEMORY-v1.15.html)
Guía larga de uso, con el formato de la Guía SDD_HCD de gomemory: instalación, qué cambia en tu máquina, launcher, memoria y grafos, skills, comandos dentro de Pi, `/sidebar`, `/tdd` (retirado en HCD-085), HCD, revisión con Splash/ACR/ADM, una sesión de punta a punta y mantenimiento. Se puede imprimir o guardar como PDF.

[Arquitectura del sistema](architecture.md)
Detalla el binario autocontenido con `embed.FS`, la integración de los servidores MCP, la retirada de Antigravity y de `/tdd` (HCD-085), la interfaz TUI y la sanitización de skills.

[Guía de instalación](installation.md)
Instrucciones paso a paso para instalar con el script de una línea, compilar desde el código fuente o usar los binarios de release.

[Guía de uso y comandos](usage.md)
Referencia de comandos de la CLI, límites de Anthropic y OpenAI Codex y atajos en la terminal interactiva.

[Antigravity: HTTP 429 y mitigación temporal](antigravity.md)
Histórico: la integración con Antigravity se retiró en HCD-085.

[Notas de v1.31.0](releases/v1.31.0.md)
Retira `/tdd` y la integración con Antigravity (plugin, cuota de `/sidebar`, `usage` y guard), y limpia al actualizar lo que dejaron versiones anteriores sin tocar lo que editaste (HCD-085).

[Notas de v1.30.0](releases/v1.30.0.md)
Integra las skills externas `impeccable`, `frontend-design` y `playwright-cli` desde skills.sh (`npx skills`) en `install`, `update` y el arranque de Pi (`skills-update --if-due` cada 24 h), añade la comprobación de `doctor` y embebe `gitlab-release` (21 skills embebidas, HCD-084).

[Notas de v1.29.3](releases/v1.29.3.md)
Corrige siete hallazgos de validación y robustez (HCD-083): retira `codebase-memory-mcp` de `scripts/clean-install-check.sh`, respeta el modo `/plan` en el auto-refresh de `gomemory-context.ts`, reintenta `mem context` tras fallos transitorios, añade timeout a `pi install`, evita un pánico en `Sync` sin sondas, propaga errores de `lite.json` corrupto y valida la versión en `scripts/build-release.sh`.

[Notas de v1.29.2](releases/v1.29.2.md)
La skill ACR que instala `pi-gomemory-setup` usa finales de línea LF, igual que la de gomemory, y una prueba impide que una skill embebida vuelva a tener CRLF. El repositorio retira `constitution.md`, porque la Constitución Técnica de gomemory no aplica a pi-gomemory.

[Notas de v1.29.1](releases/v1.29.1.md)
Corrige dos fallos de estado de `v1.29.0`: `skills-sync --restore` ya no borra skills ni devuelve éxito cuando `pi-gomemory` está desinstalado, y `doctor` informa `MCP codegraph` como `INFO` cuando el modo `lite` está activo.

[Notas de v1.29.0](releases/v1.29.0.md)
Adapta pi-gomemory a gomemory 2.59.0: en `/tasks`, una tarea `stale` cuenta como progreso mientras la unidad sigue en `implement`. Pi comprueba sus skills al iniciar la sesión y tras `mem update` (`skills-sync --check`), avisa de las copias editadas que se quedaron atrás y las repone con `skills-sync --restore`.

[Notas de v1.28.0](releases/v1.28.0.md)
Retira codebase-memory-mcp del catálogo, del instalador, de `mcp.json` y de las skills. CodeGraph queda como único grafo externo.

[Notas de v1.27.0](releases/v1.27.0.md)
Corrige el foco y el tono gris en `/tasks` (prioridad `pending` → `failed` → `stale`, indicador `· en curso`, llamadas terminadas en verde), unifica el protocolo de impacto a `codegraph` único y añade auto-refresh debounced de `mem code-refresh`.

[Notas de v1.26.1](releases/v1.26.1.md)
Fija `pi-cc-extensions` en 1.0.22, la versión que publica npm, después de validarla contra lo que pi-gomemory usa de ella.

[Notas de v1.26.0](releases/v1.26.0.md)
Un solo catálogo de componentes alimenta la instalación, `doctor`, `deps-*` y `uninstall`. El nuevo `sync` instala lo que falta y actualiza lo desactualizado (ahora también `herdr`, opcional), con una lista para elegir en la terminal y el comando `/componentes` dentro de Pi. `install.sh` instala siempre la última release.

[Notas de v1.25.1](releases/v1.25.1.md)
Cierra los cuatro hallazgos del ACR sobre `v1.25.0`: truncado de contexto por bytes UTF-8 en el launcher `pi-gomemory`, aviso en `doctor` si falla la lectura de `~/.pi-gomemory/skills`, detección de caché corrupta en `DepsCacheError` y respaldo seguro si falta `gomemory-plan-patterns.json`.

[Notas de v1.25.0](releases/v1.25.0.md)
Sincroniza las skills de Pi desde la copia vigente de `mem` en el anfitrión, añade comprobación de desfase en `doctor`, corrige la ruta de Gemini a `~/.gemini/config/skills` y prioriza `codegraph` como grafo principal en el protocolo de impacto.

[Notas de v1.24.1](releases/v1.24.1.md)
Actualiza las skills embebidas `hcd-implement` y `hcd-verify` para que Pi evite revalidar tareas anteriores por caducidad durante la implementación y no duplique comprobaciones fuera y dentro de `mem hcd verify`.

[Notas de v1.24.0](releases/v1.24.0.md)
`install` y `update` vuelven a dejar Pi en codemode híbrido por defecto (27 tools por turno en lugar de unas 80) y respetan `codemode off`. El protocolo de Pi pide buscar por `codemode` una tool antes de darla por inexistente.

[Notas de v1.23.2](releases/v1.23.2.md)
Avisa cuando codemode queda a medias (`+codemode` con los servidores MCP en directo) y cuando la extensión `cbmem.ts` de codebase-memory-mcp repite las tools del servidor MCP. Lo hacen `install`, `update`, `doctor` y `codemode status`.

[Notas de v1.23.1](releases/v1.23.1.md)
Corrige seis hallazgos de la auditoría de `v1.23.0`: cancelación de suites en `/tdd`, detección de `-skip` y `-short` en `go test`, bloqueo de `hcd_rename` en `/plan`, refresco de `/tasks` al cambiar de sesión y limpieza de temporales con `flock` y staging.

[Notas de v1.23.0](releases/v1.23.0.md)
Añade `Alt+P` (y `Ctrl+Alt+P`) para el modo plan con su estado en la cabecera, cambia la barra y el glifo de tarea caducada de `/tasks` y corrige la versión que mostraba la cabecera tras actualizar.

[Notas de v1.22.0](releases/v1.22.0.md)
Activa `hideThinkingBlock` (`Ctrl+T`) e indicador animado `(thinking...)` por defecto, exige análisis de impacto con `codegraph` y `codebase-memory` siempre y en `/plan`, y corrige los nueve hallazgos de la auditoría de `v1.21.1`.

[Notas de v1.21.1](releases/v1.21.1.md)
Sube `pi-antigravity` a `0.15.0` y corrige señales, limpieza concurrente de temporales, rutas relativas en `PI_GOMEMORY_BIN_DIR` y extracción de tarballs en el paso de herramientas.

[Notas de v1.21.0](releases/v1.21.0.md)
La instalación trae `pi`, `mem`, `codegraph` y `codebase-memory-mcp` desde su canal oficial cuando faltan, y `deps-install` actualiza codebase-memory-mcp desde su release verificado.

[Notas de v1.20.0](releases/v1.20.0.md)
La tarea en curso de `/tasks` parpadea con `●` y el panel se oculta cuando la unidad se cierra, también desde otra sesión de Pi.

[Notas de v1.19.2](releases/v1.19.2.md)
Cierre de los dos hallazgos de la auditoría defensiva del 2026-11-20: `/tdd` no pierde el final de una salida de Bash truncada o con error, e `install.sh` no deja el binario temporal si falla al colocarlo.

[Notas de v1.19.1](releases/v1.19.1.md)
Correcciones de la auditoría de v1.19.0: `/tdd` conserva la salida final de los tests lanzados por Bash, la sugerencia del input no se cuela en el turno siguiente y watch no da por comprobado el frontend tras un `go test`.

[Notas de v1.19.0](releases/v1.19.0.md)
Suite de pruebas el doble de rápida, `/tdd` que muestra en vivo lo que corre por Bash y un filtro para el ciclo rojo-verde, `/tasks` con progreso y ventana desplazable, y sugerencia del siguiente comando en el input.

[Notas de v1.18.0](releases/v1.18.0.md)
Subcomandos explícitos `/plan on`, `/plan off` y `/plan status`, corrección de carga tardía de `mem plan-context` y pruebas unitarias en `tests/gomemory-plan.test.mjs`.

[Notas de v1.17.0](releases/v1.17.0.md)
Árbol de actividad por tarea en `/tasks`, progreso de las verificaciones HCD en `/tdd` y correcciones al modo `lite`.

[Notas de v1.16.0](releases/v1.16.0.md)
Soporte multi-lenguaje y escucha en `bash` en `/tdd`, refresco vivo y guardrail en `/tasks`, modo liviano (`lite on|off`) y guard temporal de Antigravity (HCD-051).

[Notas de v1.15.4](releases/v1.15.4.md)
Modo MCP directo (`codemode off`) por defecto en `install` y `update`, con `codemode on` opt-in persistente y transición limpia desde v1.15.x (HCD-050).

[Notas de v1.15.3](releases/v1.15.3.md)
Regresión offline de Antigravity, cierre verificado de HCD-040 y documentación de la mitigación local opt-in.

[Notas de v1.15.2](releases/v1.15.2.md)
Cierre de los cinco hallazgos (dos medios y tres bajos) de la [auditoría ACR/ADM de v1.15.1](audits/2026-11-18-acr-adm-v1.15.1.md) (HCD-048): re-entrancia en `openSidebar`, códigos de salida en `playwrightLineParser.finish`, marca `.uninstalled` ante fallos parciales, líneas >64 KiB en `SanitizeFrontmatter` y carrera en `/tasks`.

[Notas de v1.15.1](releases/v1.15.1.md)
Corrección de los ocho hallazgos del ACR de v1.15.0: codemode se restaura si falla a medias y respeta `off` sin tocar `defaultTools`, la migración de `AGENTS.md` conserva el texto propio y `uninstall` solo retira lo que puede probar como propio.

[Notas de v1.15.0](releases/v1.15.0.md)
Contexto de gomemory cargado por sesión, codemode híbrido por defecto con elección off persistente y retirada del protocolo heredado de AGENTS.md.

[Notas de v1.14.0](releases/v1.14.0.md)
Modo `codemode` híbrido (`pi-gomemory-setup codemode on|off|status`), validación y subida de pins de Pi (`pi-antigravity@0.14.0`, `pi-cc-extensions@1.0.21`), y cierre de los hallazgos ACR/ADM H-02..H-05 y F-01..F-02.

[Notas de v1.13.8](releases/v1.13.8.md)
Cierre de los dos hallazgos de la auditoría defensiva del 2026-11-17: `/tdd` falla en lugar de acumular sin límite una línea de salida enorme, y la cuota de Antigravity rechaza una respuesta vacía, prueba el endpoint de respaldo y no inventa el plan.

[Notas de v1.13.7](releases/v1.13.7.md)
`PI_GOMEMORY_NO_UPDATE_CHECK` se interpreta igual en el binario, el launcher y el pie de Pi: vacía, `0` o `false` dejan la comprobación activa. `deps-refresh` la respeta.

[Notas de v1.13.6](releases/v1.13.6.md)
Cierre de H-02, el último hallazgo de la auditoría ACR/ADM del 2026-11-14: la configuración de Pi enlazada con symlinks sigue enlazada tras `install`, `update` y `uninstall`, y los enlaces rotos se rechazan sin tocar nada.

[Notas de v1.13.5](releases/v1.13.5.md)
Cierre de H-01, H-03, H-04 y H-05 de la auditoría ACR/ADM del 2026-11-14: `uninstall` conserva el manifiesto si algo falla, cancelar `/tdd` no deja procesos vivos y `update` reinstala en el `BIN_DIR` de la instalación y consulta siempre. H-02 sigue abierto.

[Notas de v1.13.4](releases/v1.13.4.md)
Cierre de H-01, H-02 y H-03 de la auditoría defensiva del 2026-11-13: `uninstall` revalida cada retirada antes de escribir y falla si no pudo leer algo; `update` rechaza un ejecutable vacío, truncado o mayor de 200 MiB.

[Notas de v1.13.3](releases/v1.13.3.md)
Cierre de C-002 y C-003: las consultas de versiones tienen timeout propio y el empaquetado ejecuta las pruebas con el detector de carreras.

[Notas de v1.13.2](releases/v1.13.2.md)
`install`, `update` y `uninstall` conservan los permisos que tenga `~/.pi/agent/AGENTS.md` (C-001 de la auditoría de v1.12.1).

[Notas de v1.13.1](releases/v1.13.1.md)
La comprobación de versiones vuelve a la red cada hora: una release nueva se anuncia como mucho una hora después de publicarse.

[Notas de v1.13.0](releases/v1.13.0.md)
Aviso de actualizaciones que compara con la versión instalada real y aparece en el pie de Pi; `uninstall` acotado a la instalación de su `HOME`.

[Notas de v1.12.1](releases/v1.12.1.md)
Cierre de los seis hallazgos de HCD-027 y corrección del bucle de `/tdd watch`.

[Notas de v1.12.0](releases/v1.12.0.md)
Cambios de cuota Codex, plataformas, verificación de binarios y límites de la entrega.

[Problemas conocidos](known-issues.md)
Estado de los hallazgos recientes, correcciones históricas, mitigaciones y límites de validación.

[Decisiones de arquitectura (ADR)](adr/README.md)
Registro de las decisiones vigentes: repositorio central de skills, extensiones TUI de HCD y orquestación de dependencias.

## Histórico

Las [notas de release](releases/README.md), las guías HTML de versiones anteriores, los [informes de auditoría](audits/README.md) y los ADR conservan la revisión y el alcance que documentan. Para el comportamiento actual, consulta instalación, uso, arquitectura y problemas conocidos.

## Por qué existe este proyecto

Configurar un agente de código como Pi para trabajar en equipo con memoria persistente suele requerir editar varios archivos JSON a mano, registrar plugins de npm, instalar temas y copiar carpetas de skills una por una.

pi-gomemory empaqueta todo ese proceso en un único binario compilado en Go. No necesita conectarse a internet para copiar las skills o los temas porque todo viaja dentro del ejecutable. Tampoco sobrescribe configuraciones existentes de otros servidores MCP que ya tengas instalados.

## Proyectos vinculados

* [gomemory](https://github.com/Sayoner-000/gomemory) — Memoria colectiva persistente, sinapsis y protocolo de checkpoints y HCD.
* [CodeGraph](https://github.com/colbymchenry/codegraph) — Inteligencia de código mediante grafo SQLite AST local para más de 30 lenguajes.
