package initModules

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func Init(enableLoadProp bool, enableLoadProcesses bool) {
	if enableLoadProp {
		RunLoadProperties()
	}
	if enableLoadProcesses {
		RunProcesses()
	}
}

func Run(enableLoadProp bool, enableLoadProcesses bool) {
	Init(enableLoadProp, enableLoadProcesses)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGKILL)
	
	s := <-signalChan
	log.Println("Got signal: ", s)
	os.Exit(0)
}
