/*Copyright [2023] [Alejandro Escanero Blanco <aescanero@disasterproject.com>]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.*/

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

// NewLauncher crea un nuevo launcher para etcd
func NewLauncher(myConfig config.Config) *EtcdLauncher {
	return &EtcdLauncher{
		Config: &myConfig,
	}
}

// Start inicia el proceso etcd
func (l *EtcdLauncher) Start() error {
	args := l.Config.BuildArgs()
	l.cmd = exec.Command("etcd", args...)

	// Conectar stdout y stderr
	l.cmd.Stdout = os.Stdout
	l.cmd.Stderr = os.Stderr

	// Iniciar el proceso
	if err := l.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start etcd: %v", err)
	}

	return nil
}

// Wait espera a que el proceso termine
func (l *EtcdLauncher) Wait() error {
	if l.cmd == nil {
		return fmt.Errorf("etcd not started")
	}
	return l.cmd.Wait()
}

// Stop detiene el proceso etcd
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

// IsRunning verifica si el proceso está activo
func (l *EtcdLauncher) IsRunning() bool {
	if l.cmd == nil || l.cmd.Process == nil {
		return false
	}

	// Enviar señal 0 para verificar si el proceso existe
	err := l.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// Monitor inicia un monitoreo periódico del proceso
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

			// Si el proceso no está corriendo, intentamos obtener el estado de salida
			if !status.Running {
				if state, err := l.cmd.Process.Wait(); err == nil {
					status.ExitCode = state.ExitCode()
				}
			}

			// Enviamos el estado actual
			statusChan <- status

			// Si el proceso no está corriendo, terminamos el monitoreo
			if !status.Running {
				return
			}
		}
	}()

	return statusChan
}
