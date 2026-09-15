# Plan 001: Estabilizar una verificación local y CI repetible

> **Instrucciones para el ejecutor**: Sigue este plan paso a paso. Ejecuta cada
> verificación antes de avanzar. No limpies ni restaures el árbol: contiene
> trabajo del propietario. Si ocurre una condición de parada, informa y no
> improvises. Al terminar, actualiza `plans/README.md`, salvo que el revisor diga
> que él mantiene el índice.
>
> **Drift check (primero)**:
> `git diff --stat df5abb2..HEAD -- app_test.go once_test.go initinstance_test.go prop_test.go main.go Makefile .github/workflows/ci.yml README.md examples/standalone/go.mod && git status --short -- app_test.go once_test.go initinstance_test.go prop_test.go main.go Makefile .github/workflows/ci.yml README.md examples/standalone/go.mod`
> La segunda parte debe mostrar los cambios no confirmados descritos abajo. Si el
> worktree no los contiene, detente: este plan fue escrito sobre esa base.

## Estado

- **Prioridad**: P1
- **Esfuerzo**: M
- **Riesgo**: MED
- **Depende de**: ninguno
- **Categoría**: tests / dx
- **Planificado en**: commit `df5abb2`, 2026-09-15, más cambios locales del propietario

## Por qué importa

La suite pasa una vez, pero falla con `-count=20` porque los registros globales
persisten entre iteraciones. Además, race/coverage falla antes de ejecutar tests
en la instalación local de Go 1.26.3 y el ejemplo anidado no forma parte de
`go test ./...`. Sin una base repetible, cualquier corrección posterior puede
parecer estable por accidente.

## Estado actual

- `once.go:13` define `var globalSingletons sync.Map` sin limpieza de test.
- `initinstance.go` mantiene otro `sync.Map` global usado por `GetInstance`.
- `once_test.go:25,58,91` usa tipos globales fijos y `t.Parallel()`; una segunda
  ejecución del binario reutiliza singletons de la primera.
- `initinstance_test.go:12,45` usa claves fijas (`concurrent-test-key`,
  `reuse-key`) y no las elimina.
- `prop_test.go:35-40` restablece manualmente `props`, `propPath` y `propType`,
  pero no registra cleanup.
- `.github/workflows/ci.yml:33-34` sólo ejecuta una pasada
  `go test -race -coverprofile=coverage.out -covermode=atomic ./...`.
- `Makefile:3-15` no comprueba formato, repetición ni el módulo
  `examples/standalone`.
- En la máquina auditada: `go test -count=1 ./...` y `go vet ./...` pasan;
  `go test -count=20 ./...` falla por estado global; `go test -race ./...` y
  `go test -cover ./...` fallan con `runtime/race|coverage: package testmain:
  cannot find package`, antes de correr tests. No ocultar ese fallo.
- `gofmt -d *.go examples/standalone/*.go` detecta formato pendiente en
  `main.go` y `prop_test.go`.
- El módulo anidado pasa con:
  `cd examples/standalone && GOWORK=off go test ./...`.

Convención: paquete único `initModules`, tests junto al código y fixtures en
`testdata/`. Mantenerla.

## Comandos necesarios

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Formato | `gofmt -w main.go prop_test.go once_test.go initinstance_test.go app_test.go` | exit 0 |
| Unidad repetida | `go test -count=20 ./...` | todas las pasadas OK |
| Vet | `go vet ./...` | exit 0, sin diagnósticos |
| Ejemplo | `(cd examples/standalone && GOWORK=off go test ./...)` | exit 0 |
| Race/coverage | `go test -race -coverprofile=coverage.out -covermode=atomic ./...` | exit 0 y `coverage.out` no vacío, en toolchain sano |
| Diff | `git diff --check` | exit 0 |

## Alcance

**Dentro del alcance**:
- `app_test.go`
- `once_test.go`
- `initinstance_test.go`
- `prop_test.go`
- `main.go` (sólo formato)
- `Makefile`
- `.github/workflows/ci.yml`
- `README.md` (sólo comandos de desarrollo)
- `plans/README.md` (estado)

**Fuera del alcance**:
- Cambiar APIs productivas o exportar funciones de reset para consumidores.
- Actualizar versiones de Go, actions, golangci-lint o dependencias; corresponde
  al plan 006.
