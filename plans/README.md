# Planes de implementación

Generados con la skill `improve` el 2026-09-15. Ejecutar en el orden indicado,
salvo que las dependencias permitan trabajo independiente. Cada ejecutor debe
leer el plan completo, respetar sus condiciones de parada y actualizar su fila.

> Estos planes describen el árbol de trabajo actual, que contiene cambios del
> propietario todavía no confirmados sobre `df5abb2`. No limpiar, restaurar,
> resetear ni sobrescribir esos cambios. Un ejecutor en un worktree aislado debe
> recibir primero una rama o commit que contenga esa base; de lo contrario debe
> detenerse.

## Orden y estado

| Plan | Título | Prioridad | Esfuerzo | Depende de | Estado |
|------|--------|-----------|----------|------------|--------|
| 001 | Estabilizar la verificación local y CI | P1 | M | — | DONE |
| 002 | Propagar errores de apagado de lifecycles | P1 | M | 001 | DONE |
| 003 | Hacer la carga de configuración validable, estricta y atómica | P1 | L | 001 | DONE |
| 004 | Convertir `App` en una unidad realmente aislada | P1 | L | 002 | DONE |
| 005 | Endurecer contenedores y adaptadores heredados | P2 | M | 001, 004 | DONE |
| 006 | Actualizar Go, dependencias y supply chain de CI | P1 | L | 001–005 | TODO |
| 007 | Preparar una release v1 verificable | P1 | M | 001–006 | TODO |

Valores: `TODO`, `IN PROGRESS`, `DONE`, `BLOCKED (<razón>)` o
`REJECTED (<razón>)`.

## Registro de verificación

- **Plan 001 — verificado 2026-09-15 en `ecb6389`**: alcance correcto y árbol
  limpio; pasan formato, YAML, `git diff --check`, `go vet`, `go test -count=20
  ./...`, 50 repeticiones de lifecycle/adaptador, el módulo standalone y
  `go test -race` con cobertura atómica de 71,8 % usando Go 1.26.3. La ejecución
  remota de GitHub Actions y `golangci-lint` quedan sin observar: la rama no está
  publicada y el binario del linter no está instalado localmente.
- **Plan 002 — verificado 2026-09-15 en `acf4670`**: alcance limitado a
  `app.go`, `app_test.go` e índice; pasan 20 y 50 repeticiones enfocadas, suite
  completa repetida, race detector, `go vet`, formato y `git diff --check`.
  Los tests verifican errores unidos de startup/rollback, best-effort y orden
  inverso de `Stop`, cancelación limpia, deadline y timeout de apagado.
- **Plan 003 — revisión 2026-09-15 en `ac44dc3`: REVISE**. Pasan tests config,
  100 repeticiones de atomicidad/strict, suite repetida, race, vet, formato,
  ejemplo y diff; el alcance también es correcto. Bloquea aprobación que
  `newTempTarget` parte desde cero: un load exitoso parcial reemplaza defaults
  preexistentes por ceros, en vez de conservar el comportamiento v1. Falta una
  prueba de defaults y de no-aliasing para mapas, slices y punteros.
- **Plan 003 — corregido**: `newTempTarget` ahora clona el valor actual en
  profundidad; un campo omitido conserva su default y los fallos no mutan mapas,
  slices ni punteros preexistentes. Pasan tests config, 100 repeticiones de
  atomicidad/defaults/aliasing, suite repetida, race, vet, formato y ejemplo.
- **Plan 004 — verificado 2026-09-16 en `ee5f4e9`**: alcance limitado a la
  implementación, tests y documentación de `App`; pasan 30 y 100 repeticiones
  de escenarios de instancia, suite repetida, race, vet, formato y diff. Se
  verificaron apps aisladas en paralelo, registro concurrente, rechazo de un
  segundo run, contextos nil, typed nil y snapshot de registros durante un run.
  `ResetApp` permanece explícitamente restringido a tests fuera de un run global.

## Dependencias

- 001 crea una señal de pruebas repetible; todos los cambios posteriores se
  validan contra ella.
- 004 depende de 002 porque las APIs globales e instanciadas deben compartir la
  semántica final de errores de apagado.
- 005 depende de 004 para que las pruebas de aislamiento usen `App` y
  `Container` como fronteras coherentes.
- 006 se hace después del comportamiento funcional para separar fallos de código
  de fallos de toolchain o dependencias.
- 007 sólo puede declarar una versión cuando todos los gates anteriores pasan y
  la historia/documentación coincide con artefactos publicables.

## Hallazgos considerados y descartados

- Dividir la librería en muchos paquetes: descartado por ahora; el dominio es
  pequeño y cohesivo, y la fragmentación agregaría superficie pública sin
  resolver un fallo concreto.
- Incorporar un framework de inyección de dependencias: descartado; `App`,
  `ConfigLoader` y `Container` explícitos son suficientes y más auditables.
- Migrar ya a `go.yaml.in/yaml/v4`: descartado mientras sólo exista una versión
  release candidate; mantener `gopkg.in/yaml.v3` estable y revisarlo al aparecer
  v4 estable.
- Optimizar rendimiento sin benchmark: descartado; no se encontró un hot path ni
  evidencia de presión de CPU/memoria.
- Tratar `WithFilePath` como path traversal: descartado como vulnerabilidad; la
  ruta la proporciona código de bootstrap confiable, no una entrada remota.
- Añadir una licencia elegida por el ejecutor: descartado; la elección legal debe
  hacerla el propietario. El plan 007 bloquea la release hasta resolverla.
