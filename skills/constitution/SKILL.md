---
name: constitution
description: Aplica la constitución técnica vigente del proyecto, servida desde la memoria de gomemory. Úsala antes de decidir stack, capas o convenciones.
---

# Constitución del proyecto

La constitución vigente vive en la memoria de gomemory, no en un archivo del
repositorio. Recupérala en el momento de usarla — nunca la reproduzcas de
memoria ni desde una copia antigua.

## Cómo obtenerla

1. Ejecuta `./mem constitution` (o `mem constitution` si el binario está en PATH).
   Si el servidor MCP `gomemory` está conectado, puedes usar en su lugar
   `search_memories` con "constitución" y luego `get_memory` sobre el resultado.
2. Aplica lo que devuelva a la tarea en curso: stack, capas, convenciones,
   prohibiciones.

Si el proyecto usa spec-kit y quieres reflejarla en su archivo:
`./mem constitution --sync` escribe `.specify/memory/constitution.md`.
En un proyecto sin spec-kit no crea nada.

## Si el equipo quiere cambiarla

La constitución es del equipo, no de la herramienta. El contenido que trae
gomemory es solo un punto de partida:

    ./mem docs export constitution -o constitucion.md   # exportar
    # editar constitucion.md
    ./mem docs import constitution constitucion.md      # aplicar

`./mem docs list` muestra si el contenido actual es el que trae la herramienta
o uno propio, y `./mem docs reset constitution` vuelve al de por defecto.

## Referencia de las reglas de trabajo

La constitución dice CÓMO escribir código. Las reglas de trabajo —cuándo
planificar, cómo verificar, cómo tratar un bug— son otro documento, y llegan
solas en `get_context()` al inicio de cada sesión. No las confundas: aplicar un
requisito de la constitución que rompa el flujo real de trabajo es justamente lo
que las reglas de trabajo prohíben.
