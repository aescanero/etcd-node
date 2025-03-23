package service

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/aescanero/etcd-node/config"
)

type EtcdLauncher struct {
	Config *config.Config
	cmd    *exec.Cmd
}

// NewLauncher creates a new launcher for etcd
func NewLauncher(myConfig config.Config) *EtcdLauncher {
	return &EtcdLauncher{
		Config: &myConfig,
	}
}

// Start starts the etcd process
func (l *EtcdLauncher) Start() error {
	args := l.Config.BuildArgs()
	l.cmd = exec.Command("etcd", args...)

	// Connect stdout and stderr
	l.cmd.Stdout = os.Stdout
	l.cmd.Stderr = os.Stderr

	// Start the process
	if err := l.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start etcd: %v", err)
	}

	return nil
}

// GetProcessID returns the process ID of the running etcd instance
func (l *EtcdLauncher) GetProcessID() int {
	if l.cmd == nil || l.cmd.Process == nil {
		return -1
	}
	return l.cmd.Process.Pid
}

// Wait waits for the process to terminate
func (l *EtcdLauncher) Wait() error {
	if l.cmd == nil {
		return fmt.Errorf("etcd not started")
	}
	return l.cmd.Wait()
}

// Stop stops the etcd process
func (l *EtcdLauncher) Stop() error {
	if l.cmd == nil || l.cmd.Process == nil {
		return nil
	}
	return l.cmd.Process.Signal(os.Interrupt)
}

type ProcessStatus struct {
	Pid      int
	Running  bool
	ExitCode int
}

// IsRunning checks if the process is active
func (l *EtcdLauncher) IsRunning() bool {
	if l.cmd == nil || l.cmd.Process == nil {
		return false
	}

	// Send signal 0 to check if the process exists
	err := l.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// Monitor starts periodic monitoring of the process
func (l *EtcdLauncher) Monitor(interval time.Duration) chan ProcessStatus {
	statusChan := make(chan ProcessStatus)
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		defer close(statusChan)

		for range ticker.C {
			status := ProcessStatus{
				Pid:     l.cmd.Process.Pid,
				Running: l.IsRunning(),
			}

			// If the process is not running, try to get the exit status
			if !status.Running {
				if state, err := l.cmd.Process.Wait(); err == nil {
					status.ExitCode = state.ExitCode()
				}
			}

			// Send the current status
			statusChan <- status

			// If the process is not running, end monitoring
			if !status.Running {
				return
			}
		}
	}()

	return statusChan
}