- Cambiar comportamiento de configuración, lifecycle o singleton.

## Flujo Git

- Rama sugerida: `codex/001-verificacion-repetible` sólo si el operador entrega
  una base que incluya los cambios locales actuales.
- Un commit lógico, mensaje acorde al historial informal: `stabilize verification baseline`.
- No hacer push ni abrir PR sin instrucción explícita.

## Pasos

### 1. Aislar tests que comparten registros globales

En tests, registrar `t.Cleanup` que limpie las entradas creadas. Para
`sync.Map`, usar `Clear()` si el mínimo de Go lo soporta; en caso contrario,
usar `Range/Delete` dentro de helpers `_test.go`. El cleanup no debe convertirse
en API exportada. Evitar `t.Parallel` en tests que escriben el mismo registro
global. Las pruebas internas puras pueden seguir paralelas.

**Verificar**: `go test -count=20 ./...` → 20 pasadas sin reutilizar instancias.

### 2. Eliminar polling y sleeps frágiles donde sea razonable

En `app_test.go`, preferir canales con timeout a bucles de sleep. Toda goroutine
iniciada por una prueba debe finalizar antes de que la prueba termine.

**Verificar**: `go test -count=50 -run 'TestRunContext|TestProcessAdapter' ./...`
→ exit 0 sin timeout.

### 3. Formatear y consolidar gates locales

Aplicar `gofmt` sólo a los archivos Go del alcance. Añadir al `Makefile` targets
separados y componibles para formato-check, test normal repetido, race/coverage,
vet, lint y ejemplo. `make ci` debe usar los mismos gates que CI.

**Verificar**: `test -z "$(gofmt -l *.go examples/standalone/*.go)"` → exit 0.

### 4. Cubrir el módulo anidado en CI

Añadir un paso que ejecute sus tests con `working-directory:
examples/standalone` y `GOWORK=off`. Añadir una pasada repetida de unidad antes
de race/coverage. No actualizar todavía versiones de actions.

**Verificar**: inspeccionar el YAML con `ruby -e 'require "yaml"; YAML.load_file(".github/workflows/ci.yml"); puts "ok"'`
→ imprime `ok`; luego ejecutar los comandos equivalentes del Makefile.

### 5. Separar fallo de repositorio de fallo de instalación Go

Ejecutar race/coverage. Si aparece el error `package testmain: cannot find
package`, probar el mismo commit en una instalación oficial limpia de Go 1.26.3
o superior, sin cambiar el código para acomodar un GOROOT roto. Registrar en la
entrega qué toolchain produjo el gate exitoso.

**Verificar**: `go test -race -coverprofile=coverage.out -covermode=atomic ./...`
→ exit 0 y `test -s coverage.out` → exit 0.

## Plan de pruebas

- Repetir tests de `Once`, `GetInstance`, carga global y `App` al menos 20 veces.
- Conservar cobertura concurrente de singleton con race detector.
- Ejecutar el ejemplo como módulo independiente.
- Usar como patrón los tests existentes en `once_test.go`, pero con cleanup
  determinista y sin compartir estado entre casos.

## Criterios de término

- [ ] `go test -count=20 ./...` pasa.
- [ ] `go vet ./...` pasa.
- [ ] race/coverage pasa en una instalación oficial sana y genera artefacto.
- [ ] el ejemplo anidado compila y prueba con `GOWORK=off`.
- [ ] `gofmt -l` no devuelve archivos y `git diff --check` pasa.
- [ ] no se añadió API productiva sólo para tests.
- [ ] no se modificaron archivos fuera del alcance.

## Condiciones de parada

- El worktree no contiene los archivos/API no confirmados descritos aquí.
- El arreglo exige cambiar semántica productiva en lugar de aislar tests.
- Race/coverage sigue fallando en un Go oficial limpio; reportar comando, versión
  y salida completa.
- Un gate falla dos veces después de una corrección razonable.

## Notas de mantenimiento

El revisor debe comprobar que no queden goroutines de test y que los helpers de
reset sólo existan en archivos `_test.go`. El plan 006 modernizará versiones de
CI después de estabilizar este contrato.

