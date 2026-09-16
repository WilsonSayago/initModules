package initModules

import (
	"context"
	"fmt"
)

// Lifecycle components support ordered startup and graceful shutdown.
type Lifecycle interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// ProcessAdapter adapts the legacy IProcess interface to Lifecycle.
// Start runs the legacy Start() in a new goroutine; Stop is a no-op.
//
// Prefer implementing Lifecycle directly in new code.
type ProcessAdapter struct {
	Process IProcess
}

func (a ProcessAdapter) Start(ctx context.Context) error {
	if isNilValue(a.Process) {
		return fmt.Errorf("ProcessAdapter: nil IProcess")
	}
	go a.Process.Start()
	return nil
}

func (a ProcessAdapter) Stop(ctx context.Context) error {
	return nil
}
