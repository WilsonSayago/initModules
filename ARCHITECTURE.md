# initModules — Architecture

`initModules` is a **bootstrap toolkit** for Go microservices. It is not a full DI framework: it provides small, composable primitives that each service wires in `internal/bootstrap`.

## What belongs in the library

| Concern | Package API | Notes |
|---------|-------------|--------|
| Config load | `LoadProperties`, `ConfigLoader`, `AddPropE` | YAML / `.properties`, `${ENV}` expansion |
| Validation hook | `Prop` interface | Called after successful decode |
| Singletons | `Once`, `OnceValue`, `OnceIn` | Prefer over string-key `GetInstance` |
| Lifecycle | `Lifecycle`, `Register`, `RunContext`, `RunWithSignals` | Ordered start / reverse stop |
| Legacy bridge | `IProcess`, `ProcessAdapter`, `RegisterProcess` | Deprecated path |

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
            ├── Register(lifecycle…)  → DB, queue, HTTP, …
            └── RunWithSignals        → block → graceful Stop
```

## Dependency direction

```text
  cmd  →  bootstrap  →  initModules
              ↓
         infra / core (service code must not import initModules from domain)
```

Domain packages should **not** import `initModules`. Only `cmd`, `bootstrap`, and infrastructure wiring layers may import it.

## Versioning

- **v1.x** — additive changes; deprecated APIs retained (`GetInstance(string)`, `SetFilePath`, `IProcess`).
- **v2** (planned) — remove deprecated globals and `log.Fatal` wrappers; see [docs/MIGRATION.md](docs/MIGRATION.md).

## Related docs

- [README.md](README.md) — usage and quickstart
- [ROADMAP.md](ROADMAP.md) — evolution plan
- [docs/INVENTORY.md](docs/INVENTORY.md) — consumer inventory
- [examples/standalone](examples/standalone) — minimal config-only app
