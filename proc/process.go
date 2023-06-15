package proc

import (
	"bufio"
	"os/exec"
	"regexp"
	"syscall"
	"time"

	"github.com/gayanper/kpm/logger"
)

var K8S_CONNECTION_FAILURE_TEST *regexp.Regexp = regexp.MustCompile(".*connection refused")
const CONNECTION_RETRY_INTERVAL = 30
const MAX_RETRIES = 10

type RestartedCallback func ()

type RestartableProcess struct {
	Command string
	Arguments []string
	process *exec.Cmd
	restartCount int16
	Running bool
	OnRestarted RestartedCallback
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
	p.restartCount = 0
	p.Running = false
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
			p.Running = true
			if p.restartCount > 0 && p.OnRestarted != nil {
				p.OnRestarted()
			}
			break
		}
	}(cmd, outputChannel)
	

	// start listening for errors
	go func (command *exec.Cmd, output chan bool)  {
		scanner := bufio.NewScanner(stdErr)
		for scanner.Scan() {
			p.Running = false
			// if we read something and has no output then restart
			logger.Debug("STDERR:", scanner.Text())
			hasOutput := false
			select {
			case _, hasOutput = <- output:
			default:
			}
			// This means we had a connection, but after a while we may lost it, so restart and see.
			if hasOutput {
				p.Restart()
			} else {
				message := scanner.Text()
				if K8S_CONNECTION_FAILURE_TEST.MatchString(message) && p.restartCount <= MAX_RETRIES {
					logger.Info("Couldn't connect to kubernetes server, will retry in", CONNECTION_RETRY_INTERVAL, "seconds")
					time.Sleep(CONNECTION_RETRY_INTERVAL * time.Second)
					p.restartCount += 1
					p.Restart()
				} else {
					logger.Error(scanner.Text())
					p.SendSigTerm()
					if p.restartCount > MAX_RETRIES {
						logger.Info("Giving up after max retries (", MAX_RETRIES, ") exceeded [", command, "]")
					}
				}
			}
		}
	}(cmd, outputChannel)

	return nil
}
