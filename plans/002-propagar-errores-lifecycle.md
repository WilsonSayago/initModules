# Plan 002: Propagar todos los errores de apagado de lifecycles

> **Instrucciones para el ejecutor**: Ejecuta los pasos y gates en orden. No
> descartes errores para hacer pasar tests. Actualiza `plans/README.md` al final,
> salvo indicación del revisor.
>
> **Drift check (primero)**:
> `git diff --stat df5abb2..HEAD -- app.go app_test.go && git status --short -- app.go app_test.go`
> Confirma además que `app.go` contiene `func (a *App) stopAll(...){...}` sin
> retorno. Si no coincide, detente y reporta drift.

## Estado

- **Prioridad**: P1
- **Esfuerzo**: M
- **Riesgo**: MED
- **Depende de**: `plans/001-estabilizar-verificacion.md`
- **Categoría**: bug / reliability
- **Planificado en**: commit `df5abb2`, 2026-09-15, más cambios locales del propietario

## Por qué importa

Un `Stop` fallido hoy sólo se escribe en logs: el proceso puede salir con código
exitoso aun cuando no cerró conexiones, drenó mensajes o liberó recursos. En un
fallo de `Start`, también se pierde cualquier error del rollback. La API ya
devuelve `error`, por lo que debe conservar toda la información con `errors.Join`.

## Estado actual

`app.go:67-90` inicia en orden, hace rollback al primer fallo y espera cancelación.
`app.go:93-101` tiene esta forma:

```go
func (a *App) stopAll(ctx context.Context, started []Lifecycle) {
    // reverse order
    if err := lc.Stop(ctx); err != nil {
        log.Printf("Stop %s: %v", lifecycleName(lc), err)
    }
}
```

El error se pierde. `RunContext` considera `context.Canceled` como apagado normal
y devuelve `nil`; esa semántica debe mantenerse sólo cuando no haya error de
`Stop`. `app_test.go` cubre shutdown exitoso y rollback por fallo de inicio, pero
no errores múltiples, orden inverso ni timeout.

## Comandos necesarios

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Tests enfocados | `go test -count=20 -run 'TestRunContext|TestStopAll' ./...` | todos pasan |
| Suite | `go test -count=20 ./...` | exit 0 |
| Race | `go test -race ./...` | exit 0 en toolchain sano |
| Vet/formato | `gofmt -w app.go app_test.go && go vet ./...` | exit 0 |

## Alcance

**Dentro del alcance**: `app.go`, `app_test.go`, `plans/README.md`.

**Fuera del alcance**:
- Añadir métodos públicos por instancia a `App` (plan 004).
- Cambiar señales soportadas o timeout por defecto.
- Añadir reintentos de `Stop`, ejecución paralela o logging estructurado.
- Modificar `Lifecycle` o adaptadores heredados.

## Flujo Git

- Rama sugerida: `codex/002-lifecycle-stop-errors`.
- Un commit lógico: `propagate lifecycle shutdown errors`.
- No hacer push ni PR sin orden explícita.

## Pasos

### 1. Hacer que `stopAll` recolecte errores

Cambiarlo a `func (...) error`. Recorrer siempre todos los lifecycles en orden
inverso. Envolver cada error con el nombre del componente (`stop <name>: %w`) y
devolver `errors.Join(errs...)`. Mantener logs si son útiles, pero el retorno es
la fuente de verdad.

**Verificar**: tests unitarios de `stopAll` prueban que todos los `Stop` se llaman
y que `errors.Is` encuentra cada sentinel.

### 2. Unir errores de startup y rollback

Si `Start` falla, ejecutar rollback de los ya iniciados y devolver
`errors.Join(startWrapped, stopErr)`. Conservar el error de inicio incluso si el
rollback también falla. Cancelar el contexto de timeout mediante `defer` o justo
después del rollback, sin fugas.

**Verificar**: test con un fallo de `Start` y dos fallos de `Stop` → los tres son
detectables con `errors.Is` y ambos componentes previos fueron detenidos.

### 3. Definir retorno del shutdown normal

Después de `ctx.Done()`: si `ctx.Err()` es `context.Canceled`, devolver sólo los
errores de `Stop` (o `nil` si no hay). Para `context.DeadlineExceeded`, conservar
ese error y unirlo con errores de `Stop`. No convertir un shutdown defectuoso en
éxito.

**Verificar**: tests separados para cancelación limpia, cancelación con fallo de
stop, deadline limpia y deadline con fallo de stop.

### 4. Probar orden y timeout

Registrar tres componentes y capturar el orden; debe ser `start A,B,C` y `stop
C,B,A`. Incluir un `Stop` que espere `<-ctx.Done()` y confirmar que el timeout
aparece en el error sin bloquear indefinidamente.

**Verificar**: `go test -count=50 -run 'TestRunContext|TestStopAll' ./...` → exit 0.

## Plan de pruebas

- Happy path de cancelación mantiene retorno `nil`.
- `Stop` único y múltiples `Stop` fallidos son observables con `errors.Is`.
- Startup fallido conserva simultáneamente error de inicio y cleanup.
- Se intentan todos los stops en orden inverso aunque uno falle.
- Deadline y timeout no se confunden con cancelación normal.

## Criterios de término

- [ ] Ningún error de `Lifecycle.Stop` se descarta.
- [ ] `errors.Is` funciona para errores originales unidos.
- [ ] orden inverso y ejecución best-effort están probados.
- [ ] suite repetida, race, vet y formato pasan.
- [ ] sólo archivos dentro del alcance cambiaron.

## Condiciones de parada

- El plan 001 no está hecho o la suite base sigue siendo no determinista.
- Resolverlo exige cambiar la firma pública de `Lifecycle`.
- Un componente necesita una política de reintento específica del dominio.
- Un gate falla dos veces tras una corrección razonable.

## Notas de mantenimiento

Revisar con cuidado la semántica de `context.Canceled`: es normal sólo cuando el
apagado termina bien. El plan 004 debe reutilizar exactamente este mismo camino
para las APIs por instancia y globales.

