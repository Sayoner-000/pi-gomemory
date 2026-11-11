# ADR-0001: Repositorio Central de Skills y Resolución Jerárquica

* **Estado**: Aceptado
* **Fecha**: 2026-11-11
* **Contexto Técnico**: HCD-003, HCD-006

## Contexto

El asistente Pi Coding Agent descubre skills a través de las rutas listadas en la clave `skills` de `~/.pi/agent/settings.json`. En iteraciones tempranas, el instalador intentaba registrar directamente las carpetas de skills de otros agentes anfitriones (`~/.claude/skills`, `~/.config/opencode/skills`, etc.). 

Esto provocaba tres problemas críticos:
1. **Ocultamiento y colisiones**: En Pi 1.5.1, una ruta en `skills` tapa carpetas locales si los nombres coinciden, o genera colisiones no predecibles cuando múltiples agentes definen la misma skill.
2. **Destructividad en reinstalaciones**: Un instalador sin control de integridad sobreescribía o eliminaba skills modificadas localmente por el desarrollador.
3. **Contaminación de settings**: Multiplicación de rutas absolutas y relativas en la configuración de Pi.

## Decisión

1. **Directorio Central Único**: Unificar todas las skills bajo `~/.pi-gomemory/skills`. Pi únicamente registra `~/.pi-gomemory/skills` en `settings.json`.
2. **Copias Embebidas con Huella Digital Criptográfica**: Las 21 skills oficiales de gomemory se instalan como copias físicas en el repositorio central. Cada una incluye un archivo de control `.pi-gomemory` con el hash SHA-256 de su `SKILL.md`. Si el hash no coincide con el archivo actual (indicando edición manual), el instalador jamás la sobreescribe ni elimina.
3. **Enlaces Simbólicos Jerarquizados para el Anfitrión**: Las skills provistas por otros entornos se enlazan simbólicamente aplicando una precedencia estricta:
   `~/.agents/skills` > `~/.config/opencode/skills` > `~/.claude/skills` > `~/.codex/skills` > `~/.gemini/skills`.
   Las skills embebidas de gomemory tienen prioridad absoluta sobre cualquier otra.
4. **Sanitización Frontmatter YAML**: Proceso de limpieza para escapar campos `description` no entrecomillados con caracteres especiales (`:`), previniendo fallos en el parser estricto de Pi.

## Consecuencias

* **Positivas**:
  - Eliminación completa de colisiones en Pi.
  - Conservación íntegra de modificaciones de usuario (idempotencia y seguridad).
  - Configuración limpia y determinista en `settings.json`.
* **Negativas / Compromisos**:
  - Requiere sincronización (`skills-sync`) en el arranque del harness para reflejar skills recién añadidas en agentes anfitriones.
