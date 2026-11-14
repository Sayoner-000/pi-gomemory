# Documentación de pi-gomemory

Esta carpeta contiene la documentación técnica de pi-gomemory, el instalador y lanzador en Go para configurar Pi Coding Agent dentro de Herdr.

## Contenido

[Guía del harness pi-gomemory (HTML)](Guia-PI-GOMEMORY-v1.13.html)
Guía larga de uso, con el formato de la Guía SDD_HCD de gomemory: instalación, qué cambia en tu máquina, launcher, memoria y grafos, skills, comandos dentro de Pi, `/sidebar`, `/tdd`, HCD, revisión con Splash/ACR/ADM, una sesión de punta a punta y mantenimiento. Se puede imprimir o guardar como PDF.

[Arquitectura del sistema](architecture.md)
Detalla cómo funciona el binario autocontenido con `embed.FS`, la integración de los servidores MCP, el parche para Antigravity, la interfaz TUI y la sanitización de skills.

[Guía de instalación](installation.md)
Instrucciones paso a paso para instalar con el script de una línea, compilar desde el código fuente o usar los binarios de release.

[Guía de uso y comandos](usage.md)
Referencia de comandos de la CLI, cuotas de Google AI Pro, Anthropic y OpenAI Codex y atajos en la terminal interactiva.

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
Hallazgos de la auditoría ACR/ADM de v1.12.1, abiertos y corregidos, y los corregidos en v1.13.0 y v1.12.1, con sus mitigaciones, y los límites de validación vigentes.

[Decisiones de arquitectura (ADR)](adr/README.md)
Registro de las decisiones vigentes: repositorio central de skills, extensiones TUI de HCD y orquestación de dependencias.

## Por qué existe este proyecto

Configurar un agente de código como Pi para trabajar en equipo con memoria persistente suele requerir editar varios archivos JSON a mano, registrar plugins de npm, instalar temas y copiar carpetas de skills una por una.

pi-gomemory empaqueta todo ese proceso en un único binario compilado en Go. No necesita conectarse a internet para copiar las skills o los temas porque todo viaja dentro del ejecutable. Tampoco sobrescribe configuraciones existentes de otros servidores MCP que ya tengas instalados.

## Proyectos vinculados

* [gomemory](https://github.com/Sayoner-000/gomemory) — Memoria colectiva persistente, sinapsis y protocolo de checkpoints y HCD.
* [CodeGraph](https://github.com/colbymchenry/codegraph) — Inteligencia de código mediante grafo SQLite AST local para más de 30 lenguajes.
* [codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp) — Servidor MCP para consultas profundas y análisis de impacto.
