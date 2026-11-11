# ADR-0002: Extensiones TUI Reactivas e Interceptores en Pi

* **Estado**: Aceptado
* **Fecha**: 2026-11-11
* **Contexto Técnico**: HCD-007, HCD-008

## Contexto

El flujo HCD (Harness Continual Development) y el protocolo de memoria requieren visibilidad contextual continua durante el trabajo con agentes. En interfaces estándar de terminal, el desarrollador perdía la noción del estado de las tareas activas, requiriendo consultas manuales constantes (`mem hcd show`).

Adicionalmente, el modo de planificación (`/plan`) requería prevenir mutaciones accidentales del código mientras el modelo diseñaba el árbol de descomposición atómica.

## Decisión

1. **Panel de Tareas Reactivo (`gomemory-tasks.ts`)**:
   - Integración nativa mediante el widget `gomemory-tasks` sobre el editor de Pi (`ctx.ui.setWidget`).
   - Fuente única de verdad: delegación directa en `mem hcd show --json`.
   - Refresco guiado por eventos: activación en `session_start`, `agent_end` y reactividad inmediata tras cualquier comando bash que ejecute `mem hcd`.
   - Comando `/tasks` para alternar visibilidad a demanda.
2. **Modo Plan con Intercepción Estricta (`gomemory-plan.ts`)**:
   - Soporte para argumentos directos: `/plan <solicitud>` activa el modo de inmediato y transfiere el prompt al agente.
   - Bloqueo preventivo de herramientas de escritura (`edit`, `write`) y comandos destructivos de bash definidos en `gomemory-plan-patterns.json`.
   - Inyección contextual del método de descomposición atómica en el system prompt antes de iniciar el turno (`before_agent_start`).
   - Salida controlada mediante `/plan run` o `/plan`.

## Consecuencias

* **Positivas**:
  - Feedback visual inmediato en la consola sin ensuciar el transcript de la conversación.
  - Blindaje determinista contra ejecuciones de código prematuras en modo diseño/plan.
  - Experiencia de usuario integrada en el ciclo de vida del agente.
* **Negativas / Compromisos**:
  - Dependencia de los ganchos de eventos del runtime de extensiones de Pi (`pi.on`).
  - Necesidad de manejo defensivo de la API de UI para compatibilidad en entornos no interactivos o headless.
