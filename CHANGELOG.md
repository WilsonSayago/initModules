# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `PropValidator` with `Validate() error`; `Prop` remains for v1 compatibility.
- `WithStrictYAML` — opt-in unknown-field rejection for a single YAML target.
- `WithStrictEnv` — opt-in `${NAME}` expansion that errors on unset variables and maps `$$` to `$`.
- atomic config load: decode and validate independent copies of current values, then commit only if every target succeeds. Omitted fields keep constructor defaults; pre-existing maps, slices, and pointers are not mutated on failure.
- `NewApp`, `(*App).RunContext`, `(*App).RunWithSignals`, and `ErrAppRunning` for isolated, concurrent-safe app runs.
- Dependabot for Go modules and GitHub Actions (weekly, grouped; prereleases stay excluded).

### Changed

- Go language minimum is 1.24.0 with recommended toolchain go1.27.1 (verified 2026-09-16 on go.dev: go1.24.13 and go1.27.1).
- `github.com/magiconair/properties` v1.18.11 (official consecutive release after v1.8.10; `Decode` API unchanged).
- YAML remains `gopkg.in/yaml.v3` v3.0.1; yaml v4 is still release-candidate only.
- CI tests Go 1.24.13 and 1.27.1; golangci-lint v2.13.2 and govulncheck v1.8.0 run on current stable.
- GitHub Actions are pinned by full commit SHA (`checkout` v7.0.1, `setup-go` v7.0.0, `upload-artifact` v7.0.1, `golangci-lint-action` v9.3.0).
- Invalid options, empty target lists, typed-nil targets, and unsupported formats are rejected before reading the file.
- Shutdown-unrelated config errors wrap the operation and filename without embedding env values or file contents.
- Package-level `Register`, `RunContext`, and `RunWithSignals` delegate to a shared default `App`; registrations during a run apply only to the next run.
- `OnceIn` / `OnceValueIn` panic on a nil `Container` instead of falling back to the global registry. Nil constructors and nil constructed instances also panic with an actionable message. A failed `Once` initialization is remembered so later calls keep that message instead of a type assertion on nil.
- Legacy `IProcess` adapters skip typed-nil values, return an error from `ProcessAdapter.Start` when the process is nil, and name value or pointer implementations without panicking.

### Deprecated

- `Prop` (`Validate()` with no error) — prefer `PropValidator`.

## [1.6.0] - 2026-05-26

### Added

- [ARCHITECTURE.md](ARCHITECTURE.md) — library vs microservice boundaries.
- [docs/MIGRATION.md](docs/MIGRATION.md) — v1.x upgrade path and v2 preview.
- [examples/standalone](examples/standalone) — minimal config-only sample.
- GitHub Actions CI: `go vet`, `go test -race -cover`, `golangci-lint`.
- `Makefile` with `test`, `vet`, `lint`, `ci` targets.
- `.golangci.yml` linter configuration.

### Changed

- README rewritten: quickstart, recipes (config / DB / HTTP / queue), Go version badge, dev commands.

## [1.5.0] - 2026-05-26

### Changed (consumidores — Fase 5, sin API nueva en la lib)

- **base-golang:** `internal/bootstrap` composition root; Gin + `http.Server.Shutdown`; sin `Init(true,true)` ni doble load props.
- **groowcity-cron** / **rabbitmq-golang:** `internal/bootstrap` + `cmd/main` delgado.

## [1.4.0] - 2026-05-26

### Added

- `Lifecycle` interface (`Start`/`Stop` with `context.Context`).
- `ProcessAdapter` for legacy `IProcess`.
- `Register` / `RunContext` / `RunWithSignals` / `RunOptions` — ordered startup and graceful shutdown (default stop timeout 30s).
- `ResetApp()` for tests.
- `messaging/rabbitmq`: `RunConsumer(ctx, handler)` for cancellable consumers.

### Changed

- `Run()` uses `RunWithSignals` and no longer calls `os.Exit`.
- `RegisterProcess` also registers a `ProcessAdapter` on the global app.
- **Pilot `groowcity-cron`:** Mongo `Disconnect` and Rabbit consumer cancel/`CloseChannel` on stop; uses `Register` + `RunWithSignals`.
- **Pilot `rabbitmq-golang`:** uses `RunWithSignals` (legacy `IProcess` via adapter).

### Deprecated

- `Init` / `RunProcesses` fire-and-forget model — prefer `Register` + `RunContext`.

## [1.3.0] - 2026-05-26

### Added

- `Once[T](func() *T) *T` and `OnceValue[T](func() T) *T` — type-safe global singletons.
- `Container` with `OnceIn` / `OnceValueIn` — isolated registry per service/test.
- Tests under `-race` for value and pointer `T`, and container isolation.

### Changed

- `BaseInstance[T]` / `NewInstance[T]` now delegate to `Once` (deprecated).
- `GetInstance(string, ...)` marked deprecated.
- **Pilots:** `rabbitmq-golang` and `groowcity-cron` (props + subscriptions) migrated from string keys / `NewInstance` to `Once`.

### Deprecated

- `GetInstance(string, ...)`, `NewInstance[T]`, `BaseInstance[T]`.

## [1.2.0] - 2026-05-26

### Added

- `LoadProperties(opts ...Option) error` — load config without `log.Fatal`.
- `AddPropE(p any) error` — register property targets with error return.
- `RunLoadPropertiesE() error` — alias for `LoadProperties()`.
- `ConfigLoader` — isolated loader with `AddProp` / `Load` (no global registry).
- Options: `WithFilePath`, `WithFormat`, `WithExpandEnv`.
- `testdata/` fixtures and table-driven tests (YAML, `.properties`, `${ENV}` expansion).

### Deprecated

- `SetFilePath`, `AddProp`, `RunLoadProperties` — use `LoadProperties` + `AddPropE` (legacy wrappers retained).

### Changed

- **Pilot:** `rabbitmq-golang` migrated to `AddPropE` + `LoadProperties` + `Run(false, true)`.

## [1.1.0] - 2026-05-26

### Fixed

- **Thread-safety:** `GetInstance` now uses `sync.Map` and `sync.Once` per key; safe under concurrent `RunProcesses` and parallel getters.
- **Properties:** `Validate()` runs only after a successful unmarshal/decode (YAML or `.properties`).
- **Signals:** removed `SIGKILL` from `signal.Notify` in `Run` (not catchable on Unix).
- **DX:** `AddProp` error message now states that a pointer to struct is required.

### Added

- Unit tests for concurrent singleton, property load order, and `AddProp` validation.
- Godoc on `Run` listing supported shutdown signals.

## [1.0.6] - (previous releases)

- Initial shared bootstrap: config load, `GetInstance`, `BaseInstance`, `IProcess`, `Init`/`Run`.
