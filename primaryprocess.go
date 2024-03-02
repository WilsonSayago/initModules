package initModules

import (
	"log"
	"reflect"
)

type IProcess interface {
	Start()
}

var processes = make([]IProcess, 0)

func RegisterProcess(p IProcess) {
	processes = append(processes, p)
}

func RunProcesses() {
	for _, p := range processes {
		log.Println("Start processes: ", reflect.TypeOf(p).Elem().Name())
		go p.Start()
	}
}
