# initModules — Architecture

`initModules` is a **bootstrap toolkit** for Go microservices. It is not a full DI framework: it provides small, composable primitives that each service wires in `internal/bootstrap`.

## What belongs in the library

| Concern | Package API | Notes |
|---------|-------------|--------|
| Config load | `LoadProperties`, `ConfigLoader`, `AddPropE` | YAML / `.properties`, `${ENV}` expansion |
| Validation hook | `PropValidator` (`Prop` remains in v1) | Called after successful decode |
| Singletons | `Once`, `OnceValue`, `OnceIn` | Prefer over string-key `GetInstance` |
| Lifecycle | `App`, `NewApp`, `Lifecycle`, `(*App).RunContext`, `(*App).RunWithSignals` | Ordered start / reverse stop; isolated per App |
| Legacy bridge | package-level `Register` / `RunContext`, `IProcess`, `ProcessAdapter` | Shared default App; deprecated path |

## What belongs in each microservice

| Concern | Location | Example in this monorepo |
|---------|----------|---------------------------|
| Composition root | `internal/bootstrap/` | `base-golang/internal/bootstrap` |
| Config structs | `internal/.../prop` or `properties` | `DatabaseProperty`, `QueueProp` |
| Domain & use cases | `internal/core` | Unchanged hexagonal rules |
| Adapters | `internal/infra` | HTTP, DB, messaging |
| Thin entrypoint | `cmd/main.go` | `bootstrap.Run(ctx)` |

## Typical startup flow

```text
cmd/main.go
    └── bootstrap.Run(ctx)
            ├── LoadConfig()          → initModules.LoadProperties
            ├── migrations / setup    → service-specific
            ├── app := NewApp()
            ├── app.Register(lifecycle…)  → DB, queue, HTTP, …
            └── app.RunWithSignals        → block → graceful Stop
```

## Dependency direction

```text
  cmd  →  bootstrap  →  initModules
              ↓
         infra / core (service code must not import initModules from domain)
```

Domain packages should **not** import `initModules`. Only `cmd`, `bootstrap`, and infrastructure wiring layers may import it.

`App` is a **lightweight composition root**: it stores an ordered list of `Lifecycle` values and runs start/stop. It does not construct or look up dependencies; each service still wires databases, clients, and config in `internal/bootstrap` and then registers the resulting components.

## Versioning

- **v1.x** — additive changes; deprecated APIs retained (`GetInstance(string)`, `SetFilePath`, `IProcess`). Last published tag: `v1.0.6`. Proposed next tag: `v1.6.0`.
- **v2** (planned) — module path `github.com/WilsonSayago/initModules/v2`; remove deprecated globals and `log.Fatal` wrappers; see [MIGRATION.md](MIGRATION.md).

## Related docs

- [README.md](../README.md) — usage and quickstart
- [ROADMAP.md](ROADMAP.md) — evolution plan
- [history/inventory-2026-05-26.md](history/inventory-2026-05-26.md) — historical consumer inventory (2026-05-26 snapshot)
- [examples/standalone](../examples/standalone) — minimal config-only app
