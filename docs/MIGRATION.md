# Migration guide

## Upgrading between v1.x releases (v1.0 → v1.6)

No breaking changes if you keep using deprecated APIs. Recommended incremental steps:

1. **v1.1** — No code changes required; `GetInstance` is thread-safe.
2. **v1.2** — Switch to `AddPropE` + `LoadProperties` in `main` (handle `error`).
3. **v1.3** — Replace `NewInstance[T]().GetInstance(fn)` with `OnceValue(fn)` or `Once(fn)`.
4. **v1.4** — Implement `Lifecycle` for DB/queues; use `Register` + `RunWithSignals`.
5. **v1.5** — Add `internal/bootstrap` composition root (see `base-golang`).

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

Implement `PropValidator` (`Validate() error`) on new config structs. `Prop.Validate()` remains in v1 and is only used when the target does not implement `PropValidator`. Enable `WithStrictYAML(true)` and `WithStrictEnv(true)` for new services; the v1 defaults stay non-strict. Loads are atomic: failed decode or validation leaves previous target values unchanged. External side effects performed inside `Validate` are not reversible.

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
initModules.Register(dbLifecycle)
initModules.Register(httpServer)
initModules.RunWithSignals(ctx, initModules.RunOptions{
    LoadProperties: false,
    RunLifecycles:  true,
})
```

---

## Preparing for v2.0 (planned)

The following will be **removed** in v2:

| Removed | Replacement |
|---------|-------------|
| `GetInstance(string, …)` | `Once` / `OnceValue` / service `Container` |
| `SetFilePath`, `AddProp`, `RunLoadProperties` | `LoadProperties(opts…)` only |
| `Prop` (`Validate()`) | `PropValidator` (`Validate() error`) |
| `IProcess` + `RegisterProcess` | `Lifecycle` only |
| `log.Fatal` inside library | Always return `error` to `main` |

Before upgrading to v2:

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
