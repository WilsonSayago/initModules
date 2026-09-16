package initModules

import (
	"context"
	"sync/atomic"
	"testing"
)

type valueProcess struct {
	started chan struct{}
}

func (v valueProcess) Start() {
	close(v.started)
}

type pointerProcess struct {
	started chan struct{}
}

func (p *pointerProcess) Start() {
	close(p.started)
}

func resetProcessesForTest(t *testing.T) {
	t.Helper()
	ResetApp()
	processes = nil
	t.Cleanup(func() {
		ResetApp()
		processes = nil
	})
}

func TestLegacy_RunProcesses_ValuePointerAndTypedNil(t *testing.T) {
	resetProcessesForTest(t)

	valueStarted := make(chan struct{})
	pointerStarted := make(chan struct{})
	RegisterProcess(valueProcess{started: valueStarted})
	RegisterProcess(&pointerProcess{started: pointerStarted})

	var typed IProcess = (*pointerProcess)(nil)
	RegisterProcess(typed)
	RegisterProcess(nil)

	RunProcesses()

	waitForSignal(t, valueStarted, "value IProcess")
	waitForSignal(t, pointerStarted, "pointer IProcess")
}

func TestLegacy_RegisterProcess_IgnoresTypedNil(t *testing.T) {
	resetProcessesForTest(t)

	var typed IProcess = (*pointerProcess)(nil)
	RegisterProcess(typed)
	if len(processes) != 0 {
		t.Fatalf("processes = %d, want 0", len(processes))
	}
}

func TestLegacy_RunProcesses_SkipsTypedNilWithoutPanic(t *testing.T) {
	resetProcessesForTest(t)

	started := make(chan struct{})
	processes = []IProcess{(*pointerProcess)(nil), valueProcess{started: started}}
	RunProcesses()
	waitForSignal(t, started, "value IProcess after typed nil")
}

func TestProcessAdapter_ValidProcessStartsOnce(t *testing.T) {
	var count atomic.Int32
	started := make(chan struct{})
	adapter := ProcessAdapter{Process: legacyProcessFunc(func() {
		if count.Add(1) == 1 {
			close(started)
		}
	})}
	if err := adapter.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForSignal(t, started, "adapter Start")
	if count.Load() != 1 {
		t.Fatalf("Start calls = %d, want 1", count.Load())
	}
}
