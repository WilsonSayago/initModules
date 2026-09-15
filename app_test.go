package initModules

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
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

type operationLog struct {
	mu         sync.Mutex
	operations []string
}

func (l *operationLog) add(operation string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.operations = append(l.operations, operation)
}

func (l *operationLog) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.operations...)
}

type scriptedLifecycle struct {
	name               string
	log                *operationLog
	startErr           error
	stopErr            error
	waitForStopContext bool
	started            atomic.Int32
	stopped            atomic.Int32
}

func (l *scriptedLifecycle) Start(context.Context) error {
	l.started.Add(1)
	if l.log != nil {
		l.log.add("start " + l.name)
	}
	return l.startErr
}

func (l *scriptedLifecycle) Stop(ctx context.Context) error {
	l.stopped.Add(1)
	if l.log != nil {
		l.log.add("stop " + l.name)
	}
	if l.waitForStopContext {
		<-ctx.Done()
		return ctx.Err()
	}
	return l.stopErr
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
		if err != nil {
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
	Register(&failingLifecycle{})

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
}

func (f *failingLifecycle) Start(ctx context.Context) error {
	return context.Canceled
}

func (f *failingLifecycle) Stop(ctx context.Context) error {
	return nil
}

func TestStopAll_CollectsErrorsInReverseOrder(t *testing.T) {
	var operations operationLog
	stopAErr := errors.New("stop A")
	stopCErr := errors.New("stop C")
	a := &scriptedLifecycle{name: "A", log: &operations, stopErr: stopAErr}
	b := &scriptedLifecycle{name: "B", log: &operations}
	c := &scriptedLifecycle{name: "C", log: &operations, stopErr: stopCErr}

	err := (&App{}).stopAll(context.Background(), []Lifecycle{a, b, c})

	if !errors.Is(err, stopAErr) || !errors.Is(err, stopCErr) {
		t.Fatalf("stopAll error = %v, want both stop errors", err)
	}
	if !strings.Contains(err.Error(), "stop scriptedLifecycle") {
		t.Fatalf("stopAll error lacks lifecycle context: %v", err)
	}
	wantOrder := []string{"stop C", "stop B", "stop A"}
	if got := operations.snapshot(); !slices.Equal(got, wantOrder) {
		t.Fatalf("stop order = %v, want %v", got, wantOrder)
	}
	if a.stopped.Load() != 1 || b.stopped.Load() != 1 || c.stopped.Load() != 1 {
		t.Fatalf("stop calls = A:%d B:%d C:%d, want one each", a.stopped.Load(), b.stopped.Load(), c.stopped.Load())
	}
}

func TestRunContext_StartFailureJoinsRollbackErrors(t *testing.T) {
	ResetApp()
	t.Cleanup(ResetApp)

	var operations operationLog
	startErr := errors.New("start C")
	stopAErr := errors.New("stop A")
	stopBErr := errors.New("stop B")
	a := &scriptedLifecycle{name: "A", log: &operations, stopErr: stopAErr}
	b := &scriptedLifecycle{name: "B", log: &operations, stopErr: stopBErr}
	c := &scriptedLifecycle{name: "C", log: &operations, startErr: startErr}
	Register(a)
	Register(b)
	Register(c)

	err := RunContext(context.Background(), RunOptions{RunLifecycles: true, StopTimeout: time.Second})

	for _, wantErr := range []error{startErr, stopAErr, stopBErr} {
		if !errors.Is(err, wantErr) {
			t.Errorf("RunContext error = %v, want errors.Is(_, %v)", err, wantErr)
		}
	}
	if a.stopped.Load() != 1 || b.stopped.Load() != 1 {
		t.Fatalf("rollback stop calls = A:%d B:%d, want one each", a.stopped.Load(), b.stopped.Load())
	}
	wantOrder := []string{"start A", "start B", "start C", "stop B", "stop A"}
	if got := operations.snapshot(); !slices.Equal(got, wantOrder) {
		t.Fatalf("operation order = %v, want %v", got, wantOrder)
	}
}

func TestRunContext_CancelWithStopError(t *testing.T) {
	ResetApp()
	t.Cleanup(ResetApp)

	stopErr := errors.New("stop failed")
	Register(&scriptedLifecycle{name: "A", stopErr: stopErr})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := RunContext(ctx, RunOptions{RunLifecycles: true, StopTimeout: time.Second})

	if !errors.Is(err, stopErr) {
		t.Fatalf("RunContext error = %v, want stop error", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("RunContext error = %v, cancellation must not mask stop failure", err)
	}
}

func TestRunContext_DeadlineExceededCleanStop(t *testing.T) {
	ResetApp()
	t.Cleanup(ResetApp)

	lifecycle := &scriptedLifecycle{name: "A"}
	Register(lifecycle)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := RunContext(ctx, RunOptions{RunLifecycles: true, StopTimeout: time.Second})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RunContext error = %v, want deadline exceeded", err)
	}
	if lifecycle.stopped.Load() != 1 {
		t.Fatalf("stop calls = %d, want 1", lifecycle.stopped.Load())
	}
}

func TestRunContext_DeadlineExceededJoinsStopError(t *testing.T) {
	ResetApp()
	t.Cleanup(ResetApp)

	stopErr := errors.New("stop failed")
	Register(&scriptedLifecycle{name: "A", stopErr: stopErr})
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := RunContext(ctx, RunOptions{RunLifecycles: true, StopTimeout: time.Second})

	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, stopErr) {
		t.Fatalf("RunContext error = %v, want deadline and stop error", err)
	}
}

func TestRunContext_StartAndStopOrder(t *testing.T) {
	ResetApp()
	t.Cleanup(ResetApp)

	var operations operationLog
	for _, name := range []string{"A", "B", "C"} {
		Register(&scriptedLifecycle{name: name, log: &operations})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := RunContext(ctx, RunOptions{RunLifecycles: true, StopTimeout: time.Second}); err != nil {
		t.Fatalf("RunContext: %v", err)
	}

	wantOrder := []string{"start A", "start B", "start C", "stop C", "stop B", "stop A"}
	if got := operations.snapshot(); !slices.Equal(got, wantOrder) {
		t.Fatalf("operation order = %v, want %v", got, wantOrder)
	}
}

func TestRunContext_StopTimeoutIsReturned(t *testing.T) {
	ResetApp()
	t.Cleanup(ResetApp)

	Register(&scriptedLifecycle{name: "A", waitForStopContext: true})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- RunContext(ctx, RunOptions{RunLifecycles: true, StopTimeout: 10 * time.Millisecond})
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("RunContext error = %v, want stop timeout", err)
		}
	case <-time.After(time.Second):
		t.Fatal("RunContext blocked after stop timeout")
	}
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
		if err != nil {
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
