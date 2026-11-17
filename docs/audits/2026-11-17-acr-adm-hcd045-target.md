# ACR/ADM — target congelado HCD-045 en curso

## Identidad
Base HEAD `49cbbe91cdab33ec6f023ba09e6afe87ef5f43b8` con cambios sin commit. Snapshot independiente y estable: `/tmp/pi-gomemory-hcd045-audit.LqKdyy`. SHA256 de manifiesto de producto: `71ae56a2394743fbb3050c541e726f5b087c2fd04818e4e3c6d572277e02c1a5`. Copia verificada por rsync checksum sin diferencias inmediatamente después de capturar. Inspeccionar SOLO esa copia, no el árbol activo que sigue en implementación.

## Alcance y contratos
Revisar `extensions/gomemory-context.ts`, `extensions/gomemory-plan.ts`, `pkg/installer/instructions.go`, `installer.go`, `extension.go`, `doctor.go`, `mcp.go`, `uninstall.go`, `main.go` y tests HCD045. Spec detallada en snapshot `hcd/HCD-045-pi-recibe-el-protocolo-y-el-contexto-en.md` (criterios, no asumir que tests existentes prueban todo).
- Cada session_start inicia mem context con cwd de sesión y timeout 3000ms sin bloquear; before_agent_start espera carga e inyecta sección gomemory. Fallo/vacío/timeout: bloque Pi y un aviso por sesión. Una carga por sesión; evitar cruce entre proyectos/sesiones.
- Truncar contexto a 24576 bytes sin partir UTF8 e indicar cómo recuperar resto.
- Bloque Pi deriva herramientas de selectedTools/hiddenTools y convive con sección del plan sin sustituir systemPrompt.
- Install deja de escribir protocolo en AGENTS y retira formato histórico marcado/sin marcadores, preservando texto ajeno, permisos y symlinks; archivo solo bloque se borra, symlink se vacía. Fallo ordinario se avisa sin abortar; destino inválido mantiene rechazo seguro.
- Extensión instalada/retirada en lifecycle; doctor detecta protocolo restante. Comprobar API real Pi instalada 1.5.2 cuando sea necesario, no solo fakePi.
- Datos sintéticos, sin instalar paquetes ni escribir HOME real; no cambiar código, memoria ni HCD. HCD045 todavía en implementación: distinguir faltantes de entrega de defectos del código capturado.

## Exploración previa (Verify con fallback fuente)
Grafo proyecto `Users-josegomezj-home-rcw-pi-gomemory`, generación 2026-11-14T12:00:58Z y watcher off. Cobertura: las rutas cambiadas metadata_changed/not_tracked; leer fuente snapshot. CodeGraph motor extraction28/índice27; mostró fuente actual y conexiones InstallAll→InstallMCPServers→CodemodeMCPServers. Rutas relevantes: session_start→load(pi.exec mem context); before_agent_start→buildSection→piBlock; InstallAll→retireLegacyStep→RetireLegacyInstructions→stripInstructions/writeStrippedAgents; uninstall→planAgents→writeStrippedAgents; InstallTUIExtension y lista tuiResources. No hay traza completa fiable de símbolos nuevos en grafo antiguo.

## Contrato independiente
Revisor A o B aislado: revisar desde primeros principios el código real congelado, no solo redacción. No pedir ni inferir hallazgos del otro revisor. Sin historial ni memoria. No inventar defectos. Cada finding: id, location ruta:línea en snapshot, severity CRITICAL/HIGH/MEDIUM/LOW/INFO, category, claim, evidence_class, evidence[], confidence, cierre mínimo. Devolver findings e inspected y límites; findings vacío es válido. No modificar target. No ejecutar comandos reales de instalación o de gomemory context con datos de usuario.
