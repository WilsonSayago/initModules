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
	if t.started.Add(1) == 1 {
		close(t.startedSignal)
	}
	return nil
}

func (t *testLifecycle) Stop(ctx context.Context) error {
	if t.stopped.Add(1) == 1 {
		close(t.stoppedSignal)
	}
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

func TestProcessAdapter_NilProcess(t *testing.T) {
	t.Parallel()

	err := ProcessAdapter{}.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "nil IProcess") {
		t.Fatalf("error = %v, want nil IProcess", err)
	}

	var typed IProcess = (*pointerProcess)(nil)
	err = ProcessAdapter{Process: typed}.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "nil IProcess") {
		t.Fatalf("typed-nil error = %v, want nil IProcess", err)
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

func TestApp_NewAppIsIsolatedFromGlobal(t *testing.T) {
	ResetApp()
	t.Cleanup(ResetApp)

	globalLC := newTestLifecycle()
	Register(globalLC)

	app := NewApp()
	localLC := newTestLifecycle()
	app.Register(localLC)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := app.RunContext(ctx, RunOptions{RunLifecycles: true, StopTimeout: time.Second}); err != nil {
		t.Fatalf("App.RunContext: %v", err)
	}

	if localLC.started.Load() != 1 || localLC.stopped.Load() != 1 {
		t.Fatal("instance lifecycle was not run")
	}
	if globalLC.started.Load() != 0 {
		t.Fatal("global lifecycle must not start on a distinct App")
	}
}

func TestApp_IsolatedParallelRuns(t *testing.T) {
	a1 := NewApp()
	a2 := NewApp()
	lc1 := newTestLifecycle()
	lc2 := newTestLifecycle()
	a1.Register(lc1)
	a2.Register(lc2)

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	err1 := make(chan error, 1)
	err2 := make(chan error, 1)

	go func() {
		err1 <- a1.RunContext(ctx1, RunOptions{RunLifecycles: true, StopTimeout: time.Second})
	}()
	go func() {
		err2 <- a2.RunContext(ctx2, RunOptions{RunLifecycles: true, StopTimeout: time.Second})
	}()

	waitForSignal(t, lc1.startedSignal, "app1 Start")
	waitForSignal(t, lc2.startedSignal, "app2 Start")
	if lc1.stopped.Load() != 0 || lc2.stopped.Load() != 0 {
		t.Fatal("neither app should have stopped before cancel")
	}

	cancel1()
	cancel2()

	for i, errCh := range []chan error{err1, err2} {
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("app %d: %v", i+1, err)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout waiting for app %d", i+1)
		}
	}
	if lc1.stopped.Load() != 1 || lc2.stopped.Load() != 1 {
		t.Fatal("expected each app to stop only its own lifecycle")
	}
}

func TestApp_ConcurrentRegister(t *testing.T) {
	app := NewApp()
	const n = 50
	lifecycles := make([]*scriptedLifecycle, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		lc := &scriptedLifecycle{}
		lifecycles[i] = lc
		wg.Add(1)
		go func() {
			defer wg.Done()
			app.Register(lc)
		}()
	}
	wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := app.RunContext(ctx, RunOptions{RunLifecycles: true, StopTimeout: time.Second}); err != nil {
		t.Fatalf("App.RunContext: %v", err)
	}
	for i, lc := range lifecycles {
		if lc.started.Load() != 1 {
			t.Fatalf("lifecycle %d starts = %d, want 1", i, lc.started.Load())
		}
	}
}

func TestApp_RejectsConcurrentRun(t *testing.T) {
	app := NewApp()
	lc := newTestLifecycle()
	app.Register(lc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.RunContext(ctx, RunOptions{RunLifecycles: true, StopTimeout: time.Second})
	}()
	waitForSignal(t, lc.startedSignal, "Start")

	err := app.RunContext(ctx, RunOptions{RunLifecycles: true, StopTimeout: time.Second})
	if !errors.Is(err, ErrAppRunning) {
		t.Fatalf("second run error = %v, want ErrAppRunning", err)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("first run: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for first run")
	}
}

func TestApp_NilContextDoesNotPanic(t *testing.T) {
	app := NewApp()
	app.Register(&failingLifecycle{})

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		err := app.RunContext(nil, RunOptions{RunLifecycles: true, StopTimeout: time.Second}) //nolint:staticcheck // documents nil-context fallback
		if err == nil {
			t.Fatal("expected start error")
		}
		if err := app.RunWithSignals(nil, RunOptions{RunLifecycles: true, StopTimeout: time.Second}); err == nil { //nolint:staticcheck // documents nil-context fallback
			t.Fatal("expected start error from RunWithSignals")
		}
	}()
	if recovered != nil {
		t.Fatalf("panic: %v", recovered)
	}
}

func TestApp_RegisterNilAndTypedNil(t *testing.T) {
	app := NewApp()
	app.Register(nil)
	var typed Lifecycle = (*testLifecycle)(nil)
	app.Register(typed)

	keep := &scriptedLifecycle{name: "keep"}
	app.Register(keep)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := app.RunContext(ctx, RunOptions{RunLifecycles: true, StopTimeout: time.Second}); err != nil {
		t.Fatalf("App.RunContext: %v", err)
	}
	if keep.started.Load() != 1 {
		t.Fatal("expected only the non-nil lifecycle to start")
	}
}

func TestApp_RegisterDuringRunAppliesToNextRun(t *testing.T) {
	app := NewApp()
	first := newTestLifecycle()
	second := &scriptedLifecycle{name: "late"}
	app.Register(first)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.RunContext(ctx, RunOptions{RunLifecycles: true, StopTimeout: time.Second})
	}()
	waitForSignal(t, first.startedSignal, "Start")
	app.Register(second)
	if second.started.Load() != 0 {
		t.Fatal("lifecycle registered mid-run must not start")
	}
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("first run: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for first run")
	}

	ctx2, cancel2 := context.WithCancel(context.Background())
	cancel2()
	if err := app.RunContext(ctx2, RunOptions{RunLifecycles: true, StopTimeout: time.Second}); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if second.started.Load() != 1 {
		t.Fatal("lifecycle registered mid-run must start on the next run")
	}
}

func TestApp_RunContextMatchesGlobalErrorSemantics(t *testing.T) {
	app := NewApp()
	var operations operationLog
	startErr := errors.New("start C")
	stopAErr := errors.New("stop A")
	stopBErr := errors.New("stop B")
	a := &scriptedLifecycle{name: "A", log: &operations, stopErr: stopAErr}
	b := &scriptedLifecycle{name: "B", log: &operations, stopErr: stopBErr}
	c := &scriptedLifecycle{name: "C", log: &operations, startErr: startErr}
	app.Register(a)
	app.Register(b)
	app.Register(c)

	err := app.RunContext(context.Background(), RunOptions{RunLifecycles: true, StopTimeout: time.Second})
	for _, wantErr := range []error{startErr, stopAErr, stopBErr} {
		if !errors.Is(err, wantErr) {
			t.Errorf("App.RunContext error = %v, want errors.Is(_, %v)", err, wantErr)
		}
	}
}

func TestRunWithSignals_CancelParent(t *testing.T) {
	ResetApp()
	t.Cleanup(ResetApp)

	lc := newTestLifecycle()
	Register(lc)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunWithSignals(ctx, RunOptions{RunLifecycles: true, StopTimeout: time.Second})
	}()
	waitForSignal(t, lc.startedSignal, "Start")
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("RunWithSignals: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for RunWithSignals")
	}
	if lc.stopped.Load() != 1 {
		t.Fatal("expected Stop to be called")
	}
}
