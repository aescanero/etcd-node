package client

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Status representa el estado actual del servidor etcd
type Status struct {
	IsRunning   bool `default:"false"`
	HasRootUser bool `default:"false"`
	AuthEnabled bool `default:"false"`
	MemberCount int
	IsLeader    bool `default:"false"`
	Version     string
}

// GetStatus verifica el estado actual del servidor etcd
func (c *Client) GetStatus(ctx context.Context) (*Status, error) {
	status := &Status{}
	log.Printf("Default Status values:")
	log.Printf("  IsRunning: %v", status.IsRunning)
	log.Printf("  AuthEnabled: %v", status.AuthEnabled)
	log.Printf("  HasRootUser: %v", status.HasRootUser)
	log.Printf("  MemberCount: %d", status.MemberCount)
	log.Printf("  IsLeader: %v", status.IsLeader)
	//log.Printf("  Version: %s", status.Version)

	// Verificar si el servidor está ejecutándose
	log.Printf("Verify Status")
	time.Sleep(10 * time.Second)
	//ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	//defer cancel()
	log.Printf("Pre Status")
	_, err := c.client.Get(ctx, "health")
	if err != nil {
		log.Panicf("GET failed: %s", err.Error())
	}
	maintenanceStatus, err := c.client.Status(ctx, "http://127.0.0.1:2379")
	if err != nil {
		log.Panicf("STATUS failed: %s", err.Error())
		return status, fmt.Errorf("Bootstrap Status. Server not running: %v", err)
	}

	// Print maintenance status values
	log.Printf("Maintenance Status values:")
	log.Printf("  Version: %s", maintenanceStatus.Version)
	log.Printf("  DbSize: %d", maintenanceStatus.DbSize)
	log.Printf("  Leader: %d", maintenanceStatus.Leader)

	log.Panic("Pre MemberList")
	_, err = c.client.MemberList(ctx)
	log.Printf("Post MemberList")
	if err != nil {
		return status, fmt.Errorf("server not running: %v", err)
	}
	status.IsRunning = true

	// Add debug logging for status
	log.Printf("Status values:")
	log.Printf("  IsRunning: %v", status.IsRunning)
	log.Printf("  AuthEnabled: %v", status.AuthEnabled)
	log.Printf("  HasRootUser: %v", status.HasRootUser)
	log.Printf("  MemberCount: %d", status.MemberCount)
	log.Printf("  IsLeader: %v", status.IsLeader)
	log.Printf("  Version: %s", status.Version)

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
