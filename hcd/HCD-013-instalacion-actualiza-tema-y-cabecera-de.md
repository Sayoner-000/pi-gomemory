# Instalación actualiza tema y cabecera de gomemory en cualquier SO

ID: HCD-013
Tipo: fix
Carril: fast
Motivo: Fix con causa raíz confirmada (showStartupHeader de pi-cc-extensions solo se añadía si faltaba) y sin decisiones interdependientes

## Intención
Al ejecutar pi-gomemory-setup install/update en Linux, macOS o WSL, el tema gomemory y el banner/cabecera quedan siempre actualizados y activos, aunque existiera configuración previa que los desactivara o sustituyera

## Reproducción
1. Partir de un `~/.pi/agent/pi-cc-extensions.json` previo con `"showStartupHeader": true` (es el valor por defecto de pi-cc-extensions; así está hoy en esta máquina Linux).
2. Ejecutar `pi-gomemory-setup install`.
3. Resultado actual: la clave sigue en `true`, porque `InstallCCExtensions` solo añade claves ausentes. Al abrir Pi, el header de pi-cc-extensions ocupa el único hueco de `setHeader` y el banner de `extensions/gomemory-banner.ts` no aparece. En una máquina sin ese archivo previo (instalación limpia, por ejemplo el Mac) sí se ve: por eso el resultado depende del entorno.
4. Esperado: tras instalar o actualizar, `showStartupHeader` queda en `false`, el tema `gomemory` está activo y el banner de gomemory se ve, sea cual sea el SO o la configuración previa.

## Alcance
Incluido: imponer en cada instalación las claves de pi-cc-extensions que gestiona pi-gomemory (`showStartupHeader=false`) sin tocar el resto de preferencias de la persona; comprobar que el tema y las extensiones (`extensions/gomemory-banner.ts`) se reescriben siempre; que `doctor` avise si la cabecera de pi-cc-extensions está activa; tests que lo demuestran partiendo de una configuración previa contraria.
Excluido: publicar la release; cambiar cómo `install.sh` elige la versión (`VERSION` fija) o compila desde el repo local; rediseñar el banner; cambios en Pi o en pi-cc-extensions.

## ¿Qué pasa si no lo hacemos?
Quien ya tenía pi-cc-extensions configurado nunca ve el banner de gomemory aunque reinstale o actualice, y la apariencia cambia de una máquina a otra. El fallo parece un problema del binario y lleva a reinstalaciones inútiles.

## Alternativa más barata
Pedir a cada persona que ponga a mano `"showStartupHeader": false` o lo desactive con `/ccstyle`. Se descarta porque no se puede repetir de forma fiable, el instalador lo debe garantizar y se pierde en cada máquina nueva.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. La reproducción ya no falla: install fuerza showStartupHeader=false y conserva claves propias → go test ./pkg/installer -run TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean
- [x] 2. doctor avisa si showStartupHeader está activo en pi-cc-extensions.json → go test ./pkg/installer -run TestDoctorWarnsCCStartupHeader
- [x] 3. Reinstalar sobre tema y banner modificados los restaura al embebido → go test ./pkg/installer -run TestReinstallRestoresThemeAndBanner

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 08:33 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean` (salida 1)
      ══════════════════════════════════════════════════════════════╝[0m
        [38;2;244;198;107m⚠[0m Portapapeles del sistema         [38;2;145;169;197minstalar xclip para X11/WSLg o wl-clipboard para Wayland con el gestor de paquetes[0m
        [38;2;255;119;127m✗[0m Binario pi                       [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/tmp/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean1971488303/001/.pi/agent/themes/gomemory.json[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/tmp/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean1971488303/001/.local/bin/pi-gomemory[0m
      --- FAIL: TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean (0.24s)
          installer_test.go:175: showStartupHeader activo: el header de pi-cc-extensions tapa el banner de gomemory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.241s
      FAIL
- 2026-11-11 08:33 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestDoctorWarnsCCStartupHeader` (salida 1)
      --- FAIL: TestDoctorWarnsCCStartupHeader (0.00s)
          hcd013_test.go:23: doctor no avisa de showStartupHeader activo: <nil>
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.007s
      FAIL
- 2026-11-11 08:34 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.229s
- 2026-11-11 08:34 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestDoctorWarnsCCStartupHeader` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.023s
- 2026-11-11 08:34 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestReinstallRestoresThemeAndBanner` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.264s
- 2026-11-11 08:34 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean` (salida 0)
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 08:34 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestDoctorWarnsCCStartupHeader` (salida 0)
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 08:34 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestReinstallRestoresThemeAndBanner` (salida 0)
      ok  	pi-gomemory/pkg/installer	(cached)

## Siguiente
Cerrada con aprobación sobre la revisión 92c00b02c2d5.
