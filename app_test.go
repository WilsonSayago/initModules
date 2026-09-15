package initModules

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type testLifecycle struct {
	started       atomic.Int32
	stopped       atomic.Int32
	startedSignal chan struct{}
	stoppedSignal chan struct{}
}

func newTestLifecycle() *testLifecycle {
	return &testLifecycle{
		startedSignal: make(chan struct{}),
		stoppedSignal: make(chan struct{}),
	}
}

func (t *testLifecycle) Start(ctx context.Context) error {
	t.started.Add(1)
	close(t.startedSignal)
	return nil
}

func (t *testLifecycle) Stop(ctx context.Context) error {
	t.stopped.Add(1)
	close(t.stoppedSignal)
	return nil
}

func TestRunContext_StartStopOnCancel(t *testing.T) {
	ResetApp()
	t.Cleanup(ResetApp)
	lc := newTestLifecycle()
	Register(lc)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunContext(ctx, RunOptions{RunLifecycles: true, StopTimeout: 5 * time.Second})
	}()

	waitForSignal(t, lc.startedSignal, "Start")
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
	t.Cleanup(ResetApp)

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
	t.Cleanup(ResetApp)

	started := make(chan struct{})
	Register(ProcessAdapter{Process: legacyProcessFunc(func() { close(started) })})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunContext(ctx, RunOptions{RunLifecycles: true})
	}()

	waitForSignal(t, started, "legacy Start")
	cancel()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("RunContext: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for RunContext")
	}
}

type legacyProcessFunc func()

func (f legacyProcessFunc) Start() { f() }

func waitForSignal(t *testing.T, signal <-chan struct{}, operation string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for %s", operation)
	}
}
