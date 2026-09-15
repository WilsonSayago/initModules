# initModules

[![Go](https://img.shields.io/badge/Go-1.26.3+-00ADD8?logo=go&logoColor=white)](https://go.dev/dl/)
[![CI](https://github.com/WilsonSayago/initModules/actions/workflows/ci.yml/badge.svg)](https://github.com/WilsonSayago/initModules/actions/workflows/ci.yml)

Shared **bootstrap toolkit** for Go microservices: configuration loading, type-safe singletons, and graceful lifecycle (start/stop with context).

> **Go:** requires **1.26.3+** (see `go.mod`).

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

    initModules.Register(myService)

    if err := initModules.RunWithSignals(context.Background(), initModules.RunOptions{
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

```sh
go get github.com/WilsonSayago/initModules@v1.6.0
```

In a **go.work** monorepo, add `use ./initModules` and depend on the local module path.

## Features

- YAML / `.properties` config with `${ENV}` expansion
- `PropValidator` after successful decode (`Prop` remains for v1 compatibility)
- Optional strict YAML (`WithStrictYAML`) and strict env (`WithStrictEnv`)
- Atomic config load: targets are not mutated unless decode and validation succeed for all of them
- `Once` / `OnceValue` / `Container` singletons (thread-safe)
- `Lifecycle` with ordered `Start` / `Stop` and signal-aware `RunWithSignals`
- Legacy compatibility: `IProcess`, `GetInstance(string)` (deprecated)

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

v1 defaults keep compatibility: env expansion uses `os.ExpandEnv` (`$NAME` and `${NAME}`), unknown YAML keys are ignored, and `Prop.Validate()` has no error return. New services should implement `PropValidator` (`Validate() error`) and enable `WithStrictYAML(true)` plus `WithStrictEnv(true)`. Loading is atomic: if any target fails to decode or validate, none of the registered structs are updated. Side effects inside a consumer `Validate` are not rolled back.

Strict YAML accepts a single target; group sections in one root struct. Strict env expands only `${NAME}`, errors when `NAME` is unset (empty but set is allowed), and treats `$$` as a literal `$`. Error messages include the variable name, never the value.

### Config + database

Register a `Lifecycle` that pings on start and closes the pool on stop.

```go
initModules.Register(lifecycleFunc{
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
| Singleton | `OnceValue`, `Once`, `OnceIn` |
| Graceful run | `Register`, `RunWithSignals`, `RunContext` |
| Legacy | `Run`, `RegisterProcess`, `GetInstance` (deprecated) |

## Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md) — library vs microservice responsibilities
- [docs/MIGRATION.md](docs/MIGRATION.md) — v1.x upgrades and v2 preview
- [ROADMAP.md](ROADMAP.md) — evolution plan
- [CHANGELOG.md](CHANGELOG.md) — release notes
- [docs/INVENTORY.md](docs/INVENTORY.md) — known consumers

## Development

```sh
make fmt-check  # verify gofmt
make test       # repeat unit tests 20 times
make test-race  # race detector + atomic coverage
make vet        # go vet ./...
make example    # test the standalone nested module
make lint       # golangci-lint (install: https://golangci-lint.run/welcome/install/)
make ci         # run all local CI gates
```

CI runs the same formatting, repeated test, race/coverage, vet, nested-module, and lint gates on every push/PR.

## Migration

Deprecated APIs remain in v1.x for compatibility. New services should use `LoadProperties`, `Once`/`OnceValue`, `Lifecycle`, and a local `internal/bootstrap` package.

See [docs/MIGRATION.md](docs/MIGRATION.md) for step-by-step upgrades and the planned v2 breaking changes.

## License

See repository license (if applicable).

## Contributing

Issues and PRs welcome. Run `make ci` before submitting.
