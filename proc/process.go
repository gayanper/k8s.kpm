package proc

import (
	"bufio"
	"os/exec"
	"syscall"
	"time"

	"github.com/gayanper/kpm/logger"
)

const CONNECTION_RETRY_INTERVAL = 30
const CONNECTION_RETRY_INTERVAL_SHORT = 1
const MAX_RETRIES = 10

type ProcessCallback func()

type RestartableProcess struct {
	Command      string
	Arguments    []string
	process      *exec.Cmd
	restartCount int16
	Running      bool
	OnRestarted  ProcessCallback
	OnStarted    ProcessCallback
}

func Create(command string, arguments []string, onStarted ProcessCallback, onRestarted ProcessCallback) *RestartableProcess {
	return &RestartableProcess{Command: command, Arguments: arguments, OnStarted: onStarted, OnRestarted: onRestarted, restartCount: 0}
}

func (p *RestartableProcess) SendSigTerm() error {
	pid := p.process.Process.Pid
	err := p.process.Process.Signal(syscall.SIGTERM)
	if err != nil {
		logger.Error("Failed to stop process [pid: ", pid, "].")
		return err
	}
	return nil
}

func (p *RestartableProcess) Restart() {
	err := p.SendSigTerm()
	if err == nil {
		p.Start()
	} else {
		logger.Error("Restarting is suspended due to above error.")
	}
}

func (p *RestartableProcess) Start() error {
	p.Running = false
	outputChannel := make(chan bool, 1)
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
	go func(command *exec.Cmd, output chan bool) {
		out := bufio.NewScanner(stdOut)
		if out.Scan() {
			logger.Debug("STDOUT:", out.Text())
			output <- true
			p.Running = true
			if p.restartCount > 0 && p.OnRestarted != nil {
				p.OnRestarted()
			} else if p.restartCount == 0 && p.OnStarted != nil {
				p.OnStarted()
			}
		}
	}(cmd, outputChannel)

	// start listening for errors
	go func(command *exec.Cmd, output chan bool) {
		scanner := bufio.NewScanner(stdErr)
		if scanner.Scan() {
			p.Running = false
			// if we read something and has no output then restart
			logger.Debug("STDERR:", scanner.Text())
			hasOutput := false
			select {
			case _, hasOutput = <-output:
			default:
			}
			// This means we had a connection, but after a while we may lost it, so restart and see.
			if hasOutput {
				p.restartCount += 1
				p.Restart()
			} else {
				if p.restartCount <= MAX_RETRIES {
					chosenInterval := CONNECTION_RETRY_INTERVAL
					if(p.restartCount < 5) {
						chosenInterval = CONNECTION_RETRY_INTERVAL_SHORT
					}
					logger.Info("Couldn't connect to kubernetes server, will retry in", chosenInterval, "seconds")
					time.Sleep(time.Duration(chosenInterval) * time.Second)
					p.restartCount += 1
					p.Restart()
				} else {
					logger.Error(scanner.Text())
					logger.Info("Giving up after max retries (", MAX_RETRIES, ") exceeded [", command, "]")
					p.SendSigTerm()
				}
			}
		}
	}(cmd, outputChannel)

	return nil
}
