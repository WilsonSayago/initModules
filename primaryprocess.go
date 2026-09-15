package initModules

import (
	"log"
	"reflect"
)

type IProcess interface {
	Start()
}

var processes = make([]IProcess, 0)

// RegisterProcess registers a legacy IProcess on the global app as a ProcessAdapter.
//
// Deprecated: implement Lifecycle and use Register instead.
func RegisterProcess(p IProcess) {
	if p == nil {
		return
	}
	processes = append(processes, p)
	Register(ProcessAdapter{Process: p})
}

func RunProcesses() {
	for _, p := range processes {
		log.Println("Start processes: ", reflect.TypeOf(p).Elem().Name())
		go p.Start()
	}
}
