# initModules

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://go.dev/dl/)
[![CI](https://github.com/WilsonSayago/initModules/actions/workflows/ci.yml/badge.svg)](https://github.com/WilsonSayago/initModules/actions/workflows/ci.yml)

Shared **bootstrap toolkit** for Go microservices: configuration loading, type-safe singletons, and graceful lifecycle (start/stop with context).

> **Go:** language minimum **1.24.0**; recommended toolchain **go1.27.1** (see `go.mod`). CI tests the latest Go 1.24 patch and current stable.

## Quickstart

```go
package main

import (
    "context"
    "log"

    "github.com/WilsonSayago/initModules"
)

func main() {
    cfg := initModules.OnceValue(NewAppConfig)

    if err := initModules.AddPropE(cfg); err != nil {
        log.Fatal(err)
    }
    if err := initModules.LoadProperties(
        initModules.WithFilePath("internal/resources/properties.yml"),
        initModules.WithFormat(initModules.YML),
        initModules.WithStrictYAML(true),
        initModules.WithStrictEnv(true),
    ); err != nil {
        log.Fatal(err)
    }

    app := initModules.NewApp()
    app.Register(myService)

    if err := app.RunWithSignals(context.Background(), initModules.RunOptions{
        LoadProperties: false,
        RunLifecycles:  true,
    }); err != nil {
        log.Fatal(err)
    }
}
```

Run the minimal example:

```sh
cd examples/standalone && go run .
```

## Installation

The last published git tag is **v1.0.6**. This branch is preparing **v1.6.0**
(not tagged yet).

```sh
go get github.com/WilsonSayago/initModules@v1.0.6
```

In a **go.work** monorepo, add `use ./initModules` and depend on the local module path.

## Features

- YAML / `.properties` config with `${ENV}` expansion
- `PropValidator` after successful decode (`Prop` remains for v1 compatibility)
- Optional strict YAML (`WithStrictYAML`) and strict env (`WithStrictEnv`)
- Atomic config load: targets are not mutated unless decode and validation succeed for all of them
- `Once` / `OnceValue` / `Container` singletons (thread-safe)
- Isolated `App` composition root (`NewApp`, `(*App).RunContext`, `(*App).RunWithSignals`)
- `Lifecycle` with ordered `Start` / `Stop` and signal-aware `RunWithSignals`
- Isolated singletons via `NewContainer()` + `OnceIn` / `OnceValueIn` (a nil Container panics; use `Once` for the global registry)
- Legacy compatibility: package-level `Register` / `RunContext`, `IProcess`, `GetInstance(string)` (deprecated; prefer `Lifecycle`)

## Recipes by use case

### Config only

Load and validate properties — no background processes.

See [examples/standalone](examples/standalone).

```go
if err := initModules.LoadProperties(
    initModules.WithFilePath("config.yml"),
    initModules.WithFormat(initModules.YML),
    initModules.WithStrictYAML(true),
    initModules.WithStrictEnv(true),
); err != nil {
    log.Fatal(err)
}
```

v1 defaults keep compatibility: env expansion uses `os.ExpandEnv` (`$NAME` and `${NAME}`), unknown YAML keys are ignored, and `Prop.Validate()` has no error return. New services should implement `PropValidator` (`Validate() error`) and enable `WithStrictYAML(true)` plus `WithStrictEnv(true)`. Loading is atomic: decode and validation run on independent copies, so if any target fails none of the registered structs are updated, and omitted fields keep constructor defaults. Side effects inside a consumer `Validate` are not rolled back.

Strict YAML accepts a single target; group sections in one root struct. Strict env expands only `${NAME}`, errors when `NAME` is unset (empty but set is allowed), and treats `$$` as a literal `$`. Error messages include the variable name, never the value.

### Config + database

Register a `Lifecycle` that pings on start and closes the pool on stop.

```go
app := initModules.NewApp()
app.Register(lifecycleFunc{
    start: func(ctx context.Context) error { return db.Ping(ctx) },
    stop:  func(ctx context.Context) error { db.ClosePool(); return nil },
})
```

Reference: `groowcity-cron` (Mongo), `base-golang` (Postgres).

### Config + database + HTTP

Register DB lifecycles, route registration, then an `http.Server` with `Shutdown` on stop.

Reference: [`base-golang/internal/bootstrap`](https://github.com/WilsonSayago/initModules) (consumer in monorepo).

Recommended layout:

```text
cmd/main.go              → bootstrap.Run(ctx)
internal/bootstrap/      → composition root
internal/infra/          → adapters implementing Lifecycle
```

### Config + message consumer

Register a queue `Lifecycle` that cancels context and closes the client on stop.

Reference: `groowcity-cron`, `rabbitmq-golang`.

## API overview

| Task | API |
|------|-----|
| Load config | `AddPropE`, `LoadProperties`, `NewConfigLoader`, `PropValidator` |
| Singleton | `OnceValue`, `Once`, `NewContainer`, `OnceIn` |
| Graceful run | `NewApp`, `(*App).Register`, `(*App).RunWithSignals`, `(*App).RunContext` |
| Legacy | package-level `Register` / `RunContext`, `Run`, `RegisterProcess`, `GetInstance` (deprecated) |

## Documentation

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — library vs microservice responsibilities
- [docs/MIGRATION.md](docs/MIGRATION.md) — v1.x upgrades and v2 preview
- [docs/ROADMAP.md](docs/ROADMAP.md) — evolution plan
- [CHANGELOG.md](CHANGELOG.md) — release notes
- [docs/history/inventory-2026-05-26.md](docs/history/inventory-2026-05-26.md) — historical consumer inventory (2026-05-26 snapshot)
- [docs/RELEASE.md](docs/RELEASE.md) — publication checklist (no tag until authorized)

## Development

```sh
make fmt-check    # verify gofmt
make test         # repeat unit tests 20 times
make test-race    # race detector + atomic coverage
make vet          # go vet ./...
make example      # test the standalone nested module
make lint-verify  # golangci-lint v2.13.2 config schema
make lint         # golangci-lint v2.13.2
make vuln         # govulncheck v1.8.0
make ci           # run all local CI gates
```

CI tests Go 1.24.13 and 1.27.1, then runs the nested example, golangci-lint v2.13.2, and govulncheck v1.8.0 on current stable. Workflow actions are pinned by commit SHA. Dependabot updates Go modules and GitHub Actions weekly; prereleases stay excluded.

## Migration

Deprecated APIs remain in v1.x for compatibility. New services should use `LoadProperties`, `Once`/`OnceValue`, `NewApp` with `Lifecycle`, and a local `internal/bootstrap` package.

See [docs/MIGRATION.md](docs/MIGRATION.md) for step-by-step upgrades and the planned v2 breaking changes.

Package-level `Register`, `RunContext`, and `RunWithSignals` still target a shared default `App` in v1. Prefer `NewApp()` so tests and processes can isolate lifecycles. `ResetApp` is for tests only and must not run during an active global run.

## License

License is pending an explicit owner choice. A `LICENSE` file will be added
before any public tag.

## Contributing

Issues and PRs welcome. Run `make ci` before submitting.
