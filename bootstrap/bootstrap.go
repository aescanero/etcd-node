package bootstrap

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aescanero/etcd-node/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Client es el cliente de bootstrap para etcd
type Client struct {
	config    config.Config
	client    *clientv3.Client
	tlsConfig *tls.Config
}

// NewClient crea un nuevo cliente de bootstrap
func NewClient(myConfig *config.Config) (*Client, error) {
	var tlsConfig *tls.Config
	var err error

	// Configurar TLS si están presentes los certificados
	if myConfig.CertFile != "" && myConfig.KeyFile != "" && myConfig.TrustedCAFile != "" {
		tlsConfig, err = loadTLSConfig(myConfig)
		if err != nil {
			return nil, fmt.Errorf("error loading TLS config: %v", err)
		}
	}

	// Extraer endpoints del ListenClientURLs
	endpoints := []string{"http://127.0.0.1:2379"} // Default
	if myConfig.ListenClientURLs != "" {
		endpoints = strings.Split(myConfig.ListenClientURLs, ",")
		// Reemplazar 0.0.0.0 por localhost
		for i, endpoint := range endpoints {
			endpoints[i] = strings.Replace(endpoint, "0.0.0.0", "127.0.0.1", 1)
		}
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		TLS:         tlsConfig,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating etcd client: %v", err)
	}

	return &Client{
		config:    *myConfig,
		client:    client,
		tlsConfig: tlsConfig,
	}, nil
}

// Status representa el estado actual del servidor etcd
type Status struct {
	IsRunning   bool
	AuthEnabled bool
	HasRootUser bool
	MemberCount int
	IsLeader    bool
	Version     string
}

// GetStatus verifica el estado actual del servidor etcd
func (c *Client) GetStatus(ctx context.Context) (*Status, error) {
	status := &Status{}

	// Verificar si el servidor está ejecutándose
	_, err := c.client.MemberList(ctx)
	if err != nil {
		return status, fmt.Errorf("server not running: %v", err)
	}
	status.IsRunning = true

	// Verificar autenticación
	authStatus, err := c.client.AuthStatus(ctx)
	if err != nil {
		if err.Error() == "etcdserver: authentication is not enabled" {
			status.AuthEnabled = false
		} else if err.Error() == "etcdserver: user name and password required" {
			status.AuthEnabled = true
		} else {
			return status, fmt.Errorf("error checking auth status: %v", err)
		}
	} else {
		status.AuthEnabled = authStatus.Enabled
	}

	// Verificar usuario root
	if c.config.RootUser != "" {
		if status.AuthEnabled {
			authenticatedClient, err := clientv3.New(clientv3.Config{
				Endpoints:   c.client.Endpoints(),
				DialTimeout: 5 * time.Second,
				Username:    c.config.RootUser,
				Password:    c.config.RootPassword,
			})
			if err == nil {
				defer authenticatedClient.Close()
				_, err = authenticatedClient.UserGet(ctx, c.config.RootUser)
				status.HasRootUser = (err == nil)
			}
		} else {
			_, err = c.client.UserGet(ctx, c.config.RootUser)
			status.HasRootUser = (err == nil)
		}
	}

	// Obtener información del cluster
	members, err := c.client.MemberList(ctx)
	if err == nil {
		status.MemberCount = len(members.Members)
	}

	// Verificar si este nodo es el líder
	if resp, err := c.client.Status(ctx, c.client.Endpoints()[0]); err == nil {
		status.IsLeader = resp.Leader == resp.Header.MemberId
		status.Version = resp.Version
	}

	return status, nil
}

// NeedsBootstrap determina si el servidor necesita bootstrap
func (c *Client) NeedsBootstrap(ctx context.Context) (bool, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return true, err
	}

	needsBootstrap := !status.IsRunning ||
		(c.config.RootUser != "" && !status.HasRootUser)

	return needsBootstrap, nil
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

// waitForEtcd espera hasta que etcd esté disponible
func (c *Client) waitForEtcd(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_, err := c.client.MemberList(ctx)
			if err == nil {
				return nil
			}
		}
	}
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

// loadTLSConfig carga la configuración TLS
func loadTLSConfig(config *config.Config) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("error loading key pair: %v", err)
	}

	caData, err := os.ReadFile(config.TrustedCAFile)
	if err != nil {
		return nil, fmt.Errorf("error reading CA file: %v", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// Close cierra el cliente
func (c *Client) Close() error {
	return c.client.Close()
}
