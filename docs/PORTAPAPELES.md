# Portapapeles

Pi y Herdr usan el portapapeles del sistema donde se ejecutan. El instalador
diagnostica las herramientas disponibles sin leer ni modificar su contenido.

| Entorno | Herramientas |
| --- | --- |
| macOS | `pbcopy` y `pbpaste`, incluidas en el sistema |
| Linux X11 o WSL con WSLg/X11 | `xclip` y acceso a `DISPLAY` |
| Linux Wayland | `wl-copy` y `wl-paste`, paquete `wl-clipboard` |

En Ubuntu/Debian, instala `xclip` con `sudo apt-get install xclip`. Para una sesión
Wayland instala `sudo apt-get install wl-clipboard`. Otras distribuciones deben
usar su gestor de paquetes. macOS no requiere instalar xclip.

La presencia del ejecutable no comprueba la conexión gráfica. En WSL, verifica
que WSLg esté disponible y que la sesión de Herdr conserve las variables de la
sesión gráfica. En un servidor sin escritorio o por SSH, verifica el soporte OSC 52
del terminal: instalar xclip por sí solo no conecta el portapapeles del cliente.

No se embebe xclip en el binario Go: se instala como dependencia del sistema,
con sus bibliotecas y actualizaciones de la distribución.
