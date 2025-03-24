package client

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

/*
*
/**
NeedsBootstrap checks if the etcd cluster needs to be bootstrapped by performing
several validation checks:

1. Verifies if the etcd process is running using the provided process ID
2. Checks if the etcd socket is available and accessible with retries
3. Validates that the socket is not blocked with retries
4. Confirms there is an elected leader in the cluster
5. If a root user is configured, verifies if it exists

Parameters:
  - ctx: Context for timeout/cancellation
  - pid: Process ID of the etcd server

Returns:
  - bool: true if bootstrap is needed, false otherwise
  - error: Error details if any check fails

The function will retry socket checks up to 5 times with 10 second delays between
attempts. It uses context for proper timeout handling and cancellation support.

Common error cases:
- Process not running
- Socket unavailable after retries
- Socket blocked after retries
- No leader elected
- Root user does not exist (when configured)
- Authentication/connection errors
*/
func (c *Client) NeedsBootstrap(ctx context.Context, pid int) (bool, error) {
	// First check if the process is running
	if !isEtcdProcessRunning(pid) {
		return false, fmt.Errorf("etcd process is not running")
	}

	// Check socket availability with retries
	endpoint := getLocalEndpoint(c.client.Endpoints())
	socketAvailable := false
	retries := 5
	retryDelay := 10 * time.Second
	authEnabled := false

	for attempt := 1; attempt <= retries; attempt++ {
		log.Printf("Socket availability check attempt %d of %d for endpoint %s", attempt, retries, endpoint)

		if isSocketAvailable(endpoint) {
			socketAvailable = true
			log.Printf("Socket is available after %d attempt(s)", attempt)
			break
		}

		if attempt < retries {
			// Only wait if we're going to retry
			log.Printf("Socket not available. Waiting %v before retry...", retryDelay)

			// Create a timer for the delay
			timer := time.NewTimer(retryDelay)

			// Wait for either the timer to complete or the context to be canceled
			select {
			case <-timer.C:
				// Timer completed, continue to next attempt
			case <-ctx.Done():
				// Context was canceled
				if !timer.Stop() {
					<-timer.C // Drain the timer channel if it already fired
				}
				return false, fmt.Errorf("operation canceled while waiting for socket: %v", ctx.Err())
			}
		}
	}

	if !socketAvailable {
		return false, fmt.Errorf("etcd socket is not available after %d attempts", retries)
	}

	// Check if the socket is blocked with retries
	socketUnblocked := false

	for attempt := 1; attempt <= retries; attempt++ {
		log.Printf("Socket block check attempt %d of %d", attempt, retries)

		if !isSocketBlocked(c.client) {
			socketUnblocked = true
			log.Printf("Socket is unblocked after %d attempt(s)", attempt)
			break
		}

		if attempt < retries {
			// Only wait if we're going to retry
			log.Printf("Socket is blocked. Waiting %v before retry...", retryDelay)

			// Create a timer for the delay
			timer := time.NewTimer(retryDelay)

			// Wait for either the timer to complete or the context to be canceled
			select {
			case <-timer.C:
				// Timer completed, continue to next attempt
			case <-ctx.Done():
				// Context was canceled
				if !timer.Stop() {
					<-timer.C // Drain the timer channel if it already fired
				}
				return false, fmt.Errorf("operation canceled while waiting for socket to unblock: %v", ctx.Err())
			}
		}
	}

	if !socketUnblocked {
		return false, fmt.Errorf("etcd socket is blocked after %d attempts", retries)
	}

	// Check status with a safe timeout
	statusCtx, statusCancel := context.WithTimeout(ctx, 3*time.Second)
	status, err := c.client.Status(statusCtx, endpoint)
	statusCancel()

	if err != nil {
		return false, fmt.Errorf("error getting status: %v", err)
	}

	// Check if there's a leader
	if status.Leader == 0 {
		return false, fmt.Errorf("no leader elected in the cluster")
	}

	//Check authStatus
	authStatus, err := c.client.AuthStatus(ctx)
	if err != nil {
		if err.Error() == "etcdserver: authentication is not enabled" {
			authEnabled = false
		} else if err.Error() == "etcdserver: user name and password required" {
			authEnabled = true
		} else {
			return false, fmt.Errorf("error checking auth status: %v", err)
		}
	} else {
		authEnabled = authStatus.Enabled
	}

	return !authEnabled, nil
}

// Bootstrap realiza el proceso de bootstrap
func (c *Client) Bootstrap(ctx context.Context) error {
	// Verificar conectividad
	if err := c.waitForEtcd(ctx); err != nil {
		return fmt.Errorf("error waiting for etcd: %v", err)
	}

	// Configurar autenticación si hay usuario root configurado
	if c.config.RootUser != "" {
		if err := c.setupAuth(ctx); err != nil {
			return fmt.Errorf("error setting up authentication: %v", err)
		}
	}

	return nil
}

// setupAuth configura la autenticación en etcd
func (c *Client) setupAuth(ctx context.Context) error {
	if c.config.RootUser == "" || c.config.RootPassword == "" {
		return fmt.Errorf("root user and password are required for authentication setup")
	}

	// Crear usuario root
	_, err := c.client.UserAdd(ctx, c.config.RootUser, c.config.RootPassword)
	if err != nil && !strings.Contains(err.Error(), "user exists") {
		return fmt.Errorf("error creating root user: %v", err)
	}

	// Asignar rol root
	_, err = c.client.UserGrantRole(ctx, c.config.RootUser, "root")
	if err != nil && !strings.Contains(err.Error(), "role already exists") {
		return fmt.Errorf("error granting root role: %v", err)
	}

	// Habilitar autenticación
	_, err = c.client.AuthEnable(ctx)
	if err != nil {
		return fmt.Errorf("error enabling authentication: %v", err)
	}

	// Reconectar con credenciales
	c.client.Close()
	newClient, err := clientv3.New(clientv3.Config{
		Endpoints:   c.client.Endpoints(),
		DialTimeout: 5 * time.Second,
		Username:    c.config.RootUser,
		Password:    c.config.RootPassword,
		TLS:         c.tlsConfig,
	})
	if err != nil {
		return fmt.Errorf("error reconnecting with credentials: %v", err)
	}
	c.client = newClient

	return nil
}
