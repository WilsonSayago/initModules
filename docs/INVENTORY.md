# initModules — Inventario (Fase 0)

**Fecha:** 2026-05-26  
**Versión lib en repo local:** módulo `github.com/WilsonSayago/initModules` (sin tag en workspace; consumidores pinnean `v1.0.6`)  
**Go:** `initModules` → `1.26.3`; `go.work` → `1.26.3`

---

## 1. Módulos consumidores

| Módulo | En `go.work` | `initModules` en `go.mod` | Go (`go.mod`) | Build (`go build ./...` con workspace) |
|--------|--------------|---------------------------|---------------|----------------------------------------|
| `authBase` | Sí | `v1.0.6` | 1.24.x | OK |
| `groowcity-cron` | Sí | `v1.0.6` | 1.24.x | OK |
| `rabbitmq-golang` | Sí | `v1.0.6` | 1.24.x | OK |
| `base-golang` (`jwt/base`) | Sí | **No declarado** (solo vía workspace) | 1.24.0 | OK con `go.work` |
| `base-golang_old` | No | — | — | Legacy / fuera de workspace |
| `messaging`, `translation`, etc. | Sí | No importan initModules | — | — |

### Notas de dependencias

- **`base-golang`:** importa `initModules` y `authBase` en código, pero `go.mod` no los lista. Compila porque están en el mismo `go.work`. Para build fuera del workspace: `go get` / `go mod tidy` pendiente.
- **`rabbitmq-golang`:** usa `AddProp(prop.GetQueueProp())` con singleton propio en `prop` (`sync.Once`), no `NewInstance[T]`.

---

## 2. Uso por API (servicios activos)

### 2.1 `GetInstance(string, ...)` — service locator

| Servicio | Archivos | Keys registradas |
|----------|----------|------------------|
| **groowcity-cron** | 15 | `VendorRepository`, `VendorDao`, `UserRepository`, `UserDao`, `ApplicantRepository`, `ApplicantDao`, `ExportDao`, `ReportUsecase`, `ExcelGenerator`, `LocalFileStorage`, `ReportRepository`, `ReportDAO`, `TestSubscription`, `EnhancedReportSubscription`, `UserService`, `MongoPersistence` (×2: con factory y con `nil`) |
| **base-golang** | 20 | `AuthController`, `ProductRepository`, `ProductStatusHistoryDao`, `ProductDao`, `ContainerDao`, `ContainerRepository`, `UserController`, `RoleController`, `ProductController`, `ProductService`, `ContainerService`, `PermissionRepository`, `UserService`, `PermissionDao`, `RoleDao`, `RoleRepository`, `UserDao`, `UserRepository`, `UserRoleDao`, `Translate` |
| **authBase** | 4 | `RoleService`, `AuthenticationService`, `Authorization`, `ValidationService` |
| **rabbitmq-golang** | 1 | `TestSubscription` |

**Total aproximado:** ~40 call sites en 40 archivos (activos).

### 2.2 `NewInstance[T]().GetInstance(...)`

| Servicio | Usos | Dónde |
|----------|------|-------|
| **groowcity-cron** | 4 | `cmd/main.go` (2 props), `intance/primary.go` (2 en wiring de procesos) |
| **base-golang** | 6 | `cmd/main.go` (2 props), `instance/usecase.go` (2 JwtProp), `instance/primary.go` (1 DatabaseProperty) |
| **rabbitmq-golang** | 0 | — |
| **authBase** | 0 | — |

### 2.3 Config (`SetFilePath`, `AddProp`, `RunLoadProperties`)

| Servicio | Archivo | Formato | Props cargadas |
|----------|---------|---------|----------------|
| **groowcity-cron** | `cmd/main.go` | YML `./internal/resources/properties.yml` | `QueueProp`, `DatabaseProp` (`NewInstance`) |
| **base-golang** | `cmd/main.go` | YML `internal/resources/properties.yml` | `DatabaseProperty`, `JwtProp` (`NewInstance`) |
| **rabbitmq-golang** | `cmd/main.go` | YML `internal/resources/properties.yml` | `QueueProp` vía **`AddPropE` + `LoadProperties`** (Fase 2); `Run(false, true)` |

### 2.4 Procesos (`RegisterProcess`, `Init`, `Run`)

| Servicio | Bootstrap | Procesos registrados | Observación |
|----------|-----------|----------------------|-------------|
| **groowcity-cron** | `Run(false, true)` | `GetDatabaseInstance()`, `GetQueueInstance()` | Props cargadas antes en `main`; `Run` no recarga props |
| **base-golang** | `Init(true, true)` + `r.Run()` | Translate, Postgres Pgx, Auth/User/Role/Product controllers | **Doble carga props:** `RunLoadProperties()` en L35 y `Init(true,…)` otra vez |
| **rabbitmq-golang** | `Run(true, true)` | `GetQueueInstance()` | Piloto más pequeño para migraciones |

