# initModules — Plan de evolución

Objetivo: seguir usando `github.com/WilsonSayago/initModules` como librería compartida de bootstrap entre microservicios (config, instancias compartidas, lifecycle), endureciendo la API sin romper consumidores de golpe.

**Consumidores actuales:** `base-golang`, `groowcity-cron`, `rabbitmq-golang`, `authBase`.

**Principios:**

- Cambios incrementales con versiones semver (`v1.0.x` → `v1.1.0` → `v2.0.0`).
- Cada fase tiene criterios de aceptación y checklist de revisión.
- La lib exporta **mecanismos**; cada microservicio define su **grafo** (config structs, repos, procesos).

---

## Convenciones del plan

| Símbolo | Significado |
|---------|-------------|
| 🔴 | Crítico (seguridad/concurrencia/correctitud) |
| 🟡 | Importante (API, tests, DX) |
| 🟢 | Mejora (docs, deprecaciones suaves) |
| ✅ | Listo para revisar / merge |

**Versionado sugerido:**

- `v1.6.0` — primer tag público después de `v1.0.6` (fases locales 1.1–1.6 nunca se etiquetaron).
- `v2.0.0` — `module github.com/WilsonSayago/initModules/v2`; eliminar APIs deprecadas (`GetInstance` string, globals obligatorios).

---

## Fase 0 — Baseline y inventario ✅

**Objetivo:** saber desde dónde partimos antes de tocar código.

**Completada:** 2026-05-26 — detalle en [docs/INVENTORY.md](docs/INVENTORY.md).

### Tareas

- [x] Listar módulos que importan la lib y versión en `go.mod`.
- [x] Inventariar usos de:
  - [x] `GetInstance(string, ...)`
  - [x] `NewInstance[T]().GetInstance(...)`
  - [x] `AddProp` / `RunLoadProperties` / `SetFilePath`
  - [x] `RegisterProcess` / `Init` / `Run`
- [x] Documentar en un issue/ tabla: servicio → archivo `main` → procesos registrados.
- [x] Confirmar Go mínimo de la lib (`go 1.24.0`, toolchain `go1.27.1`).

### Criterios de aceptación

