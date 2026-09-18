# Diagramas (Archify)

Diagramas interactivos del toolkit `initModules`. Abrí los `.html` en el navegador (tema claro/oscuro, pan/zoom, vistas guiadas).

> Contenido autorado en español. La UI fija del visor Archify permanece en inglés (`html lang` fallback).

| Diagrama | Tipo | HTML | Especificación |
|----------|------|------|----------------|
| Mapa de componentes | architecture | [initmodules-architecture.html](./initmodules-architecture.html) | [JSON](./initmodules-architecture.json) |
| Arranque típico | workflow | [initmodules-startup.html](./initmodules-startup.html) | [JSON](./initmodules-startup.workflow.json) |
| Ciclo de vida de `App` | lifecycle | [initmodules-app-lifecycle.html](./initmodules-app-lifecycle.html) | [JSON](./initmodules-app.lifecycle.json) |

## Regenerar

Desde el paquete Archify instalado en el entorno del agente:

```sh
ARCHIFY="$HOME/.agents/skills/archify/bin/archify.mjs"
REPO="$(pwd)"  # raíz de initModules
DIR=docs/diagrams

node "$ARCHIFY" deliver architecture "$DIR/initmodules-architecture.json" \
  "$DIR/initmodules-architecture.html" --quality showcase --repo-root "$REPO"

node "$ARCHIFY" deliver workflow "$DIR/initmodules-startup.workflow.json" \
  "$DIR/initmodules-startup.html" --quality showcase

node "$ARCHIFY" deliver lifecycle "$DIR/initmodules-app.lifecycle.json" \
  "$DIR/initmodules-app-lifecycle.html" --quality showcase
```
