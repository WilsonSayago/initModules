package initModules

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const defaultStopTimeout = 30 * time.Second

// ErrAppRunning is returned when RunContext or RunWithSignals is called on an
// App that already has an active run.
var ErrAppRunning = errors.New("app is already running")

// RunOptions configures RunContext and RunWithSignals.
type RunOptions struct {
	LoadProperties bool
	RunLifecycles  bool
	StopTimeout    time.Duration
}

var defaultApp = NewApp()

// App coordinates lifecycle registration and graceful shutdown for one isolated
// process unit. It is a lightweight composition root, not a DI container.
// App must not be copied after first use.
type App struct {
	mu         sync.Mutex
	lifecycles []Lifecycle
	running    bool
}

// NewApp creates an empty App. Lifecycles registered on this instance are
// isolated from the package-level default App used by Register and RunContext.
func NewApp() *App {
	return &App{}
}

// Register adds a Lifecycle component to the global application registry.
func Register(l Lifecycle) {
	defaultApp.Register(l)
}

// RegisterLifecycle is an alias for Register.
func RegisterLifecycle(l Lifecycle) {
	Register(l)
}

// Register adds a Lifecycle. It is safe for concurrent use. Nil values,
// including typed-nil pointers stored in a Lifecycle interface, are ignored.
// Registrations that happen while a run is in progress are not started until
// the next run.
func (a *App) Register(l Lifecycle) {
	if isNilLifecycle(l) {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lifecycles = append(a.lifecycles, l)
}

// RunContext loads optional properties, starts registered lifecycles, blocks until ctx is done, then stops in reverse order.
func RunContext(ctx context.Context, opts RunOptions) error {
	return defaultApp.RunContext(ctx, opts)
}

// RunContext starts this App's registered lifecycles. A nil ctx is treated as
// context.Background(). Concurrent runs on the same instance return ErrAppRunning.
func (a *App) RunContext(ctx context.Context, opts RunOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.StopTimeout <= 0 {
		opts.StopTimeout = defaultStopTimeout
	}

	lifecycles, err := a.beginRun()
	if err != nil {
		return err
	}
	defer a.endRun()

	if opts.LoadProperties {
		if err := LoadProperties(); err != nil {
			return fmt.Errorf("load properties: %w", err)
		}
	}

	if !opts.RunLifecycles {
		<-ctx.Done()
		return ctx.Err()
	}

	return a.run(ctx, opts.StopTimeout, lifecycles)
}

func (a *App) beginRun() ([]Lifecycle, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.running {
		return nil, ErrAppRunning
	}
	a.running = true
	snap := make([]Lifecycle, len(a.lifecycles))
	copy(snap, a.lifecycles)
	return snap, nil
}

func (a *App) endRun() {
	a.mu.Lock()
	a.running = false
	a.mu.Unlock()
}

func (a *App) run(ctx context.Context, stopTimeout time.Duration, lifecycles []Lifecycle) error {
	started := make([]Lifecycle, 0, len(lifecycles))

	for _, lc := range lifecycles {
		log.Println("Starting:", lifecycleName(lc))
		if err := lc.Start(ctx); err != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), stopTimeout)
			stopErr := a.stopAll(stopCtx, started)
			cancel()
			startErr := fmt.Errorf("start %s: %w", lifecycleName(lc), err)
			return errors.Join(startErr, stopErr)
		}
		started = append(started, lc)
	}

	<-ctx.Done()

	stopCtx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()

	stopErr := a.stopAll(stopCtx, started)
	if errors.Is(ctx.Err(), context.Canceled) {
		return stopErr
	}
	return errors.Join(ctx.Err(), stopErr)
}

func (a *App) stopAll(ctx context.Context, started []Lifecycle) error {
	var errs []error
	for i := len(started) - 1; i >= 0; i-- {
		lc := started[i]
		log.Println("Stopping:", lifecycleName(lc))
		if err := lc.Stop(ctx); err != nil {
			log.Printf("Stop %s: %v", lifecycleName(lc), err)
			errs = append(errs, fmt.Errorf("stop %s: %w", lifecycleName(lc), err))
		}
	}
	return errors.Join(errs...)
}

func lifecycleName(lc Lifecycle) string {
	return typeName(lc)
}

func isNilLifecycle(l Lifecycle) bool {
	return isNilValue(l)
}

// RunWithSignals is a helper that wraps ctx with OS shutdown signals and calls RunContext.
// Supported signals: SIGHUP, SIGINT, SIGTERM, SIGQUIT (SIGKILL is not catchable).
func RunWithSignals(ctx context.Context, opts RunOptions) error {
	return defaultApp.RunWithSignals(ctx, opts)
}

// RunWithSignals wraps ctx with OS shutdown signals and calls RunContext on this App.
// A nil ctx is treated as context.Background().
func (a *App) RunWithSignals(ctx context.Context, opts RunOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	err := a.RunContext(ctx, opts)
	if err == context.Canceled {
		return nil
	}
	return err
}

// ResetApp clears registered lifecycles on the default App. It is intended for
// tests and must not be called while a global run is in progress.
func ResetApp() {
	defaultApp.mu.Lock()
	defer defaultApp.mu.Unlock()
	defaultApp.lifecycles = nil
	defaultApp.running = false
}
