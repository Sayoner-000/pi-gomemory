# Splash en Pi

Splash es un rol de revisión independiente. Su definición neutral vive en
`config/splash.md`; no requiere una instalación de Claude Code.

El instalador incorpora la herramienta `splash_review` a Pi mediante una extensión.
Recibe `document` (ruta del documento) y `lens` (enfoque opcional). Usa el modelo
seleccionado y la autenticación configurada en Pi. Los proveedores personalizados
que solo existan en una extensión pueden requerir configuración adicional del SDK.

Cada llamada crea una sesión en memoria sin historial, instrucciones del proyecto,
skills ni extensiones. Solo habilita herramientas de lectura y excluye MCP. La
revisión tiene un límite de cinco minutos y admite cancelación. No registra por sí
misma decisiones ni aprobaciones HCD.

Después de actualizar la instalación, ejecuta `/reload` en Pi para cargar la
herramienta. Pide «ejecuta splash_review sobre la ruta del documento». La sesión
principal debe examinar el informe antes de registrar la evidencia correspondiente.

La definición del rol es agnóstica. Esta extensión es su adaptador para Pi; otros
harness pueden usar la misma definición con su mecanismo de sesión independiente.
