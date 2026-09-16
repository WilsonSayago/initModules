package initModules

import (
	"log"
)

type IProcess interface {
	Start()
}

var processes = make([]IProcess, 0)

// RegisterProcess registers a legacy IProcess on the global app as a ProcessAdapter.
//
// Deprecated: implement Lifecycle and use Register instead.
func RegisterProcess(p IProcess) {
	if isNilValue(p) {
		return
	}
	processes = append(processes, p)
	Register(ProcessAdapter{Process: p})
}

// RunProcesses starts registered IProcess values in goroutines.
//
// Deprecated: implement Lifecycle and use NewApp with RunWithSignals instead.
func RunProcesses() {
	for _, p := range processes {
		if isNilValue(p) {
			log.Println("Start processes: skip nil IProcess")
			continue
		}
		log.Println("Start processes: ", typeName(p))
		go p.Start()
	}
}
