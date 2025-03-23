package client

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdHealth contains information about the health status of etcd
type EtcdHealth struct {
	IsSocketAvailable bool
	IsSocketBlocked   bool
	ProcessRunning    bool
	DataValid         bool
	HasLeader         bool
	ErrorMessage      string
}

// CheckEtcdHealth performs a comprehensive health check on the etcd server
func (c *Client) CheckEtcdHealth(ctx context.Context, processID int, dataDir string) (*EtcdHealth, error) {
	health := &EtcdHealth{
		ProcessRunning:    false,
		IsSocketAvailable: false,
		IsSocketBlocked:   false,
		DataValid:         false,
		HasLeader:         false,
	}

	// Step 1: Check if the process is running
	health.ProcessRunning = isEtcdProcessRunning(processID)
	if !health.ProcessRunning {
		health.ErrorMessage = "The etcd process is not running"
		return health, nil
	}

	// Step 2: Verify socket availability
	endpoint := getLocalEndpoint(c.client.Endpoints())
	health.IsSocketAvailable = isSocketAvailable(endpoint)
	if !health.IsSocketAvailable {
		health.ErrorMessage = "The etcd socket is not available"
		return health, nil
	}

	// Step 3: Check if the socket is blocked
	health.IsSocketBlocked = isSocketBlocked(c.client)
	if health.IsSocketBlocked {
		health.ErrorMessage = "The etcd socket is blocked"
		return health, nil
	}

	// Step 4: Verify the database
	// Create a child context with a shorter timeout for the status check
	statusCtx, statusCancel := context.WithTimeout(ctx, 3*time.Second)
	defer statusCancel()

	// Try to get the status
	status, err := c.client.Status(statusCtx, endpoint)
	if err != nil {
		health.ErrorMessage = fmt.Sprintf("Error getting status: %v", err)
		return health, nil
	}

	// Check if there's a leader
	health.HasLeader = (status.Leader != 0)
	if !health.HasLeader {
		health.ErrorMessage = "No leader elected in the cluster"
		return health, nil
	}

	// Step 5: Validate database integrity
	isValid, err := validateEtcdDB(c.client)
	health.DataValid = isValid
	if err != nil {
		health.ErrorMessage = fmt.Sprintf("Database validation error: %v", err)
		return health, nil
	}

	return health, nil
}

// isEtcdProcessRunning checks if the etcd process is running
func isEtcdProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix systems, Signal(0) verifies if the process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// getLocalEndpoint extracts the local address from the etcd endpoint
func getLocalEndpoint(endpoints []string) string {
	if len(endpoints) == 0 {
		return "http://127.0.0.1:2379"
	}

	for _, endpoint := range endpoints {
		if strings.Contains(endpoint, "127.0.0.1") ||
			strings.Contains(endpoint, "localhost") ||
			strings.Contains(endpoint, "0.0.0.0") {
			// Ensure we're using the correct port
			if !strings.Contains(endpoint, ":2379") {
				parts := strings.Split(endpoint, ":")
				if len(parts) > 1 {
					return fmt.Sprintf("%s:2379", parts[0])
				}
				return "http://127.0.0.1:2379"
			}
			return endpoint
		}
	}

	// Default value if we don't find a local endpoint
	return "http://127.0.0.1:2379"
}

// isSocketAvailable checks if the etcd socket is available
func isSocketAvailable(endpoint string) bool {
	// Extract host and port from the endpoint
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	// Try a TCP connection with a short timeout
	conn, err := net.DialTimeout("tcp", endpoint, 500*time.Millisecond)
	if err != nil {
		return false
	}

	conn.Close()
	return true
}

// isSocketBlocked checks if the socket is blocked
func isSocketBlocked(client *clientv3.Client) bool {
	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// Channel to receive the response asynchronously
	responseChan := make(chan bool, 1)

	go func() {
		// Try a quick operation
		_, err := client.Get(ctx, "_health_probe_")
		responseChan <- (err != nil)
	}()

	// Wait for the response or timeout
	select {
	case blocked := <-responseChan:
		return blocked
	case <-time.After(500 * time.Millisecond):
		return true // Consider it blocked if there's no response in 500ms
	}
}

// validateEtcdDB validates the integrity of the database
func validateEtcdDB(client *clientv3.Client) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Write/read test
	testKey := fmt.Sprintf("_health_test_%d", time.Now().UnixNano())
	testValue := "ok"

	// Create a new client with limited timeout for each operation
	putCtx, putCancel := context.WithTimeout(ctx, 1*time.Second)
	_, err := client.Put(putCtx, testKey, testValue)
	putCancel()
	if err != nil {
		return false, fmt.Errorf("write error: %v", err)
	}

	// Read the value
	getCtx, getCancel := context.WithTimeout(ctx, 1*time.Second)
	resp, err := client.Get(getCtx, testKey)
	getCancel()
	if err != nil {
		return false, fmt.Errorf("read error: %v", err)
	}

	// Clean up
	delCtx, delCancel := context.WithTimeout(ctx, 1*time.Second)
	_, err = client.Delete(delCtx, testKey)
	delCancel()

	// Verify the value
	if len(resp.Kvs) == 0 || string(resp.Kvs[0].Value) != testValue {
		return false, fmt.Errorf("verification error: value does not match")
	}

	return true, nil
}