**authBase:** solo `GetInstance`; sin `main` propio (librería consumida por `base-golang`).

---

## 3. Detalle `main` por servicio

### groowcity-cron (`cmd/main.go`)

```text
SetFilePath(YML) → AddProp ×2 (BaseInstance) → RunLoadProperties
→ RegisterProcess(Database, Queue) → Run(false, true)  // bloquea en señal
```

### base-golang (`cmd/main.go`)

```text
SetFilePath(YML) → AddProp ×2 → RunLoadProperties → migrations → Gin setup
→ RegisterProcess ×5 → Init(true, true)  // recarga props + Start procesos en goroutine
→ r.Run()  // HTTP (no usa initModules.Run)
```

### rabbitmq-golang (`cmd/main.go`) — piloto Fase 2

```text
AddPropE(GetQueueProp) → LoadProperties(WithFilePath, WithFormat) → RegisterProcess(Queue) → Run(false, true)
```

---

## 4. Tests y toolchain

| Check | Resultado |
|-------|-----------|
| `go test ./initModules/...` (workspace) | OK — sin archivos de test (`[no test files]`) |
| `go test -race` | N/A hasta Fase 1 (añadir tests) |
| `go.work` parse | Corregido: línea `#go 1.24.4` inválida → comentario `//` (bloqueaba `go` con workspace) |

---

## 5. Alineación de versión Go

| Artefacto | Versión |
|-----------|---------|
| `initModules/go.mod` | 1.26.3 |
| `go.work` | 1.26.3 |
| Consumidores (`authBase`, `groowcity-cron`, `rabbitmq-golang`) | 1.24.x en `go.mod` |
| `base-golang` | 1.24.0 |

El workspace impone toolchain 1.26.3 al compilar miembros del work. Recomendación: alinear `go` directive de consumidores a `1.26.3` en una tarea aparte (no bloqueante para Fase 1).

---

## 6. Hallazgos / riesgos

### Resueltos en v1.1.0 (Fase 1)

1. ~~**`GetInstance` sin mutex**~~ — `sync.Map` + `sync.Once` por key.
2. ~~**`Validate()` antes de `err` de unmarshal**~~ — `processLoadedProp` valida solo tras decode OK.
3. ~~**`SIGKILL` en `signal.Notify`**~~ — eliminado de `Run`.

### Resueltos en v1.3.0 (Fase 3)

1. **`NewInstance[T]` en pilotos** — sustituido por `Once` / `OnceValue` en `rabbitmq-golang` y `groowcity-cron` (props + `TestSubscription`).
2. **`GetInstance("TestSubscription")`** — eliminado en esos pilotos.

### Resueltos en v1.4.0 (Fase 4)

1. **`Lifecycle` + `RunWithSignals`** en initModules.
2. **`groowcity-cron`:** Mongo `Disconnect`, Rabbit consumer cancel + `CloseChannel` en `Stop`.
3. **`Run()`** ya no llama `os.Exit`.

### Fase 5 — bootstrap por servicio

| Servicio | Bootstrap | main |
|----------|-----------|------|
| base-golang | `internal/bootstrap` (HTTP shutdown, DB, routes) | `bootstrap.Run` |
| groowcity-cron | `internal/bootstrap` | `bootstrap.Run` |
| rabbitmq-golang | `internal/bootstrap` | `bootstrap.Run` |

### Fase 6 — documentación y CI

- README, ARCHITECTURE.md, docs/MIGRATION.md, examples/standalone, `.github/workflows/ci.yml`, Makefile, `.golangci.yml`.

### Pendientes (Fase 7+)

1. **`GetDatabaseMongo` con `newInstance == nil`** (`groowcity-cron` / `database.go`) — panic si se llama antes que el factory.
2. **`base-golang`:** doble `RunLoadProperties` vía `Init(true, true)`.
3. **`base-golang/go.mod`:** falta `require` explícito de `initModules` y `authBase`.
4. **Dos patrones de config en rabbitmq:** `prop.GetQueueProp()` con `sync.Once` local vs `NewInstance` en otros servicios.
5. **Inconsistencia de keys:** `ReportDAO` vs `ReportRepository` (convención naming).
6. **Colisión potencial entre servicios:** misma key string en distintos binaries OK; en un mismo proceso deben ser únicas.

---

## 7. Orden sugerido de migración (Fase 5)

1. `rabbitmq-golang` (menor superficie)
2. `groowcity-cron`
3. `authBase` (quitar `GetInstance` cuando `base-golang` use DI)
4. `base-golang` (mayor grafo + HTTP)

---

*Generado en Fase 0. Actualizar al cambiar consumidores o releases.*
