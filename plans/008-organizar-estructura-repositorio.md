# Plan 008: Organizar archivos sin alterar la API pública

> **Instrucciones para el ejecutor**: Sigue este plan en orden. Esta es una
> reorganización mecánica: no cambies firmas, nombres de símbolos exportados,
> comportamiento de carga, ni semántica de tests. Ejecuta cada verificación. Si
> ocurre una condición de parada, detente e informa; no improvises. Al finalizar,
> actualiza la fila de este plan en `plans/README.md`.
>
> **Drift check (primero)**:
> `git diff --stat 49a34f4..HEAD -- main.go prop_test.go README.md ARCHITECTURE.md ROADMAP.md docs plans/README.md`
> Si alguno de esos archivos cambió, compara el estado actual con los extractos
> de abajo. Si el movimiento o la división ya se hizo parcialmente, detente.

## Estado

- **Prioridad**: P3
- **Esfuerzo**: S
- **Riesgo**: LOW
- **Depende de**: planes 001–006
- **Categoría**: tech-debt / tests / docs
- **Planificado en**: commit `49a34f4`, 2026-09-16

## Por qué importa

La librería es deliberadamente un único paquete público `initModules`, por lo
que su código raíz no debe fragmentarse en subpaquetes. Aun así, el nombre
`main.go` parece un binario cuando contiene wrappers legacy, `prop_test.go` ya
mezcla más de 800 líneas de comportamientos distintos, y la documentación
canónica está repartida entre raíz y `docs/`. Ordenarlo reduce la fricción para
contribuidores sin cambiar imports ni compatibilidad de consumidores.

## Estado actual

- `main.go:9-31` declara `Init` y `Run`, ambos marcados como legacy/deprecated;
  este repositorio es `package initModules`, no `package main`. El archivo debe
  llamarse `legacy_run.go`.
- `prop_test.go` contiene tests de carga general, validación legacy, atomicidad,
  YAML/env estricto y validación de opciones. `config_loader.go` es el
  implementador de esas rutas y `testdata/` ya contiene los fixtures Go
  convencionales; no mover `testdata/`.
- `ARCHITECTURE.md` y `ROADMAP.md` viven en raíz, mientras migración y release
  viven en `docs/`. `README.md` los enlaza en su sección `Documentation`.
- `docs/INVENTORY.md` se describe como inventario de Fase 0 de 2026-05-26, pero
  todavía declara Go 1.26.3; la política actual en `go.mod` es `go 1.24.0` con
  `toolchain go1.27.1`. Es un snapshot histórico, no una fuente canónica actual.
- `plans/` sigue activo porque el Plan 007 está bloqueado; no mover ni borrar
  esos planes en esta tarea.

Convenciones que se deben preservar:

- Todos los archivos de implementación y test usan `package initModules`.
- Los tests se nombran `*_test.go` y los fixtures se resuelven desde `testdata/`
  mediante el helper `testdataPath` de `prop_test.go`.
- Los documentos Markdown de producto usan nombres en mayúsculas (`MIGRATION.md`,
  `RELEASE.md`) y enlaces relativos desde `README.md`.

## Comandos necesarios

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Referencias | `rg -n 'ARCHITECTURE\\.md|ROADMAP\\.md|INVENTORY\\.md|main\\.go' --glob '!plans/**' .` | revisar y actualizar sólo enlaces a archivos movidos |
| Formato | `make fmt-check` | exit 0 |
| Vet | `GOWORK=off go vet ./...` | exit 0 |
| Suite | `GOWORK=off go test -count=20 ./...` | exit 0 |
| Race | `GOWORK=off go test -race ./...` | exit 0 |
| Ejemplo | `(cd examples/standalone && GOWORK=off go test ./...)` | exit 0 |
| Enlaces internos | `rg -n '\]\((ARCHITECTURE|ROADMAP|docs/INVENTORY)\\.md' README.md docs || true` | no referencias al layout antiguo |
| Diff | `git diff --check` | exit 0 |

## Alcance

**Dentro del alcance**:

- renombrar `main.go` a `legacy_run.go` sin modificar su contenido salvo la
  corrección de comentarios que mencionen el nombre antiguo;
- dividir `prop_test.go` en archivos de test por comportamiento, sin eliminar ni
  debilitar asserts;
- mover `ARCHITECTURE.md` a `docs/ARCHITECTURE.md` y `ROADMAP.md` a
  `docs/ROADMAP.md`;
- mover `docs/INVENTORY.md` a `docs/history/inventory-2026-05-26.md` y marcarlo
  como snapshot histórico, preservando su contenido como evidencia;
- actualizar enlaces relativos afectados en `README.md`, `docs/*.md` y
  `plans/README.md` sólo si hacen referencia a rutas movidas;
- `plans/README.md` y este plan.

**Fuera del alcance**:

- crear subpaquetes, `pkg/`, `internal/` o cambiar el import path público;
- modificar `go.mod`, dependencias, CI o configuración de lint;
- actualizar los datos de consumidores del inventario sin una nueva auditoría de
  los repositorios consumidores;
- mover, borrar o renumerar `plans/`;
- crear licencia, tag, push o release (corresponde al Plan 007 y al propietario).

## Flujo Git

- Rama sugerida: `codex/008-organize-repository`.
- Commit sugerido: `organize repository files and documentation`.
- No hacer push, PR, tag ni release sin instrucción explícita.

## Pasos

### 1. Renombrar el wrapper legacy

