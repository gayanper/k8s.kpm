package proc

import (
	"bufio"
	"os/exec"
	"regexp"
	"syscall"
	"time"

	"github.com/gayanper/kpm/logger"
)

var K8S_CONNECTION_FAILURE_TEST *regexp.Regexp = regexp.MustCompile(".*dial tcp .* connect: connection refused")
const CONNECTION_RETRY_INTERVAL = 5

type RestartableProcess struct {
	Command string
	Arguments []string
	process *exec.Cmd
}

func (p RestartableProcess) SendSigTerm() error {
	pid := p.process.Process.Pid
	err := p.process.Process.Signal(syscall.SIGTERM)
	if err != nil {
		logger.Error("Failed to stop process [pid: ", pid, "].")
		return err
	}
	return nil
}

func (p RestartableProcess) Restart() {
	err := p.SendSigTerm()
	if err == nil {
		p.Start()
	} else {
		logger.Error("Restarting is suspended due to above error.")
	}
}

func (p RestartableProcess) Start() error {
	outputChannel := make(chan bool)
	cmd := exec.Command(p.Command, p.Arguments...)

	logger.Debug("Starting ", cmd.String())

	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	cmd.Start()
	p.process = cmd

	// start listening for out's first line
	go func (command *exec.Cmd, output chan bool)  {
		out := bufio.NewScanner(stdOut)
		for out.Scan() {
			logger.Debug("STDOUT:", out.Text())
			output <- true
			break
		}
	}(cmd, outputChannel)
	

	// start listening for errors
	go func (command *exec.Cmd, output chan bool)  {
		scanner := bufio.NewScanner(stdErr)
		for scanner.Scan() {
			// if we read something and has no output then restart
			logger.Debug("STDERR:", scanner.Text())
			hasOutput := false
			select {
			case _, hasOutput = <- output:
			default:
			}
			
			if hasOutput {
				p.Restart()
			} else {
				message := scanner.Text()
				if K8S_CONNECTION_FAILURE_TEST.MatchString(message) {
					logger.Info("Couldn't connect to kubernetes server, will retry in", CONNECTION_RETRY_INTERVAL, "seconds")
					time.Sleep(CONNECTION_RETRY_INTERVAL * time.Second)
					p.Restart()
				} else {
					logger.Error(scanner.Text())
					p.SendSigTerm()
				}
			}
		}
	}(cmd, outputChannel)

	return nil
}
