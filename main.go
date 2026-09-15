package initModules

import (
	"context"
	"log"
)

// Init loads properties and/or starts registered processes using the legacy goroutine model.
//
// Deprecated: use RunContext or RunWithSignals with Register(Lifecycle) instead.
func Init(enableLoadProp bool, enableLoadProcesses bool) {
	if enableLoadProp {
		RunLoadProperties()
	}
	if enableLoadProcesses {
		RunProcesses()
	}
}

// Run loads optional properties, runs registered lifecycles with signal-aware shutdown, and returns.
// It does not call os.Exit; the application main controls exit code.
//
// When RunLifecycles is true, components registered via Register or RegisterProcess (adapter) are started and stopped gracefully.
// Legacy RunProcesses (fire-and-forget goroutines) is not used when the global app registry has lifecycles.
func Run(enableLoadProp bool, enableLoadProcesses bool) {
	if err := RunWithSignals(context.Background(), RunOptions{
		LoadProperties: enableLoadProp,
		RunLifecycles: enableLoadProcesses,
	}); err != nil {
		log.Fatal(err)
	}
}
