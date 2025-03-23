package client

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func (c *Client) NeedsBootstrap(ctx context.Context, pid int) (bool, error) {
	// First check if the process is running
	if !isEtcdProcessRunning(pid) {
		return true, fmt.Errorf("etcd process is not running")
	}

	// Check socket availability with retries
	endpoint := getLocalEndpoint(c.client.Endpoints())
	socketAvailable := false
	retries := 5
	retryDelay := 10 * time.Second

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
				return true, fmt.Errorf("operation canceled while waiting for socket: %v", ctx.Err())
			}
		}
	}

	if !socketAvailable {
		return true, fmt.Errorf("etcd socket is not available after %d attempts", retries)
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
				return true, fmt.Errorf("operation canceled while waiting for socket to unblock: %v", ctx.Err())
			}
		}
	}

	if !socketUnblocked {
		return true, fmt.Errorf("etcd socket is blocked after %d attempts", retries)
	}

	// Check status with a safe timeout
	statusCtx, statusCancel := context.WithTimeout(ctx, 3*time.Second)
	status, err := c.client.Status(statusCtx, endpoint)
	statusCancel()

	if err != nil {
		return true, fmt.Errorf("error getting status: %v", err)
	}

	// Check if there's a leader
	if status.Leader == 0 {
		return true, fmt.Errorf("no leader elected in the cluster")
	}

	// If we have a root user configured, check if it exists
	if c.config.RootUser != "" {
		userCtx, userCancel := context.WithTimeout(ctx, 3*time.Second)
		_, err := c.client.UserGet(userCtx, c.config.RootUser)
		userCancel()

		if err != nil && !strings.Contains(err.Error(), "user name not found") {
			// If there's an error other than "user not found", something is wrong with the service
			return true, fmt.Errorf("error checking root user: %v", err)
		}

		// If the user doesn't exist, we need bootstrap
		if strings.Contains(err.Error(), "user name not found") {
			return true, nil
		}
	}

	// Everything looks good, we don't need bootstrap
	return false, nil
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
	// Verificar si la autenticación ya está habilitada
	authEnabled := false
	authStatus, err := c.client.AuthStatus(ctx)
	if err != nil {
		if err.Error() == "etcdserver: authentication is not enabled" {
			authEnabled = false
		} else if err.Error() == "etcdserver: user name and password required" {
			authEnabled = true
		} else {
			return fmt.Errorf("error checking auth status: %v", err)
		}
	} else {
		authEnabled = authStatus.Enabled
	}

	if authEnabled {
		return nil // La autenticación ya está habilitada
	}

	if c.config.RootUser == "" || c.config.RootPassword == "" {
		return fmt.Errorf("root user and password are required for authentication setup")
	}

	// Crear usuario root
	_, err = c.client.UserAdd(ctx, c.config.RootUser, c.config.RootPassword)
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
