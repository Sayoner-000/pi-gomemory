# uninstall no debe tocar un launcher fuera de HOME o del BIN_DIR del manifiesto

ID: HCD-030
Tipo: fix
Carril: fast
Motivo: Vía rápida: fix con reproducción clara (C-004 de la auditoría v1.12.1), sin decisiones abiertas interdependientes; la persona acordó HCD-030 solo con C-004

## Intención
Con HOME apuntando a otra carpeta, pi-gomemory-setup uninstall --yes borró el launcher real ~/.local/bin/pi-gomemory porque launcherDirs (pkg/installer/uninstall.go:106) añade la carpeta del ejecutable en curso. La desinstalación debe limitarse a la instalación de ese HOME: solo considerar la carpeta del ejecutable si está dentro de HOME o es el BIN_DIR registrado en el manifiesto.

Antecedente: HCD-011 (revisión Splash H7) decidió buscar el launcher «en el `BIN_DIR` del manifiesto, en `PI_GOMEMORY_BIN_DIR`, en `~/.local/bin` y junto al ejecutable». La búsqueda junto al ejecutable sirve a instalaciones sin manifiesto (anteriores a HCD-011) con un `BIN_DIR` propio y sin la variable. Esta unidad acota esa búsqueda sin retirar ese caso de forma silenciosa.

## Reproducción
`bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd030-repro.sh`: compila el binario en `<tmp>/bin`, genera ahí el launcher con `harness` y ejecuta `HOME=<tmp>/home pi-gomemory-setup uninstall --yes`. Hoy imprime `[retirar] <tmp>/bin/pi-gomemory — instalado por pi-gomemory` y sale con 1 porque el launcher, fuera de ese HOME, desaparece. Caso real del 2026-11-13: borró `~/.local/bin/pi-gomemory` (restaurado con `pi-gomemory-setup harness`).

## Alcance
Incluido: `launcherDirs` (`pkg/installer/uninstall.go:96`) deja de retirar el launcher junto al ejecutable en curso cuando esa carpeta no está dentro de `HOME` ni es el `BIN_DIR` del manifiesto ni `PI_GOMEMORY_BIN_DIR`; ese launcher se lista en el plan como `[omitir]` con el motivo «fuera de HOME y sin manifiesto: retíralo a mano o usa PI_GOMEMORY_BIN_DIR» y no se borra (Q1 → A); test de regresión en sandbox con el ejecutable fuera de HOME.
Excluido: C-001, C-002 y C-003 de la auditoría v1.12.1 (decisión de la persona); `planSelf` y `--self` (ya exigen el `BIN_DIR` efectivo); las demás carpetas de búsqueda (manifiesto, `PI_GOMEMORY_BIN_DIR`, `~/.local/bin` del HOME); la regla de propiedad por SHA256.

## ¿Qué pasa si no lo hacemos?
Cualquier `uninstall` con un `HOME` distinto del real (pruebas, sandbox, otra cuenta que use el mismo binario) borra el launcher de la instalación real que esté junto al binario, y la persona pierde `pi-gomemory` sin haberlo pedido. Ya ocurrió una vez en este proyecto.

## Alternativa más barata
Documentar «no ejecutes uninstall con otro HOME usando el binario instalado». No protege a nadie que no lea la documentación y deja el defecto activo. Se descarta.


## Aclaraciones
- [x] Q1. ¿Qué hace uninstall con un launcher junto al ejecutable pero fuera de HOME y sin respaldo del manifiesto ni de PI_GOMEMORY_BIN_DIR? A) Omitirlo e informarlo en el plan (recomendado: protege la instalación real y no deja huérfano en silencio el caso H7 de HCD-011). B) Ignorarlo sin mencionarlo. C) Borrarlo si no hay manifiesto (no corrige la reproducción). → A — Omitir e informar (Recomendado)

