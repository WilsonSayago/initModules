# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

Proposed first public **v2** tag: **v2.0.0** (not tagged yet). Module path is
`github.com/WilsonSayago/initModules/v2` per Go’s major-version suffix rule.
Previous major remains `github.com/WilsonSayago/initModules@v1.0.6`.

Compared with `v1.0.6` using `golang.org/x/exp/cmd/apidiff@v0.0.0-20260908205506-85c1c2202aba`
(before the module-path change): compatible API additions only; no removed or
signature-incompatible exports. The `/v2` module path itself is the SemVer major
bump for consumers.

### Added

- `PropValidator` with `Validate() error`; `Prop` remains for v1 compatibility.
- `WithStrictYAML` — opt-in unknown-field rejection for a single YAML target.
- `WithStrictEnv` — opt-in `${NAME}` expansion that errors on unset variables and maps `$$` to `$`.
- atomic config load: decode and validate independent copies of current values, then commit only if every target succeeds. Omitted fields keep constructor defaults; pre-existing maps, slices, and pointers are not mutated on failure.
- `LoadProperties`, `AddPropE`, `RunLoadPropertiesE`, `ConfigLoader`, and options `WithFilePath`, `WithFormat`, `WithExpandEnv`.
- `Once` / `OnceValue` / `Container` (`NewContainer`, `OnceIn`, `OnceValueIn`).
- `Lifecycle`, `ProcessAdapter`, `Register` / `RunContext` / `RunWithSignals` / `RunOptions`.
- `NewApp`, `(*App).RunContext`, `(*App).RunWithSignals`, and `ErrAppRunning` for isolated, concurrent-safe app runs.
- `ResetApp` for tests (must not run during an active global run).
- Dependabot for Go modules and GitHub Actions (weekly, grouped; prereleases stay excluded).
- `examples/standalone`, CI workflow, Makefile gates, and architecture/migration docs.

### Changed

- Module path is `github.com/WilsonSayago/initModules/v2` (Go convention for major version ≥ 2). Import `github.com/WilsonSayago/initModules/v2`; package name remains `initModules`. Publish tags as `v2.x.y`.
- Go language minimum is 1.24.0 with recommended toolchain go1.27.1 (verified 2026-09-16 on go.dev: go1.24.13 and go1.27.1).
- `github.com/magiconair/properties` v1.18.11 (official consecutive release after v1.8.10; `Decode` API unchanged).
- YAML remains `gopkg.in/yaml.v3` v3.0.1; yaml v4 is still release-candidate only.
- CI tests Go 1.24.13 and 1.27.1 with `GOTOOLCHAIN=local`; golangci-lint v2.13.2 and govulncheck v1.8.0 run on current stable.
- GitHub Actions are pinned by full commit SHA (`checkout` v7.0.1, `setup-go` v7.0.0, `upload-artifact` v7.0.1, `golangci-lint-action` v9.3.0).
- Invalid options, empty target lists, typed-nil targets, and unsupported formats are rejected before reading the file.
- Shutdown-unrelated config errors wrap the operation and filename without embedding env values or file contents.
- Package-level `Register`, `RunContext`, and `RunWithSignals` delegate to a shared default `App`; registrations during a run apply only to the next run.
- `OnceIn` / `OnceValueIn` panic on a nil `Container` instead of falling back to the global registry. Nil constructors and nil constructed instances also panic with an actionable message. A failed `Once` initialization is remembered so later calls keep that message instead of a type assertion on nil.
- Legacy `IProcess` adapters skip typed-nil values, return an error from `ProcessAdapter.Start` when the process is nil, and name value or pointer implementations without panicking.
- `GetInstance` is safe for concurrent use (`sync.Map` + `sync.Once` per key).
- `Validate()` on config targets runs only after a successful decode.
- `Run` uses `RunWithSignals` and no longer calls `os.Exit` (it still `log.Fatal`s on error for v1 compatibility).
- `RegisterProcess` also registers a `ProcessAdapter` on the global app.
- `SIGKILL` is not registered with `signal.Notify` (not catchable).
- Repository layout: legacy wrappers live in `legacy_run.go`, config tests are split by domain, and design docs live under `docs/`.

### Deprecated

- `Prop` (`Validate()` with no error) — prefer `PropValidator`.
- `SetFilePath`, `AddProp`, `RunLoadProperties` — prefer `LoadProperties` + `AddPropE`.
- `GetInstance(string, ...)`, `NewInstance`, `BaseInstance` — prefer `Once` / `OnceValue` / `Container`.
- `Init` / `RunProcesses` fire-and-forget model — prefer `NewApp` + `Lifecycle`.

### Fixed

- `AddProp` error message states that a pointer to struct is required.

## [1.0.6]

Last published git tag. Initial shared bootstrap: config load, `GetInstance`,
`BaseInstance`, `IProcess`, `Init` / `Run`.
