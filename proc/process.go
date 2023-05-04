package proc

import (
	"bufio"
	"os/exec"
	"syscall"

	"github.com/gayanper/kpm/logger"
)

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
			// if we read something and has not output then restart
			logger.Debug("STDERR:", scanner.Text())
			hasOutput := false
			select {
			case _, hasOutput = <- output:
			default:
			}
			
			if hasOutput {
				p.Restart()
			} else {
				logger.Error(scanner.Text())
				p.SendSigTerm()
			}
		}
	}(cmd, outputChannel)

	return nil
}
