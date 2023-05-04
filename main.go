package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/gayanper/kpm/config"
	"github.com/gayanper/kpm/logger"
	"github.com/gayanper/kpm/proc"
	"github.com/posener/complete"
)

func main() {
	if !hasKubeCtl() {
		logger.Fatal("please install kubectl command.")
	}

	var profile string
	var printHelp bool
	var listProfiles bool
	var debug bool

	// process flags
	flag.StringVar(&profile, "p", "default", "The profile name in configuration file")
	flag.BoolVar(&printHelp, "h", false, "Print help")
	flag.BoolVar(&listProfiles, "l", false, "List profiles from configuration")
	flag.BoolVar(&debug, "v", false, "Verbose output like debug")

	config := config.Read()

	// add completions
	profileNames := func(a complete.Args) []string {
		result := make([]string, len(config))
		i := 0
		for _, p := range config {
			result[i] = p.Name
			i++
		}
		return result
	}

	cmp := complete.New("kpm", complete.Command{Flags: complete.Flags{"-p": complete.PredictFunc(profileNames),
		"-l": complete.PredictNothing, "-h": complete.PredictNothing}})
	cmp.CLI.InstallName = "complete"
	cmp.CLI.UninstallName = "uncomplete"
	cmp.AddFlags(nil)

	flag.Parse()

	if cmp.Complete() {
		return
	}

	logger.DEBUG = debug

	if printHelp {
		flag.PrintDefaults()
		return
	}

	if listProfiles {
		for _, p := range config {
			fmt.Println()
			fmt.Println(p.Name)
			fmt.Println()
		}
		return
	}

	lock := make(chan os.Signal, 1)
	signal.Notify(lock, syscall.SIGTERM)

	p, exist := config[profile]

	if !exist {
		logger.Error("Profile with name [", profile, "] not found in the configuration file")
		return
	}
	procs := startAllPortMappings(p)
	logger.Info()
	logger.Info("Port forwarding started for profile:", profile)
	logger.Info()
	logger.Info("Press Control+C to close down all port forwardings and exit.")
	logger.Info()

	<-lock // wait till we receive sigterm

	killAllPortMappings(procs)
}

func killAllPortMappings(procs []proc.RestartableProcess) {
	for _, proc := range procs {
		proc.SendSigTerm()
	}
}

func startAllPortMappings(profile config.Profile) []proc.RestartableProcess {
	config := profile.Configuration
	procs := make([]proc.RestartableProcess, len(config.Entries))
	for index, entry := range config.Entries {
		arguments := []string{"-n", config.Namespace, "port-forward", entry.ServiceName,
			fmt.Sprint(entry.LocalPort, ":", entry.ServicePort)}
		procs[index] = proc.RestartableProcess{Command: "kubectl", Arguments: arguments}
		procs[index].Start()
	}
	return procs
}

func hasKubeCtl() bool {
	_, err := exec.LookPath("kubectl")
	return err == nil
}
