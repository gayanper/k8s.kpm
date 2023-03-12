package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/gayanper/kpm/config"
	"github.com/gayanper/kpm/logger"
)

func main() {
	if !hasKubeCtl() {
		logger.Fatal("please install kubectl command.")
	}

	profile := "default"

	config := config.Read("")
	lock := make(chan os.Signal, 1)
	signal.Notify(lock, syscall.SIGTERM)

	procs := startAllPortMappings(config[profile])
	logger.Info()
	logger.Info("Port forwarding started for profile:", profile)
	logger.Info()
	logger.Info("Press Control+C to close down all port forwardings and exit.")
	logger.Info()

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
			logger.Error("failed to stop kubectl process [pid: ", pid, "].")
		}
	}
}

func startAllPortMappings(profile config.Profile) []*exec.Cmd {
	config := profile.Configuration
	procs := make([]*exec.Cmd, len(config.Entries))
	for index, entry := range config.Entries {
		procs[index] = runCommand("kubectl", "-n", config.Namespace, "port-forward", entry.ServiceName,
		fmt.Sprint(entry.LocalPort, ":", entry.ServicePort))
	}
	return procs
}

func hasKubeCtl() bool {
	_, err := exec.LookPath("kubectl")
	return err == nil
}

func runCommand(command string, args ...string) *exec.Cmd {
	c := exec.Command(command, args...)

	go func (command *exec.Cmd)  {
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		err := command.Run()
		if err != nil {
			logger.Error(err)
		}
	}(c)

	return c
}
