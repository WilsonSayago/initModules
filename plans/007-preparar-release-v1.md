# Plan 007: Preparar una release v1 verificable y honesta

> **Instrucciones para el ejecutor**: Este plan prepara la release; no autoriza
> tag, push, GitHub Release ni publicación. Esas mutaciones requieren instrucción
> explícita del propietario después de revisar el diff y los gates.
>
> **Drift check (primero)**:
> `git diff --stat df5abb2..HEAD -- README.md CHANGELOG.md ROADMAP.md ARCHITECTURE.md docs examples .github go.mod go.sum Makefile LICENSE && git status --short -- README.md CHANGELOG.md ROADMAP.md ARCHITECTURE.md docs examples .github go.mod go.sum Makefile LICENSE`
> Si las APIs resultantes de los planes 001–006 no están presentes o alguno no
> figura `DONE`, detente.

## Estado

- **Prioridad**: P1
- **Esfuerzo**: M
- **Riesgo**: MED
- **Depende de**: planes 001–006
- **Categoría**: release / docs / migration
- **Planificado en**: commit `df5abb2`, 2026-09-15, más cambios locales del propietario

## Por qué importa

README recomienda instalar `v1.6.0` y Changelog marca v1.1–v1.6 como publicados,
pero el último tag visible es `v1.0.6` y las implementaciones están sin commit.
Así, usuarios externos no pueden obtener lo documentado. La release debe partir
de una historia coherente, compatibilidad comprobada y una licencia decidida por
el propietario.

## Estado actual

- `git tag --sort=-v:refname` devuelve como última tag `v1.0.6`.
- `README.md:52-56` indica `go get ...@v1.6.0`.
- `CHANGELOG.md:8` marca `[1.6.0] - 2026-05-26` y también lista v1.1–v1.5,
  aunque no hay tags correspondientes.
- `README.md:152-154` dice “See repository license (if applicable)” y no existe
  `LICENSE`.
- `docs/MIGRATION.md:62-78` planea v2 pero no recuerda que Go exige path
  `/v2` para una nueva major.
- El árbol auditado contiene muchos archivos modificados/no rastreados del
  propietario; no deben atribuirse automáticamente a un ejecutor ni perderse.

## Comandos necesarios

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Estado | `git status --short && git tag --sort=-v:refname | head` | cambios entendidos; v1.0.6 visible antes de release |
| API | `go doc ./...` | exit 0, godoc coherente |
| Suite completa | `make ci` | exit 0 |
| Ejemplo | `make example` | exit 0 y config cargada |
| Módulo publicado | `GOWORK=off go list -m -json github.com/WilsonSayago/initModules@<candidate>` | sólo después de publicar; versión/resumen correctos |
| Diff | `git diff --check` | exit 0 |

## Alcance

**Dentro del alcance**:
- `README.md`, `CHANGELOG.md`, `ROADMAP.md`, `ARCHITECTURE.md`
- `docs/*.md`, `examples/standalone/*`
- `LICENSE` sólo después de decisión explícita del propietario
- metadata/config de release bajo `.github/` si ya existe el mecanismo
- correcciones de godoc en `*.go` sin cambio de comportamiento
- `plans/README.md`

**Fuera del alcance**:
- Crear tags, hacer push, publicar release o paquetes sin autorización nueva.
- Inventar fechas históricas/tags que nunca existieron.
- Elegir una licencia por el propietario.
- Lanzar v2 o cambiar el module path.
- Modificar comportamiento o API para resolver documentación tardía.

## Flujo Git

- Rama sugerida: `codex/007-release-v1-ready` sobre una base confirmada que
  contenga todo el trabajo previo.
- Commits: `prepare v1 release documentation`; licencia en commit separado si el
  propietario la elige.
- No push, tag ni PR sin orden explícita.

## Pasos

### 1. Congelar y explicar la base candidata

Confirmar que planes 001–006 están `DONE`, capturar commit candidato, estado
limpio esperado y lista de APIs exportadas. No crear release desde un árbol
sucio. Cada cambio preexistente debe estar atribuido/revisado por el propietario.