- [x] Tabla de usos actualizada en este doc (sección [Inventario](#inventario)) y en `docs/INVENTORY.md`.
- [x] `go test ./initModules/...` ejecutable en CI (sin tests aún; `[no test files]`).

### Revisión

- [x] ¿Todos los servicios compilan con la versión actual de la lib? — Sí, vía `go.work`: `authBase`, `groowcity-cron`, `rabbitmq-golang`, `base-golang`.
- [x] Fix menor: `go.work` tenía `#go 1.24.4` (directiva inválida); cambiado a comentario `//`.

---

## Fase 1 — Blindaje crítico (v1.1.0) 🔴 ✅

**Objetivo:** corregir bugs y races sin cambiar el flujo de los consumidores.

**Completada:** 2026-05-26 — ver [CHANGELOG.md](CHANGELOG.md).

### 1.1 Thread-safety en `GetInstance(string, ...)`

- [x] Añadir `sync.Map` + `sync.Once` por key.
- [x] Test concurrente: N goroutines, misma key → una sola instancia.

### 1.2 Orden correcto en carga de properties

- [x] En `RunLoadProperties`: comprobar `err` de unmarshal **antes** de `Validate()`.
- [x] Test: YAML inválido → no debe llamar `Validate()`.

### 1.3 Mensajes y validación de `AddProp`

- [x] Corregir mensaje de error: exige puntero a struct.
- [x] Test: `validatePropTarget` (misma validación que `AddProp`; API con `error` en Fase 2).

### 1.4 Señales en `Run()`

- [x] Quitar `syscall.SIGKILL` de `signal.Notify` (no capturable).
- [x] Documentar señales soportadas en godoc de `Run`.

### Criterios de aceptación

- [x] `go test -race ./...` en `initModules` pasa.
- [x] Sin cambios obligatorios en consumidores (misma firma de funciones).
- [ ] Tag `v1.1.0` y bump en `base-golang`, `groowcity-cron`, etc. (pendiente release git).

### Revisión

- [x] Code review enfocado en locks y orden de errores.
- [x] Actualizar CHANGELOG con fixes 🔴.

---

## Fase 2 — API que devuelve `error` (v1.2.0) 🟡 ✅

**Objetivo:** dejar de usar `log.Fatal` dentro de la librería; el `main` decide si abortar.

**Completada:** 2026-05-26 — ver [CHANGELOG.md](CHANGELOG.md).

### 2.1 Nuevas funciones (mantener las viejas como wrappers)

- [x] `LoadProperties(opts ...Option) error` y `RunLoadPropertiesE()`.
- [x] `AddPropE(p any) error`.
- [x] Las funciones actuales llaman a las nuevas y hacen `log.Fatal` si fallan (deprecadas en comentario).

### 2.2 Opciones de configuración

- [x] Tipo `ConfigLoader` + opciones `WithFilePath`, `WithFormat`, `WithExpandEnv`.
- [x] Globals `propPath` / `propType` / `props` siguen como default de `LoadProperties()` sin opciones.

### Criterios de aceptación

- [x] Tests table-driven para YAML válido/inválido, `.properties`, `${ENV}` en YAML (`testdata/`).
- [x] Consumidor piloto: `rabbitmq-golang`.
- [x] README actualizado con ejemplo `LoadProperties` / `AddPropE`.

### Revisión

- [x] `rabbitmq-golang/cmd/main.go` hace `log.Fatal` solo en `main`.
- [x] Tests usan `testdata/` (sin depender de repos consumidores).

---

## Fase 3 — Unificar instancias compartidas (v1.3.0) 🟡 ✅

**Objetivo:** un solo patrón para “una instancia por app”, manteniendo compatibilidad.

**Completada:** 2026-05-26 — ver [CHANGELOG.md](CHANGELOG.md).

### 3.1 Endurecer `BaseInstance[T]`

- [x] Documentar struct vs pointer `T` en godoc de `Once`.
- [x] `sync.Map` + `sync.Once` por tipo (sin mutex global durante constructor).
- [x] `Once[T](constructor func() T) *T`.

### 3.2 Marcar deprecación de `GetInstance(string, ...)`

- [x] `Deprecated` en godoc.
- [x] Guía de migración en README.

### 3.3 Introducir `Container` (opcional, en la lib)

- [x] `Container` + `NewContainer()` + `OnceIn[T](c, constructor)`.
- [x] Sin tipos de dominio en la lib.

### Criterios de aceptación

- [x] Pilotos `rabbitmq-golang` y `groowcity-cron` (props/suscripciones) usan `Once`; sin nuevos `GetInstance(string)` allí.
- [x] Tests de `Once` bajo `-race`.

### Revisión

- [x] Eliminados `GetInstance("TestSubscription", ...)` en pilotos.
- [x] `GetSubscriptionInstance` sigue recibiendo dependencias por parámetro (use cases).

---

## Fase 4 — Lifecycle y graceful shutdown (v1.4.0) 🔴🟡 ✅

**Objetivo:** reemplazar `go Start()` + `os.Exit(0)` por arranque/parada ordenados.

**Completada:** 2026-05-26 — ver [CHANGELOG.md](CHANGELOG.md).

### 4.1 Nueva interfaz (convivir con `IProcess`)

- [x] `Lifecycle` con `Start`/`Stop` y `context.Context`.
- [x] `ProcessAdapter` para `IProcess`.

### 4.2 `App` / `RunContext`

- [x] `Register` / `RegisterProcess` (adapter).
- [x] `RunContext` + `RunWithSignals` + stop inverso (30s timeout).

### 4.3 Señales

- [x] `signal.NotifyContext` en `RunWithSignals`.
- [x] `Run` sin `os.Exit`.

### Criterios de aceptación

- [x] `groowcity-cron`: Mongo `Disconnect` + Rabbit stop en piloto.
- [x] Tests `app_test.go`: Start → cancel → Stop.

### Revisión

- [x] DB ping en `Start`, `Disconnect` en `Stop` (groowcity-cron).
- [ ] HTTP Gin `Shutdown` en `base-golang` — pendiente Fase 5.

---

## Fase 5 — Patrón por microservicio (sin cambiar la lib) 🟢 ✅

**Objetivo:** cada servicio tiene un composition root claro; la lib solo provee primitivas.

**Completada:** 2026-05-26.

### Por cada servicio (`base-golang`, `groowcity-cron`, `rabbitmq-golang`)

- [x] `internal/bootstrap/` con `Run(ctx) error`.
- [x] `cmd/main.go` delgado (`bootstrap.Run`).
- [x] Sin doble carga props / `Init(true, …)` en servicios migrados.
- [x] **Referencia:** `base-golang` — `Container`, `LoadConfig`, Gin, `HTTPServer` con `Shutdown`, DB `Ping`/`ClosePool`.
- [x] `OnceValue` en wiring de `instance` (JWT, DB) — migración parcial de getters.
- [x] README por servicio enlaza a bootstrap.

### Pendiente gradual (post–Fase 5)

- [ ] Migrar todos los `GetXxxInstance()` / `GetInstance(string)` de `base-golang` al `Container`.
- [ ] Tests de use cases con `initModules.Container` / mocks sin globals.

### Revisión

- [x] Grafo de arranque legible en `base-golang/internal/bootstrap/bootstrap.go`.
- [ ] Tests unitarios sin globals — Fase 5+ / backlog.

---

## Fase 6 — Documentación y CI (v1.6.0) 🟢 ✅

**Completada:** 2026-05-26.

- [x] `README.md`: quickstart, migración (→ `docs/MIGRATION.md`), recetas por caso.
- [x] `CHANGELOG.md` con semver.
- [x] `ARCHITECTURE.md` — lib vs microservicio.
- [x] CI `.github/workflows/ci.yml`: `go vet`, `go test -race -cover`, `golangci-lint`.
- [x] README sin typos legacy (`initComponent`, Features duplicado corregidos antes).

### Criterios de aceptación

- [x] Badge / nota Go 1.24+ (mínimo) y toolchain 1.27.1 en README.
- [x] `examples/standalone` (config only).

---

## Hecho (código en esta rama, tag pendiente)

Todo el trabajo de las fases 1–6 está en esta rama. No hay tags `v1.1.0`–`v1.6.0`.
La publicación está bloqueada por licencia y por autorización explícita de tag;
ver [docs/RELEASE.md](docs/RELEASE.md).

## Futuro

### Licencia y tag v1.6.0

Pendiente de decisión del propietario (licencia + autorización de tag/push).

## Fase 7 — v2.0.0 (breaking, cuando consumidores estén listos)

- [ ] Eliminar `GetInstance(string, ...)`.
- [ ] Eliminar variables globales obligatorias (`SetFilePath` → solo `LoadProperties(opts)`).
- [ ] Eliminar `IProcess` sin contexto (solo `Lifecycle`).
- [ ] Eliminar wrappers que hacen `log.Fatal` internamente.

### Criterios de aceptación

- [ ] Todos los consumidores en `go.mod` usan `github.com/WilsonSayago/initModules/v2`.
- [ ] Guía de migración v1 → v2 publicada (module path `/v2`).

---

## Inventario (Fase 0 — ver [docs/INVENTORY.md](docs/INVENTORY.md))

| Servicio | Versión initModules | main | GetInstance (archivos) | BaseInstance | RegisterProcess / Run |
|----------|---------------------|------|------------------------|--------------|------------------------|
| base-golang | workspace (no en `go.mod`) | `cmd/main.go` | ~20 | Sí (props + use cases) | 5 procesos; `Init(true,true)` — **doble load props** |
| groowcity-cron | v1.0.6 | `cmd/main.go` | ~15 | Sí (props) | DB + Queue; `Run(false,true)` |
| rabbitmq-golang | v1.0.6 | `cmd/main.go` | 1 | No (`prop.sync.Once`) | Queue; `Run(true,true)` |
| authBase | v1.0.6 | — (lib) | 4 | No | No |
| base-golang_old | — | `cmd/main.go` | 1 | Sí | Legacy, fuera de `go.work` |

---

## Orden recomendado de ejecución

```text
Fase 0 → Fase 1 (v1.1.0) → Fase 2 → Fase 3 + Fase 4 (paralelo posible)
         ↓
Fase 5 por servicio (rabbitmq-golang → groowcity-cron → base-golang → authBase)
         ↓
Fase 6 en paralelo desde Fase 1
         ↓
Fase 7 cuando inventario sin GetInstance(string)
```

---

## Checklist de release (cada versión)

- [ ] Tests + race detector verdes.
- [ ] CHANGELOG actualizado.
- [ ] Tag git `vX.Y.Z`.
- [ ] Bump dependencia en consumidores (PR por servicio o monorepo `go.work`).
- [ ] Smoke test manual: arranque + señal SIGTERM + logs de Stop.

---

## Referencias útiles

- [uber-go/fx](https://github.com/uber-go/fx) — DI + lifecycle (alternativa futura).
- [oklog/run](https://github.com/oklog/run) — grupo de actores + señales.
- [spf13/viper](https://github.com/spf13/viper) — config multi-fuente (opcional detrás de `LoadProperties`).
- Composition root: mantener wiring en `cmd` / `internal/bootstrap`, no en domain.

---

## Notas

- **No** mover lógica de negocio a `initModules`.
- Preferir **constructores explícitos** en capas internas; la lib solo garantiza singleton y orden de vida.
- Si el grafo supera ~15 dependencias, evaluar `fx` o `wire` en Fase 7+ sin reescribir la lib entera.

*Última actualización: 2026-09-16*
