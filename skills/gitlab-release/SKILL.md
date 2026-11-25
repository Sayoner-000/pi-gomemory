---
name: gitlab-release
description: Workflow completo de commit, push, release tag y changelog. Agnóstico a cualquier repositorio. Usa Conventional Commits.
---

# GitLab Release Skill

Workflow automatizado para validar cambios, commitear con Conventional Commits, generar tags de release, actualizar CHANGELOG y badge de versión en README.

**Agnóstico:** funciona en cualquier repositorio Git sin configuración previa.

---

## 1. Validación Pre-commit

Antes de commitear, verifica el estado del repositorio:

```bash
git status
git diff --stat
git diff --cached --stat
```

**Checks obligatorios:**
- [ ] No hay archivos staged innecesarios
- [ ] No hay secrets, tokens o credenciales expuestos
- [ ] Los archivos modificados corresponden al cambio reportado
- [ ] No hay merge conflicts pendientes

Si hay archivos sensibles expuestos → **detenerse** y pedir al usuario que los remueva antes de continuar.

---

## 2. Commit con Conventional Commits

Formato estándar:

```
<tipo>(<scope>): <descripción corta>

[opcional: cuerpo con detalles]
```

### Tipos permitidos

| Tipo | Cuándo usar |
|------|-------------|
| `feat` | Nueva funcionalidad |
| `fix` | Corrección de bug |
| `docs` | Cambios en documentación |
| `style` | Formato, sin cambio de lógica |
| `refactor` | Reestructuración sin cambiar comportamiento |
| `test` | Agregar o modificar pruebas |
| `chore` | Mantenimiento, dependencias, CI/CD |
| `perf` | Mejoras de rendimiento |
| `build` | Sistema de build o dependencias externas |
| `ci` | Configuración de integración continua |
| `revert` | Revertir un commit anterior |

### Reglas de redacción

1. **Lenguaje claro y preciso** — sin ambigüedades, sin jerga innecesaria
2. **Presente, no pasado** — "agrega" no "agregó"
3. **Sin punto final** en el asunto
4. **Scope opcional** — solo si aporta claridad (ej: `feat(auth): ...`)
5. **Cuerpo opcional** — explicar el POR QUÉ, no el QUÉ

### Ejemplo de commit

```
feat(auth): agrega autenticación JWT con refresh token

Implementa flujo completo de login/logout con tokens de corta vida
y refresh automático. Incluye middleware de verificación en rutas protegidas.
```

---

## 3. Push a Rama Local

```bash
git push origin <branch>
```

**Antes de push:**
- Verificar que no se está haciendo push a `main` o `master` directamente (salvo que sea la intención)
- Confirmar que el branch remoto existe o crearlo con `-u`

---

## 4. Tag Semántico

Formato: `v<major>.<minor>.<patch>` (ej: `v1.2.3`)

### Determinar la versión

Preguntar al usuario o inferir de los commits:
- `feat` → bump `minor` (1.0.0 → 1.1.0)
- `fix` / `perf` → bump `patch` (1.0.0 → 1.0.1)
- Cambio breaking → bump `major` (1.0.0 → 2.0.0)

### Crear tag

```bash
git tag -a v<version> -m "Release v<version>: <resumen de cambios>"
```

**Mensaje del tag:** lenguaje claro y preciso. Describir QUÉ cambió y POR QUÉ, sin tecnicismos innecesarios.

### Ejemplo

```bash
git tag -a v1.3.0 -m "Release v1.3.0: agrega exportación a PDF y corrige timeout en conexiones lentas"
```

---

## 5. CHANGELOG.md

Si no existe, crearlo con esta estructura base:

```markdown
# Changelog

Todos los cambios notables de este proyecto se documentan en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/).
```

### Formato de entrada

```markdown
## [v<version>] - <YYYY-MM-DD>

### Added
- Descripción de funcionalidad nueva

### Changed
- Descripción de cambio en funcionalidad existente

### Fixed
- Descripción de bug corregido

### Deprecated
- Funcionalidad que será removida

### Removed
- Funcionalidad removida

### Security
- Cambios en seguridad
```

### Reglas

1. **Fecha** en formato ISO 8601: `YYYY-MM-DD`
2. **Sin trailers de IA** — no incluir `Co-authored-by` o referencias a herramientas de IA
3. **Lenguaje claro** — describir cambios como si se le explicara a otro desarrollador
4. **Orden** — entradas más recientes arriba

---

## 6. README Badge (Shields.io)

Buscar en README el badge de versión existente y actualizarlo. Si no existe, agregarlo.

### Formato del badge

```markdown
[![Version](https://img.shields.io/badge/version-<VERSION>-blue)](<repo-url>/releases/tag/v<VERSION>)
```

### Ejemplo

```markdown
[![Version](https://img.shields.io/badge/version-1.3.0-blue)](https://github.com/usuario/repo/releases/tag/v1.3.0)
```

### Búsqueda en README

Patrones a buscar:
- `[![Version]`
- `[![version]`
- `img.shields.io/badge/version-`
- `img.shields.io/badge/v`

Si no se encuentra ningún badge → agregarlo después del título `# ` del README.

---

## 7. Flujo Completo (Ejecución Paso a Paso)

```
1. git status → verificar estado limpio
2. git diff / git diff --cached → revisar cambios
3. git add <archivos> → stagear cambios relevantes
4. git commit -m "feat(scope): descripción" → commit conventional
5. git push origin <branch> → push a rama
6. git tag -a v<version> -m "..." → crear tag
7. git push origin v<version> → push del tag
8. Actualizar CHANGELOG.md → agregar entrada
9. Actualizar badge en README → versión correcta
10. git add CHANGELOG.md README.md → stagear docs
11. git commit -m "docs: actualiza changelog y badge a v<version>"
12. git push origin <branch> → push final
```

---

## 8. Reglas Obligatorias

1. **Sin trailers de coautoría IA** — no agregar `Co-authored-by: ...` ni referencias a Claude, ChatGPT, Copilot u otras herramientas
2. **Lenguaje claro y preciso** — cada commit y tag debe entenderse sin contexto adicional
3. **No commitear secrets** — verificar antes de cada commit
4. **Un cambio lógico por commit** — no mezclar feat con fix en el mismo commit
5. **Verificar después de cada cambio** — `git status` debe mostrar estado limpio al finalizar
6. **Si algo falla, detenerse** — no forzar un commit o tag con errores pendientes

---

## 9. Comandos de Utilidad

```bash
# Ver últimos commits
git log --oneline -10

# Ver commits desde último tag
git log $(git describe --tags --abbrev=0 2>/dev/null || echo "HEAD~5")..HEAD --oneline

# Verificar si hay cambios pendientes
git status --porcelain

# Ver tags existentes
git tag -l "v*" --sort=-version:refname | head -5

# Verificar rama actual
git branch --show-current
```
