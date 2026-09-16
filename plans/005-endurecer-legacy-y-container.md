# Plan 005: Endurecer contenedores y puentes heredados contra nil y reflexión insegura

> **Instrucciones para el ejecutor**: Corrige sólo los bordes descritos. No
> rediseñes las APIs legacy ni agregues recuperación genérica de panics.
>
> **Drift check (primero)**:
> `git diff --stat df5abb2..HEAD -- container.go once.go lifecycle.go primaryprocess.go once_test.go app_test.go && git status --short -- container.go once.go lifecycle.go primaryprocess.go once_test.go app_test.go`
> Verifica que `OnceIn(nil, ...)` aún cae al global y que `RunProcesses` usa
> `reflect.TypeOf(p).Elem().Name()`.

## Estado

- **Prioridad**: P2
- **Esfuerzo**: M
- **Riesgo**: MED
- **Depende de**: planes 001 y 004
- **Categoría**: bug / migration
- **Planificado en**: commit `df5abb2`, 2026-09-15, más cambios locales del propietario

## Por qué importa

`OnceIn(nil, ...)` cambia silenciosamente de aislamiento local a estado global,
lo que puede compartir dependencias entre tests o tenants. Los adaptadores legacy
aceptan punteros tipados nil y `RunProcesses` asume que toda implementación es un
puntero, por lo que puede hacer panic. Como estas APIs todavía no están en una tag
posterior a v1.0.6, conviene corregirlas antes de publicar la release pendiente.

## Estado actual

```go
// container.go:17-21
if c == nil { return Once(constructor) }
```

```go
// lifecycle.go:19-21
go a.Process.Start()
return nil
```

```go
// primaryprocess.go:25-29
log.Println("Start processes: ", reflect.TypeOf(p).Elem().Name())
go p.Start()
```

`RegisterProcess` sólo compara `p == nil`, lo que no detecta una interface con
puntero nil. `lifecycleName` en `app.go` ya contiene un patrón de nombre que
tolera valores y punteros y debe reutilizarse/generalizarse.

## Comandos necesarios

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Tests enfocados | `go test -count=30 -run 'TestContainer|TestOnce|TestProcess|TestLegacy' ./...` | todos pasan |
| Race/suite | `go test -race ./... && go test -count=20 ./...` | exit 0 |
| Vet/formato | `gofmt -w container.go once.go lifecycle.go primaryprocess.go once_test.go app_test.go && go vet ./...` | exit 0 |

## Alcance

**Dentro del alcance**:
- `container.go`, `once.go`, `lifecycle.go`, `primaryprocess.go`
- `once_test.go`, `app_test.go` y un nuevo `primaryprocess_test.go` si conviene
- `README.md`, `docs/MIGRATION.md`, `CHANGELOG.md`
- `plans/README.md`

**Fuera del alcance**:
- Eliminar APIs legacy o cambiar `IProcess.Start()` para devolver error.
- Recuperar panics lanzados dentro del código del consumidor.
- Añadir scopes, nombres o resolución dinámica al `Container`.
- Cambiar singleton por tipo a singleton por constructor.

## Flujo Git

- Rama sugerida: `codex/005-legacy-container-guards`.
- Un commit: `harden container and legacy adapters`.
- No push/PR sin orden explícita.

## Pasos

### 1. Hacer fail-fast el container nil

Eliminar la caída silenciosa al registro global. Dado que `OnceIn` aún no existe
en una tag publicada posterior a v1.0.6, mantener la firma y hacer panic con un
mensaje estable y accionable (`initModules: nil Container; use NewContainer or
Once`). Aplicar igual a `OnceValueIn`. Validar constructor nil con mensaje claro;
no aceptar una instancia nil silenciosa si luego la aserción oculta la causa.

**Verificar**: tests prueban panic/mensaje para container y constructor nil, y
confirman que el registro global no fue tocado. Si un constructor devuelve nil o
hace panic dentro de `sync.Once`, una segunda llamada del mismo tipo debe seguir
fallando con un mensaje accionable y nunca con una aserción de tipo sobre nil.

### 2. Detectar interfaces tipadas nil

Crear helper interno pequeño para nil de interfaces (`reflect.ValueOf` y kinds
nilables) y usarlo en `Register`, `RegisterProcess` y `ProcessAdapter.Start` donde
corresponda. `ProcessAdapter.Start` debe retornar error descriptivo antes de crear
goroutine. No llamar `IsNil` sobre kinds no nilables.

**Verificar**: valor válido, nil real y puntero tipado nil no producen panic; los
dos nil son ignorados al registrar o retornan error en el adapter según contrato.

### 3. Hacer seguro el nombre y arranque legacy

Reemplazar `.Elem()` incondicional en `RunProcesses` por un helper que soporte
implementaciones valor y puntero. Saltar nil tipado con log claro, sin goroutine.
Conservar fire-and-forget por compatibilidad y su deprecación; no intentar
convertir panics asíncronos en errores falsamente manejables.

**Verificar**: tests con `IProcess` implementado por valor, puntero y puntero
tipado nil; ningún caso válido hace panic y los válidos se ejecutan una vez.

### 4. Documentar el borde

README/Migration deben decir que `OnceIn` requiere `NewContainer()` no nil y que
legacy sólo se conserva para migración; recomendar `Lifecycle`. Changelog bajo
`Unreleased`.

**Verificar**: `rg -n 'nil Container|NewContainer|IProcess|Lifecycle' README.md docs/MIGRATION.md CHANGELOG.md` → política documentada.

## Plan de pruebas

- `OnceIn`/`OnceValueIn`: aislamiento normal, container nil, constructor nil,
  resultado nil y global intacto; repetir la llamada después de un constructor
  que devolvió nil para comprobar que `sync.Once` no degrada el error.
- `ProcessAdapter`: proceso válido y tipado nil.
- `RegisterProcess`/`RunProcesses`: implementación valor, puntero y tipado nil.
- Suite con race para confirmar que guards no introducen acceso inseguro.

## Criterios de término

- [ ] `OnceIn(nil, ...)` nunca cae silenciosamente al global.
- [ ] Una inicialización fallida no deja una entrada `sync.Once` que produzca un
  panic de aserción de tipo en llamadas posteriores.
- [ ] ningún `.Elem()` inseguro queda en el flujo legacy.
- [ ] interfaces tipadas nil no causan goroutines con panic.
- [ ] comportamiento legacy válido sigue funcionando y está deprecado.
- [ ] suite repetida, race, vet y formato pasan.

## Condiciones de parada

- Se confirma que `Container`/`OnceIn` ya se publicó y consumidores dependen de
  la caída global; en ese caso proponer API aditiva y reportar antes de cambiar.
- Corregir requiere cambiar la firma de `IProcess`.
- Se propone `recover` genérico alrededor del código consumidor.
- Un gate falla dos veces tras una corrección razonable.

## Notas de mantenimiento

En v2, eliminar `IProcess` y decidir si `OnceIn` devuelve error en vez de panic.
El revisor debe verificar que el helper de nil no confunda valores cero válidos.
