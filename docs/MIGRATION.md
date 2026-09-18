# Migration guide

## Module path (v2)

This repository’s root `go.mod` declares:

```text
module github.com/WilsonSayago/initModules/v2
```

Import as:

```go
import "github.com/WilsonSayago/initModules/v2"
```

The Go package name remains `initModules`. Publish with annotated tags
`v2.0.0`, `v2.0.1`, … (not `v1.x`). Consumers still on major 1 use
`github.com/WilsonSayago/initModules@v1.0.6` without the `/v2` suffix.

---

## Upgrading from v1.0.6 to v2

1. Change the require / import path to `github.com/WilsonSayago/initModules/v2`.
2. Prefer the additive APIs below (`LoadProperties`, `Once`, `NewApp`, …).
3. Deprecated v1 symbols remain for compatibility until a later breaking cleanup.

No breaking changes to symbol names if you keep using deprecated APIs.
Recommended incremental steps from the v1 eras:

1. **v1.1** — No code changes required; `GetInstance` is thread-safe.
2. **v1.2** — Switch to `AddPropE` + `LoadProperties` in `main` (handle `error`).
3. **v1.3** — Replace `NewInstance[T]().GetInstance(fn)` with `OnceValue(fn)` or `Once(fn)`.
4. **v1.4** — Implement `Lifecycle` for DB/queues; use `Register` + `RunWithSignals`.
5. **v1.5** — Add `internal/bootstrap` composition root (see `base-golang`).
6. **v1.6 (this branch)** — `PropValidator`, strict YAML/env, atomic load, `NewApp` isolation, hardened `OnceIn` / legacy adapters, Go 1.24 minimum.

### Config loading

```go
// Before
initModules.SetFilePath(initModules.YML, "config.yml")
initModules.AddProp(&cfg)
initModules.RunLoadProperties()

// After
if err := initModules.AddPropE(initModules.OnceValue(NewCfg)); err != nil { log.Fatal(err) }
if err := initModules.LoadProperties(
    initModules.WithFilePath("config.yml"),
    initModules.WithFormat(initModules.YML),
    initModules.WithStrictYAML(true),
    initModules.WithStrictEnv(true),
); err != nil { log.Fatal(err) }
```

Implement `PropValidator` (`Validate() error`) on new config structs. `Prop.Validate()` remains in v1 and is only used when the target does not implement `PropValidator`. Enable `WithStrictYAML(true)` and `WithStrictEnv(true)` for new services; the v1 defaults stay non-strict. Loads are atomic: failed decode or validation leaves previous target values unchanged, including constructor defaults for omitted fields. External side effects performed inside `Validate` are not reversible.

### Singletons

```go
// Before
initModules.GetInstance("UserRepo", func() interface{} {
    return &UserRepository{}
}).(*UserRepository)

// After
initModules.Once(func() *UserRepository {
    return &UserRepository{}
})
```

### Process lifecycle

```go
// Before
initModules.RegisterProcess(db)
initModules.Init(true, true)
r.Run() // HTTP outside initModules

// After
app := initModules.NewApp()
app.Register(dbLifecycle)
app.Register(httpServer)
if err := app.RunWithSignals(ctx, initModules.RunOptions{
    LoadProperties: false,
    RunLifecycles:  true,
}); err != nil { log.Fatal(err) }
```

Package-level `Register`, `RunContext`, and `RunWithSignals` remain for compatibility and operate on a shared default `App`. New services should use `NewApp`. Those globals are planned for removal in a later cleanup.

`OnceIn` / `OnceValueIn` require a non-nil `NewContainer()`. Passing `nil` panics with a message to use `NewContainer` or the global `Once` APIs; it no longer falls back to the process-wide registry. Keep `IProcess` / `RegisterProcess` only while migrating; new components should implement `Lifecycle`.

---

## Preparing for a later breaking cleanup

The **module path is already** `github.com/WilsonSayago/initModules/v2`.
A future release may still remove deprecated APIs:

| Removed | Replacement |
|---------|-------------|
| `GetInstance(string, …)` | `Once` / `OnceValue` / service `Container` |
| `SetFilePath`, `AddProp`, `RunLoadProperties` | `LoadProperties(opts…)` only |
| `Prop` (`Validate()`) | `PropValidator` (`Validate() error`) |
| `IProcess` + `RegisterProcess` | `Lifecycle` only |
| package-level `Register`, `RunContext`, `RunWithSignals` | `NewApp()` and instance methods |
| `log.Fatal` inside library | Always return `error` to `main` |

Before relying on a cleanup release:

- [ ] No `GetInstance("…")` in your module (grep your codebase).
- [ ] No `Init(true, true)` with duplicate property loading.
- [ ] All long-running components implement `Stop(ctx)`.

---

## Consumer reference implementations

| Pattern | Service |
|---------|---------|
| Config + DB + HTTP | `base-golang` |
| Config + Mongo + Rabbit | `groowcity-cron` |
| Config + Rabbit only | `rabbitmq-golang` |
