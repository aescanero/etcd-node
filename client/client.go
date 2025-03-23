package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
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

	// Add debug logging for myConfig
	log.Printf("Config values:\n")
	log.Printf("  RootUser: %s\n", myConfig.RootUser)
	log.Printf("  ListenClientURLs: %s\n", myConfig.ListenClientURLs)
	log.Printf("  CertFile: %s\n", myConfig.CertFile)
	log.Printf("  KeyFile: %s\n", myConfig.KeyFile)
	log.Printf("  TrustedCAFile: %s\n", myConfig.TrustedCAFile)
	// Configurar TLS si están presentes los certificados
	if myConfig.CertFile != "" && myConfig.KeyFile != "" && myConfig.TrustedCAFile != "" {
		tlsConfig, err = loadTLSConfig(myConfig)
		if err != nil {
			return nil, fmt.Errorf("error loading TLS config: %v", err)
		}
	}

	// Add debug logging for TLS config
	if tlsConfig != nil {
		log.Printf("TLS Config loaded:\n")
		log.Printf("  Certificates: %d\n", len(tlsConfig.Certificates))
		log.Printf("  RootCAs: %v\n", tlsConfig.RootCAs != nil)
		log.Printf("  MinVersion: %v\n", tlsConfig.MinVersion)
	} else {
		log.Printf("No TLS Config loaded\n")
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
	// Add debug logging for endpoints
	log.Printf("Endpoints: %v\n", endpoints)
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 3 * time.Second,
		TLS:         tlsConfig,
	})
	// Add debug logging for client

	if err != nil {
		log.Printf("Error creating etcd client: %v", err)
		return nil, fmt.Errorf("error creating etcd client: %v", err)
	}

	if client != nil {
		log.Printf("Client created:\n")
		log.Printf("  Endpoints: %v\n", client.Endpoints())
		log.Printf("  Username: %s\n", client.Username)
		//log.Printf("  Auth enabled: %v\n", client.Auth() != nil)
	} else {
		log.Printf("Client creation failed\n")
	}

	return &Client{
		config:    *myConfig,
		client:    client,
		tlsConfig: tlsConfig,
	}, nil
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