**Verificar**: `git status --short` → vacío en el commit candidato; todos los
planes figuran `DONE`.

### 2. Corregir historia y versión objetivo

Mover cambios no publicados a `## [Unreleased]`. Elegir con el propietario una
única versión candidata basada en SemVer y en el diff real desde v1.0.6. No crear
tags retroactivos v1.1–v1.5 sólo para coincidir con el Changelog. Antes del tag,
README debe referir la última versión realmente disponible o usar `<latest>` en
el branch de preparación; en el commit final de release, debe coincidir con la
tag autorizada.

**Verificar**: `git tag --list` y encabezados de Changelog no afirman releases
inexistentes; `rg -n 'v1\.6\.0|Unreleased' README.md CHANGELOG.md` es coherente
con la decisión registrada.

### 3. Hacer revisión de compatibilidad pública

Comparar API exportada del candidato contra `v1.0.6` con una herramienta oficial
o mantenida de API diff, fijando su versión. Clasificar cada cambio. No eliminar
símbolos v1 ni cambiar firmas publicadas. Cambios de comportamiento estrictos
deben ser opt-in tal como plan 003. Si se detecta breaking change, reparar de
forma compatible o proponer v2 `/v2`; no etiquetar como minor.

**Verificar**: reporte de API diff sin removals/cambios incompatibles para la
minor elegida; suite y ejemplo pasan.

### 4. Resolver licencia con decisión humana

Presentar al propietario las opciones apropiadas y esperar elección explícita.
Sólo entonces añadir texto oficial exacto de la licencia, año/titular confirmados
y actualizar README. Si no hay decisión, dejar este plan `BLOCKED` y no publicar.

**Verificar**: `test -s LICENSE` y la sección License enlaza el archivo; el texto
coincide con la licencia elegida sin modificaciones ad hoc.

### 5. Completar documentación de operación y migración

Quickstart debe compilar, usar APIs recomendadas de los planes 003–005 y declarar
Go mínimo/recomendado del plan 006. Changelog sólo contiene cambios reales.
Migration v2 debe especificar `module github.com/WilsonSayago/initModules/v2` e
imports `/v2`, además de removals planificados. Roadmap separa hecho de futuro.

**Verificar**: copiar/compilar snippets críticos o mantenerlos basados en el
ejemplo ejecutable; `go doc ./...`, `make ci` y `make example` pasan.

### 6. Producir checklist de publicación, sin ejecutarla

Dejar checklist: commit limpio, CI verde, API diff, changelog fechado, licencia,
tag anotada `vX.Y.Z`, GitHub Release, y verificación desde un módulo temporal con
`GOWORK=off`. Incluir rollback/comunicación si la versión publicada falla. No
ejecutar pasos externos hasta autorización explícita.

**Verificar**: revisión humana del diff y autorización registrada; sólo entonces
otro paso/tarea puede crear tag y publicar.

## Plan de pruebas

- Todos los gates del Makefile/CI y módulo de ejemplo.
- API diff contra v1.0.6.
- Snippets de README compilables.
- Instalación desde tag en un módulo temporal después de publicación autorizada.
- Links/documentación y module path v2 revisados.

## Criterios de término

- [ ] Historia/tag/changelog/README no se contradicen.
- [ ] candidato está limpio, con CI y gates completos verdes.
- [ ] no hay breaking API para la minor elegida.
- [ ] licencia fue elegida explícitamente y está presente.
- [ ] docs usan APIs recomendadas y aclaran `/v2`.
- [ ] checklist está listo, pero no se hizo publicación no autorizada.

## Condiciones de parada

- Cualquier plan previo no está `DONE` o CI no está verde.
- El árbol candidato está sucio o contiene cambios no atribuidos.
- El API diff detecta un breaking change para una minor.
- El propietario no ha elegido licencia.
- Se requiere tag, push o publicación sin autorización explícita.
- Un gate falla dos veces tras una corrección razonable.

## Notas de mantenimiento

Después de publicar, verificar el módulo desde fuera del go.work y documentar el
commit/tag exacto. No reconstruir una tag existente: una corrección posterior
debe ser una nueva versión patch.

