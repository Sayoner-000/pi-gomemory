# Banner de Pi con la imagen del logo (pi-gomemory-dark) y respaldo ASCII

ID: HCD-025
Tipo: feature
Carril: fast
Motivo: comportamiento nuevo acotado; la persona eligió imagen fija y el soporte del terminal (Herdr + Ghostty, protocolo Kitty) está verificado en el código de pi-tui

## Intención
Al iniciar Pi en un terminal con soporte de imágenes, el banner muestra el logo pi-gomemory-dark como imagen fija centrada y debajo la línea «HCD · pi - gomemory <versión> · gomemory <versión>»; sin soporte de imágenes, con NO_COLOR o con PI_IMAGE_PROTOCOL=none se mantiene el banner ASCII actual


## Evidencia previa (2026-11-11)
- `assets/pi-gomemory-dark.webp` es un WebP animado de 13 fotogramas (520×440, 90 ms cada uno). El fotograma 1 (459×399) es el logo completo: gota, anillo, «Pi-gomemory» y «HCD (Harness Continual Development)». En PNG ocupa 50 KB.
- pi-tui 1.5.2 (módulo terminal-image del paquete de Pi, `detectCapabilitiesFromEnvironment`): «Herdr forwards OSC 8 hyperlinks and supports the Kitty graphics protocol»; con `TERM_PROGRAM=herdr`, `detectCapabilities()` da `images: "kitty"`. El protocolo Kitty solo acepta PNG. `PI_IMAGE_PROTOCOL=none` desactiva las imágenes. `isImageLine` acepta la secuencia en cualquier posición de la línea, así que se puede centrar con espacios.
- Hoy el banner (`extensions/gomemory-banner.ts`) es una versión ASCII en color de esa misma imagen, más el título con versiones y el lema.

## Alcance
Incluido: PNG fijo del logo `extensions/gomemory-banner.png`, generado del fotograma 1 del WebP, embebido e instalado junto a las extensiones (`tuiResources`). En `extensions/gomemory-banner.ts`, si el terminal soporta imágenes (`getCapabilities().images`), no hay `NO_COLOR` y el PNG existe, la cabecera muestra el logo con el `Image` de pi-tui, centrado y con alto acotado, y debajo la línea «HCD · pi - gomemory <versión> · gomemory <versión>»; el lema se omite porque ya está en la imagen. Las columnas del logo se leen de la propia secuencia de imagen (Kitty `c=`), porque el index de pi-tui no exporta `calculateImageCellSize`. En cualquier otro caso, el banner ASCII actual sin cambios. Imagen fija, sin animación (Q1). Tests nuevos en `pkg/installer/hcd025_test.go`.
Excluido: animar el logo; versión para tema claro; cambiar el contenido del banner ASCII; el resto de extensiones.

## ¿Qué pasa si no lo hacemos?
El banner sigue siendo una aproximación en caracteres del logo, de menor calidad que la imagen real que el terminal de la persona ya puede mostrar.

## Alternativa más barata
Mejorar la resolución del ASCII actual. No necesita soporte de imágenes, pero nunca iguala la imagen y alarga el banner.


## Aclaraciones
- [x] Q1. Q1. ¿Imagen fija o animada en el banner? → Persona: «Imagen fija (Recomendado)»: un fotograma completo del logo como PNG con la línea de versiones debajo; sin soporte de imágenes, el ASCII actual

## Checklist
- [x] 1. Generar `extensions/gomemory-banner.png` (fotograma 1 del WebP) e instalarlo junto a las extensiones → `go test ./pkg/installer -run TestHCD025BannerPNGInstalled -count=1`
- [x] 2. Mostrar en la cabecera el logo centrado y la línea de versiones cuando el terminal soporta imágenes → `go test ./pkg/installer -run TestHCD025BannerImage -count=1`
- [x] 3. Mantener el banner ASCII sin soporte de imágenes, con NO_COLOR o sin el PNG → `go test ./pkg/installer -run TestHCD025BannerFallback -count=1`
- [x] 4. Mantener en verde toda la suite y `go vet` → `go test ./... -count=1 && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 20:05 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD025BannerPNGInstalled -count=1` (salida 1)
      --- FAIL: TestHCD025BannerPNGInstalled (0.04s)
          hcd025_test.go:19: gomemory-banner.png no se instala: open /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD025BannerPNGInstalled2501089368/001/.pi/agent/extensions/gomemory-banner.png: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.207s
      FAIL
- 2026-11-11 20:05 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD025BannerImage -count=1` (salida 1)
      --- FAIL: TestHCD025BannerImage (0.13s)
          hcd025_test.go:71: TestHCD025BannerImage: FAIL con soporte de imágenes la cabecera debe incluir el logo como imagen | debajo del logo va la línea de versiones: "                                          HCD · Harness Continual Development" | el lema ya está en la imagen: no se repite | con imagen no se dibuja el ASCII
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.324s
      FAIL
- 2026-11-11 20:07 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD025BannerPNGInstalled -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.212s
- 2026-11-11 20:07 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD025BannerImage -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.292s
- 2026-11-11 20:07 -05:00 · tarea 3 · falló · caducada · `go test ./pkg/installer -run TestHCD025BannerFallback -count=1` (salida 1)
      --- FAIL: TestHCD025BannerFallback (0.37s)
          --- FAIL: TestHCD025BannerFallback/sin_el_PNG_instalado (0.12s)
              hcd025_test.go:96: sin el PNG instalado: FAIL no debe emitir imagen | debe dibujar el banner ASCII | el ASCII mantiene el lema
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.532s
      FAIL
- 2026-11-11 20:07 -05:00 · tarea 4 · falló · caducada · `go test ./... -count=1 && go vet ./...` (salida 1)
      ═════════════════════════════╝[0m
        [38;2;244;198;107m⚠[0m Portapapeles del sistema         [38;2;145;169;197mse requieren pbcopy y pbpaste del sistema[0m
        [38;2;255;119;127m✗[0m Binario pi                       [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean2771239354/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean2771239354/001/.local/bin/pi-gomemory[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	17.409s
      FAIL
- 2026-11-11 20:08 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestHCD025BannerPNGInstalled -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.195s
- 2026-11-11 20:08 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD025BannerImage -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.293s
- 2026-11-11 20:08 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD025BannerFallback -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.523s
- 2026-11-11 20:08 -05:00 · tarea 4 · verde · vigente · `go test ./... -count=1 && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	17.350s

## Siguiente
Cancelada: el banner con imagen tiene el defecto de desaparecer al hacer scroll; dejar el banner ASCII
