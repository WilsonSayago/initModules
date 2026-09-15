package initModules

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type testLifecycle struct {
	started atomic.Int32
	stopped atomic.Int32
	block   chan struct{}
}

func newTestLifecycle() *testLifecycle {
	return &testLifecycle{block: make(chan struct{})}
}

func (t *testLifecycle) Start(ctx context.Context) error {
	t.started.Add(1)
	return nil
}

func (t *testLifecycle) Stop(ctx context.Context) error {
	t.stopped.Add(1)
	return nil
}

func TestRunContext_StartStopOnCancel(t *testing.T) {
	ResetApp()
	lc := newTestLifecycle()
	Register(lc)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunContext(ctx, RunOptions{RunLifecycles: true, StopTimeout: 5 * time.Second})
	}()

	deadline := time.Now().Add(2 * time.Second)
	for lc.started.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if lc.started.Load() != 1 {
		t.Fatal("expected Start to be called")
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("RunContext: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for RunContext")
	}

	if lc.stopped.Load() != 1 {
		t.Fatal("expected Stop to be called")
	}
}

func TestRunContext_StartFailureStopsPrevious(t *testing.T) {
	ResetApp()

	ok := newTestLifecycle()
	Register(ok)
	Register(&failingLifecycle{after: ok})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := RunContext(ctx, RunOptions{RunLifecycles: true})
	if err == nil {
		t.Fatal("expected start error")
	}
	if ok.stopped.Load() != 1 {
		t.Fatal("expected first lifecycle to be stopped after second failed")
	}
}

type failingLifecycle struct {
	after *testLifecycle
}

func (f *failingLifecycle) Start(ctx context.Context) error {
	return context.Canceled
}

func (f *failingLifecycle) Stop(ctx context.Context) error {
	return nil
}

func TestProcessAdapter_StartsLegacyProcess(t *testing.T) {
	ResetApp()
	var started atomic.Bool
	Register(ProcessAdapter{Process: legacyProcessFunc(func() { started.Store(true) })})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = RunContext(ctx, RunOptions{RunLifecycles: true})
	}()

	deadline := time.Now().Add(time.Second)
	for !started.Load() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !started.Load() {
		t.Fatal("legacy Start was not invoked")
	}
	cancel()
	time.Sleep(50 * time.Millisecond)
}

type legacyProcessFunc func()

func (f legacyProcessFunc) Start() { f() }
