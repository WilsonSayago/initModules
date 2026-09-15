package initModules

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/signal"
	"reflect"
	"syscall"
	"time"
)

const defaultStopTimeout = 30 * time.Second

// RunOptions configures RunContext and RunWithSignals.
type RunOptions struct {
	LoadProperties bool
	RunLifecycles  bool
	StopTimeout    time.Duration
}

var defaultApp App

// App coordinates lifecycle registration and graceful shutdown.
type App struct {
	lifecycles []Lifecycle
}

// Register adds a Lifecycle component to the global application registry.
func Register(l Lifecycle) {
	defaultApp.Register(l)
}

// RegisterLifecycle is an alias for Register.
func RegisterLifecycle(l Lifecycle) {
	Register(l)
}

func (a *App) Register(l Lifecycle) {
	if l == nil {
		return
	}
	a.lifecycles = append(a.lifecycles, l)
}

// RunContext loads optional properties, starts registered lifecycles, blocks until ctx is done, then stops in reverse order.
func RunContext(ctx context.Context, opts RunOptions) error {
	if opts.StopTimeout <= 0 {
		opts.StopTimeout = defaultStopTimeout
	}

	if opts.LoadProperties {
		if err := LoadProperties(); err != nil {
			return fmt.Errorf("load properties: %w", err)
		}
	}

	if !opts.RunLifecycles {
		<-ctx.Done()
		return ctx.Err()
	}

	return defaultApp.run(ctx, opts.StopTimeout)
}

func (a *App) run(ctx context.Context, stopTimeout time.Duration) error {
	started := make([]Lifecycle, 0, len(a.lifecycles))

	for _, lc := range a.lifecycles {
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
	if lc == nil {
		return "nil"
	}
	t := reflect.TypeOf(lc)
	if t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	}
	return t.Name()
}

// RunWithSignals is a helper that wraps ctx with OS shutdown signals and calls RunContext.
// Supported signals: SIGHUP, SIGINT, SIGTERM, SIGQUIT (SIGKILL is not catchable).
func RunWithSignals(ctx context.Context, opts RunOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	err := RunContext(ctx, opts)
	if err == context.Canceled {
		return nil
	}
	return err
}

// ResetApp clears registered lifecycles (for tests).
func ResetApp() {
	defaultApp = App{}
}
