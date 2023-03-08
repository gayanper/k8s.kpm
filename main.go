package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/gayanper/kpm/config"
)

func main() {
	if !hasKubeCtl() {
		log.Fatalln("please install kubectl command.")
	}

	config := config.Read("")
	lock := make(chan os.Signal, 1)
	signal.Notify(lock, syscall.SIGTERM)

	procs := startAllPortMappings(config)
	log.Println("press Control+C to close down all port forwardings and exit.")

	<-lock // wait till we receive sigterm

	killAllPortMappings(procs)
}

func killAllPortMappings(procs []*exec.Cmd) {
	for _, proc := range procs {
		if proc == nil {
			continue
		}
		pid := proc.Process.Pid
		e := proc.Process.Signal(syscall.SIGTERM)
		if e != nil {
			log.Println("failed to stop kubectl process [pid: ", pid, "].")
		}
	}
}

func startAllPortMappings(config config.Config) []*exec.Cmd {
	procs := make([]*exec.Cmd, len(config.Entries))
	for index, entry := range config.Entries {
		kcmd := exec.Command("kubectl", "-n", config.Namespace, "port-forward", entry.ServiceName,
			fmt.Sprint(entry.LocalPort, ":", entry.ServicePort))

		if e := kcmd.Start(); e != nil {
			procs[index] = nil
		} else {
			procs[index] = kcmd
		}
	}
	return procs
}

func hasKubeCtl() bool {
	_, err := exec.LookPath("kubectl")
	return err == nil
}
