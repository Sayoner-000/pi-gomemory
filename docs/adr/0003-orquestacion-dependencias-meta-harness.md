# ADR-0003: Orquestación Desacoplada de Dependencias y Versiones Fijadas

* **Estado**: Aceptado
* **Fecha**: 2026-11-11
* **Contexto Técnico**: HCD-004, HCD-005, HCD-009

## Contexto

El ecosistema `pi-gomemory` articula múltiples herramientas independientes:
- Meta-harness (`pi-gomemory-setup`, launcher `pi-gomemory`)
- Motor cognitivo y memoria (`mem`)
- Motores de grafo de código (`codegraph`, `codebase-memory-mcp`)
- Motor base (`pi`)
- Plugins de extensiones (`pi-antigravity`, `pi-cc-extensions`)

Las comprobaciones síncronas de red en cada arranque bloqueaban la apertura del terminal o fallaban ante caídas de conexión. Además, las actualizaciones automáticas no controladas de plugins de npm introducían incompatibilidades con la API de Pi.

## Decisión

1. **Sondeo Asíncrono no Bloqueante**:
   - `pi-gomemory-setup deps-refresh` corre en segundo plano (`&`) desacoplado del inicio del agente.
   - Las decisiones de advertencia en el launcher usan la caché atómica `pi-gomemory-deps.json` de la ejecución anterior.
   - Si se detectan dependencias desactualizadas y el terminal es interactivo (TTY), se ofrece una ventana de decisión acotada (10 segundos) antes de arrancar Pi.
2. **Fijación Estricta de Versiones de Plugins**:
   - `pi-antigravity` fijado a `@0.12.0`.
   - `pi-cc-extensions` fijado a `@1.0.16`.
   - Los plugins solo se actualizan deliberadamente con nuevas releases de `pi-gomemory-setup`, garantizando estabilidad comprobada.
3. **Mecanismo de Auto-actualización Autocontenido**:
   - Subcomando `update` que descarga el asset compilado desde GitHub Releases, verifica la integridad mediante `SHA256SUMS` oficial y relanza la autoinstalación en caliente.
4. **Última release sin la API de GitHub** (HCD-009):
   - La última versión de pi-gomemory y de `codebase-memory-mcp` se obtiene de la redirección de `https://github.com/<repo>/releases/latest` (cabecera `Location` → `/releases/tag/<tag>`), sin seguirla.
   - Se evita `api.github.com`, limitada a 60 consultas por hora y por IP sin autenticar, que hacía fallar `update` con HTTP 403 en IP compartidas. No se requiere token.

## Consecuencias

* **Positivas**:
  - Tiempo de arranque instantáneo de la sesión de trabajo.
  - Reproducibilidad y ausencia de derivas de versión en plugins críticos.
  - Actualización controlada de extremo a extremo sin intervención manual compleja.
* **Negativas / Compromisos**:
  - El aviso de actualización refleja el estado de la comprobación anterior (máxima latencia: 1 sesión).