## Checklist
- [x] 1. La reproducción ya no falla → re-ejecutar la reproducción: `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd030-repro.sh`
- [x] 2. Acotar `launcherDirs` en `pkg/installer/uninstall.go`: el launcher junto al ejecutable fuera de HOME y sin respaldo del manifiesto ni de `PI_GOMEMORY_BIN_DIR` queda como `[omitir]` con su motivo; con el manifiesto que registra ese `BIN_DIR` se sigue retirando → `go test ./pkg/installer -run '^TestHCD030LauncherOutsideHome$' -count=1 -v`; observar primero el rojo con `--expect-fail`
- [x] 3. Mantener verde la suite completa, incluidos los tests de HCD-011 → `go vet ./... && go test ./... -count=1`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-13 06:20 -05:00 · tarea 1 · rojo observado · caducada · `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd030-repro.sh` (salida 1)
        [retirar] /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd030.088V/bin/pi-gomemory — instalado por pi-gomemory
      ✗ uninstall con HOME=/private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd030.088V/home borró /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd030.088V/bin/pi-gomemory, que está fuera de ese HOME
- 2026-11-13 06:20 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD030LauncherOutsideHome$' -count=1 -v` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd030_uninstall_test.go:25:11: undefined: executablePath
      pkg/installer/hcd030_uninstall_test.go:26:3: undefined: executablePath
      pkg/installer/hcd030_uninstall_test.go:27:22: undefined: executablePath
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-13 06:20 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD030LauncherOutsideHome$' -count=1 -v` (salida 1)
      === RUN   TestHCD030LauncherOutsideHome
      === RUN   TestHCD030LauncherOutsideHome/fuera_de_HOME_y_sin_manifiesto:_se_omite_e_informa
          hcd030_uninstall_test.go:44: quiero [omitir] «fuera de HOME y sin manifiesto…», obtengo &{Kind:retirar Target:/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD030LauncherOutsideHomefuera_de_HOME_y_sin_manifiesto_se2178045485/002/pi-gomemory Reason:instalado por pi-gomemory apply:0x1007f81c0}
      === RUN   TestHCD030LauncherOutsideHome/fuera_de_HOME_con_el_BIN_DIR_del_manifiesto:_se_retira
      === RUN   TestHCD030LauncherOutsideHome/junto_al_ejecutable_dentro_de_HOME:_se_retira_como_antes
      --- FAIL: TestHCD030LauncherOutsideHome (0.01s)
          --- FAIL: TestHCD030LauncherOutsideHome/fuera_de_HOME_y_sin_manifiesto:_se_omite_e_informa (0.00s)
          --- PASS: TestHCD030LauncherOutsideHome/fuera_de_HOME_con_el_BIN_DIR_del_manifiesto:_se_retira (0.00s)
          --- PASS: TestHCD030LauncherOutsideHome/junto_al_ejecutable_dentro_de_HOME:_se_retira_como_antes (0.00s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.193s
      FAIL
- 2026-11-13 06:21 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD030LauncherOutsideHome$' -count=1 -v` (salida 0)
      === RUN   TestHCD030LauncherOutsideHome
      === RUN   TestHCD030LauncherOutsideHome/fuera_de_HOME_y_sin_manifiesto:_se_omite_e_informa
      === RUN   TestHCD030LauncherOutsideHome/fuera_de_HOME_con_el_BIN_DIR_del_manifiesto:_se_retira
      === RUN   TestHCD030LauncherOutsideHome/junto_al_ejecutable_dentro_de_HOME:_se_retira_como_antes
      --- PASS: TestHCD030LauncherOutsideHome (0.01s)
          --- PASS: TestHCD030LauncherOutsideHome/fuera_de_HOME_y_sin_manifiesto:_se_omite_e_informa (0.00s)
          --- PASS: TestHCD030LauncherOutsideHome/fuera_de_HOME_con_el_BIN_DIR_del_manifiesto:_se_retira (0.00s)
          --- PASS: TestHCD030LauncherOutsideHome/junto_al_ejecutable_dentro_de_HOME:_se_retira_como_antes (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.195s
- 2026-11-13 06:21 -05:00 · tarea 1 · verde · vigente · `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd030-repro.sh` (salida 0)
      ✓ el launcher fuera del HOME aislado se conserva
- 2026-11-13 06:21 -05:00 · tarea 3 · verde · vigente · `go vet ./... && go test ./... -count=1` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	31.647s

## Siguiente
Cerrada con aprobación sobre la revisión 1da96ab4b3a5.
