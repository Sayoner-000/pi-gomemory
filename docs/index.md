# Documentación de pi-gomemory

Esta carpeta contiene la documentación técnica de pi-gomemory, el instalador y lanzador en Go para configurar Pi Coding Agent dentro de Herdr.

## Contenido

[Arquitectura del sistema](architecture.md)
Detalla cómo funciona el binario autocontenido con `embed.FS`, la integración de los servidores MCP, el parche para Antigravity, la interfaz TUI y la sanitización de skills.

[Guía de instalación](installation.md)
Instrucciones paso a paso para instalar con el script de una línea, compilar desde el código fuente o usar los binarios de release.

[Guía de uso y comandos](usage.md)
Referencia de comandos de la CLI, consulta del monitor de cuotas de Google AI Pro y atajos en la terminal interactiva.

[Decisiones de arquitectura (ADR)](adr/README.md)
Registro de las decisiones vigentes: repositorio central de skills, extensiones TUI de HCD y orquestación de dependencias.

## Por qué existe este proyecto

Configurar un agente de código como Pi para trabajar en equipo con memoria persistente suele requerir editar varios archivos JSON a mano, registrar plugins de npm, instalar temas y copiar carpetas de skills una por una.

pi-gomemory empaqueta todo ese proceso en un único binario compilado en Go. No necesita conectarse a internet para copiar las skills o los temas porque todo viaja dentro del ejecutable. Tampoco sobrescribe configuraciones existentes de otros servidores MCP que ya tengas instalados.

## Proyectos vinculados

* [gomemory](https://github.com/Sayoner-000/gomemory) — Memoria colectiva persistente, sinapsis y protocolo de checkpoints y HCD.
* [CodeGraph](https://github.com/colbymchenry/codegraph) — Inteligencia de código mediante grafo SQLite AST local para más de 30 lenguajes.
* [codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp) — Servidor MCP para consultas profundas y análisis de impacto.