Usa `git mv main.go legacy_run.go`. Conserva `package initModules`, `Init` y
`Run` exactamente con sus firmas y deprecaciones actuales. Busca referencias a
la ruta antigua: no cambies menciones a `cmd/main.go` de los proyectos
consumidores, ya que no se refieren a este archivo de librería.

**Verificar**: `test -f legacy_run.go && test ! -e main.go && GOWORK=off go test ./...`
→ exit 0.

### 2. Separar los tests de configuración por responsabilidad

Mantén el mismo paquete y los mismos fixtures/aserciones. Reparte el contenido
actual de `prop_test.go` en estos archivos, dejando cada helper una única vez:

- `config_loader_test.go`: carga YAML/properties normal, `ConfigLoader`,
  `AddPropE`, `Prop`, `PropValidator` y wrappers legacy.
- `config_atomic_test.go`: rollback, defaults omitidos, clonado profundo y los
  helpers/tipos usados exclusivamente por esos casos.
- `config_strict_test.go`: YAML estricto, expansión env estricta y no filtración
  de valores.
- `config_options_test.go`: opción nil, formato inválido, cero targets y
  target typed-nil antes de I/O.

No cambies nombres de tests ni los contenidos de `testdata/`. Si un tipo helper
es usado por más de un archivo, colócalo en `config_loader_test.go` o crea
`config_test_helpers_test.go`; no dupliques definiciones.

**Verificar**: `GOWORK=off go test -count=20 ./... && GOWORK=off go test -race ./...`
→ ambos exit 0; `test ! -e prop_test.go` → exit 0.

### 3. Consolidar la documentación canónica y archivar el inventario

Usa `git mv` para mover los dos documentos de raíz a `docs/`. Crea
`docs/history/` y mueve el inventario allí con nombre fechado. Al inicio del
inventario añade una nota breve que diga que es un snapshot histórico de
2026-05-26 y no refleja la política actual de Go ni el estado actual de los
consumidores. No alteres sus tablas históricas para que parezcan actuales.

Actualiza los enlaces de `README.md` para apuntar a:

- `docs/ARCHITECTURE.md`
- `docs/ROADMAP.md`
- `docs/history/inventory-2026-05-26.md`, etiquetado como inventario histórico

Actualiza cualquier enlace Markdown adicional que encuentre la búsqueda del
paso 1. No cambies enlaces externos ni contenido de release no relacionado.

**Verificar**: `test -f docs/ARCHITECTURE.md && test -f docs/ROADMAP.md && test -f docs/history/inventory-2026-05-26.md && test ! -e ARCHITECTURE.md && test ! -e ROADMAP.md && test ! -e docs/INVENTORY.md` → exit 0; la búsqueda de enlaces internos no debe devolver rutas antiguas.

### 4. Verificar que la reorganización no altera el módulo

Ejecuta todos los gates de la tabla. Revisa `git diff --name-status` para que
los cambios sean renombres, división de tests y enlaces/documentación; no debe
haber cambios en APIs ni fixtures. Finalmente, actualiza el estado de Plan 008
en el índice a `DONE` con el commit y los gates ejecutados.

**Verificar**: `make fmt-check && GOWORK=off go vet ./... && GOWORK=off go test -count=20 ./... && GOWORK=off go test -race ./... && (cd examples/standalone && GOWORK=off go test ./...) && git diff --check` → todo exit 0.

## Plan de pruebas

- No se agregan casos funcionales: se preservan todos los tests existentes de
  configuración y sus fixtures.
- La suite repetida cubre que el movimiento de helpers no introduzca estado
  compartido entre archivos.
- El race detector cubre la misma ruta concurrente del paquete tras el rename.
- El ejemplo standalone comprueba que el layout documental no afectó el módulo
  anidado.

## Criterios de término

- [ ] `legacy_run.go` contiene los wrappers legacy y no existe `main.go` raíz.
- [ ] No existe `prop_test.go`; todos sus tests sobreviven divididos por dominio.
- [ ] `testdata/` no se modificó ni movió.
- [ ] Arquitectura y roadmap están bajo `docs/`; el inventario histórico está
  fechado bajo `docs/history/` y README no lo presenta como estado vigente.
- [ ] No quedan enlaces Markdown al layout documental antiguo.
- [ ] Formato, vet, suite repetida, race, ejemplo y `git diff --check` pasan.
- [ ] No se modifican archivos fuera del alcance ni APIs públicas.

## Condiciones de parada

- `main.go` contiene símbolos o build tags no descritos arriba.
- La división exige cambiar una aserción, fixture o comportamiento para que los
  tests pasen.
- Existen consumidores, workflows o enlaces externos dentro del repositorio que
  requieren conservar una ruta documental vieja y no hay redirección posible.
- La reorganización revela que el inventario se usa como fuente operativa actual;
  en ese caso no archivarlo hasta obtener una nueva auditoría de consumidores.
- Algún gate falla dos veces después de una corrección mecánica razonable.

## Notas de mantenimiento

La raíz debe seguir reservada para la API Go, configuración del módulo y los
documentos universales de release. Los documentos de diseño viven en `docs/` y
los snapshots fechados en `docs/history/`. Si aparecen más ejemplos, renombrar
`examples/standalone/` a un nombre de capacidad sólo cuando todos sus enlaces y
su módulo independiente puedan actualizarse juntos. Mantener `plans/` en raíz
hasta que el Plan 007 se cierre; decidir su archivo o retención pública en una
tarea posterior, no aquí.
