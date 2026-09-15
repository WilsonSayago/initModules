# Plan 004: Convertir `App` en una unidad realmente aislada y segura para concurrencia

> **Instrucciones para el ejecutor**: Mantén wrappers globales como compatibilidad
> y concentra la lógica en métodos de instancia. Ejecuta cada gate y detente ante
> drift o breaking changes no descritos.
>
> **Drift check (primero)**:
> `git diff --stat df5abb2..HEAD -- app.go app_test.go main.go README.md ARCHITECTURE.md docs/MIGRATION.md && git status --short -- app.go app_test.go main.go README.md ARCHITECTURE.md docs/MIGRATION.md`
> Confirma que `RunContext` aún llama `defaultApp.run` y que `App` sólo contiene
> `lifecycles []Lifecycle` antes de comenzar.

## Estado

- **Prioridad**: P1
- **Esfuerzo**: L
- **Riesgo**: HIGH
- **Depende de**: `plans/002-propagar-errores-lifecycle.md`
- **Categoría**: architecture / reliability / tests
- **Planificado en**: commit `df5abb2`, 2026-09-15, más cambios locales del propietario

## Por qué importa

Aunque existe `App`, sólo el registro es por receptor; `RunContext` y
`RunWithSignals` siempre ejecutan `defaultApp`. Esto impide dos apps aisladas en
un proceso y obliga tests/consumidores a estado global. Una API instance-first
reduce interferencias sin romper wrappers v1.

## Estado actual

- `app.go:23`: `var defaultApp App`.
- `app.go:25-28`: `App` sólo tiene `lifecycles []Lifecycle`, sin mutex.
- `app.go:30-45`: `Register` global delega, pero `(*App).Register` no documenta
  concurrencia.
- `app.go:48-65`: `RunContext` procesa opciones y llama `defaultApp.run`.
- `app.go:116-128`: `RunWithSignals` también termina en el wrapper global.
- `app.go:130-133`: `ResetApp` reemplaza el global sin sincronización.
- `RunContext(nil, ...)` puede hacer panic en `<-ctx.Done()` cuando no pasa por
  `RunWithSignals`.

Mantener nombres del dominio (`App`, `Lifecycle`, `RunOptions`) y paquete único.
La lógica de errores final es la implementada por el plan 002.

## Comandos necesarios

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Tests App | `go test -count=30 -run 'TestApp|TestRunContext|TestRunWithSignals' ./...` | todos pasan |
| Race | `go test -race ./...` | exit 0 en toolchain sano |
| Suite | `go test -count=20 ./...` | exit 0 |
| Vet/formato | `gofmt -w app.go app_test.go main.go && go vet ./...` | exit 0 |

## Alcance

**Dentro del alcance**:
- `app.go`, `app_test.go`, `main.go`
- `README.md`, `ARCHITECTURE.md`, `docs/MIGRATION.md`, `CHANGELOG.md`
- `plans/README.md`

**Fuera del alcance**:
- Integrar `ConfigLoader` dentro de `App` o inventar un DI framework.
- Eliminar funciones globales en v1.
- Reiniciar/reusar automáticamente lifecycles tras una ejecución.
- Ejecutar starts/stops en paralelo.
- Cambiar contratos de `Lifecycle` o `RunOptions`.

## Flujo Git

- Rama sugerida: `codex/004-app-aislada`.
- Commits: `add instance-first app runner`; `document app isolation`.
- No push/PR sin orden explícita.

## Pasos

### 1. Definir construcción y sincronización de `App`

Agregar `NewApp() *App`. Incorporar mutex para proteger registro/snapshot y un
estado de ejecución que rechace dos runs simultáneos sobre la misma instancia
con error sentinel exportado o error documentado estable. No mantener el lock
mientras se llama código del consumidor. `Register(nil)` debe seguir ignorando
nil por compatibilidad, incluyendo interfaces con puntero tipado nil si se puede
detectar sin panic.

**Verificar**: tests para creación, registro concurrente y segundo run rechazado;
`go test -race -run TestApp ./...` pasa.

### 2. Añadir métodos públicos de ejecución

Implementar `(*App).RunContext(ctx, opts) error` y
`(*App).RunWithSignals(ctx, opts) error`. Normalizar `nil` a
`context.Background()` en ambos. Hacer snapshot de lifecycles al iniciar; los
registros posteriores aplican a una ejecución futura o son rechazados según el
contrato documentado, pero nunca aparecen a mitad del run. Reutilizar exactamente
el manejo de errores/rollback del plan 002.

**Verificar**: dos `App` ejecutadas en paralelo sólo inician/detienen sus propios
componentes y pasan con race detector.

### 3. Convertir funciones globales en wrappers finos

`Register`, `RunContext` y `RunWithSignals` deben delegar a `defaultApp` sin
duplicar lógica. Proteger acceso/reset del default global para que `ResetApp` no
corra en carrera; documentar que `ResetApp` es para tests y no debe usarse durante
una ejecución activa.

**Verificar**: tests de compatibilidad prueban las funciones globales y comparan
su semántica con métodos de instancia.

### 4. Documentar el camino recomendado

Actualizar quickstart para mostrar `app := initModules.NewApp()`, registro y
ejecución por instancia; mantener una sección breve de compatibilidad global.
Arquitectura debe mostrar `App` como composition root liviano, no como service
locator. Migration debe indicar que globales permanecen en v1 y se deprecian para
v2. Changelog: entradas bajo `Unreleased`.

**Verificar**: `rg -n 'NewApp|\.RunContext|\.RunWithSignals' README.md ARCHITECTURE.md docs/MIGRATION.md` → documentación y ejemplo presentes.

## Plan de pruebas

- Dos instancias con lifecycles distintos, simultáneas y sin contaminación.
- Registro concurrente sin data race.
- Segundo run simultáneo sobre la misma instancia falla de forma determinista.
- Contexto nil no produce panic.
- Wrappers globales conservan compatibilidad y errores del plan 002.
- Snapshot/registro durante ejecución cumple el contrato documentado.

## Criterios de término

- [ ] Toda lógica de ejecución vive en métodos de instancia.
- [ ] Los wrappers globales son delegación compatible.
- [ ] `App` no tiene races de registro/run/reset.
- [ ] dos apps independientes están cubiertas por tests.
- [ ] suite repetida, race, vet y formato pasan.
- [ ] documentación promueve instance-first sin presentar `App` como DI container.

## Condiciones de parada

- El plan 002 no está completado o la semántica de error diverge entre caminos.
- Hacer `App` segura requiere mantener locks durante callbacks del consumidor.
- Un consumidor confirmado depende de registrar lifecycles a mitad de un run.
- Se requiere eliminar una función global para completar el cambio.
- Un gate falla dos veces tras una corrección razonable.

## Notas de mantenimiento

Revisar especialmente el ciclo del flag `running` en todos los retornos, incluido
fallo de carga, fallo de start y panic no recuperado del consumidor. No hacer a
`App` responsable de construir dependencias; esa sigue siendo labor del bootstrap.

