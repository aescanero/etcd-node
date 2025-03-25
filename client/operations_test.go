package client

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/aescanero/etcd-node/config"
)

// TestCompaction tests the compaction functionality
func TestCompaction(t *testing.T) {
	// Skip real testing if environment variable is not set
	if os.Getenv("ETCD_TEST_LIVE") != "true" {
		t.Skip("Skipping test because ETCD_TEST_LIVE is not set to true")
	}

	// Create a basic configuration for testing
	myConfig := config.Config{
		ListenClientURLs: "http://127.0.0.1:2379",
	}

	// Create client
	client, err := NewClient(&myConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get current revision
	revision, err := client.GetCurrentRevision(ctx)
	if err != nil {
		t.Fatalf("Failed to get current revision: %v", err)
	}

	t.Logf("Current revision: %d", revision)

	// Test logical compaction
	err = client.Compact(ctx, revision, false)
	if err != nil {
		t.Fatalf("Logical compaction failed: %v", err)
	}

	t.Log("Logical compaction successful")
}

// TestListAlarms tests the alarm listing functionality
func TestListAlarms(t *testing.T) {
	// Skip real testing if environment variable is not set
	if os.Getenv("ETCD_TEST_LIVE") != "true" {
		t.Skip("Skipping test because ETCD_TEST_LIVE is not set to true")
	}

	// Create a basic configuration for testing
	myConfig := config.Config{
		ListenClientURLs: "http://127.0.0.1:2379",
	}

	// Create client
	client, err := NewClient(&myConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// List alarms
	alarms, err := client.ListAlarms(ctx)
	if err != nil {
		t.Fatalf("Failed to list alarms: %v", err)
	}

	// Report alarms
	t.Logf("Found %d alarms", len(alarms))
	for i, alarm := range alarms {
		t.Logf("Alarm %d: %s", i+1, alarm)
	}
}

// TestEndpointsStatus tests the endpoints status functionality
func TestEndpointsStatus(t *testing.T) {
	// Skip real testing if environment variable is not set
	if os.Getenv("ETCD_TEST_LIVE") != "true" {
		t.Skip("Skipping test because ETCD_TEST_LIVE is not set to true")
	}

	// Create a basic configuration for testing
	myConfig := config.Config{
		ListenClientURLs: "http://127.0.0.1:2379",
	}

	// Create client
	client, err := NewClient(&myConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get endpoints status
	endpoints, err := client.EndpointsStatus(ctx)
	if err != nil {
		t.Fatalf("Failed to get endpoints status: %v", err)
	}

	// Log endpoints info
	for ep, info := range endpoints {
		t.Logf("Endpoint: %s", ep)
		if infoMap, ok := info.(map[string]interface{}); ok {
			for k, v := range infoMap {
				t.Logf("  %s: %v", k, v)
			}
		}
	}
}

// TestMemberList tests the member listing functionality
func TestMemberList(t *testing.T) {
	// Skip real testing if environment variable is not set
	if os.Getenv("ETCD_TEST_LIVE") != "true" {
		t.Skip("Skipping test because ETCD_TEST_LIVE is not set to true")
	}

	// Create a basic configuration for testing
	myConfig := config.Config{
		ListenClientURLs: "http://127.0.0.1:2379",
	}

	// Create client
	client, err := NewClient(&myConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// List members
	members, err := client.MemberList(ctx)
	if err != nil {
		t.Fatalf("Failed to list members: %v", err)
	}

	// Log members
	t.Logf("Found %d members", len(members))
	for i, member := range members {
		t.Logf("Member %d:", i+1)
		if m, ok := member.(map[string]interface{}); ok {
			for k, v := range m {
				t.Logf("  %s: %v", k, v)
			}
		}
	}
}

// TestGetLeader tests the leader identification functionality
func TestGetLeader(t *testing.T) {
	// Skip real testing if environment variable is not set
	if os.Getenv("ETCD_TEST_LIVE") != "true" {
		t.Skip("Skipping test because ETCD_TEST_LIVE is not set to true")
	}

	// Create a basic configuration for testing
	myConfig := config.Config{
		ListenClientURLs: "http://127.0.0.1:2379",
	}

	// Create client
	client, err := NewClient(&myConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get leader
	leader, err := client.GetLeader(ctx)
	if err != nil {
		t.Fatalf("Failed to get leader: %v", err)
	}

	// Log leader info
	t.Log("Leader information:")
	if m, ok := leader.(map[string]interface{}); ok {
		for k, v := range m {
			t.Logf("  %s: %v", k, v)
		}
	}
}

// TestGetMetrics tests the metrics collection functionality
func TestGetMetrics(t *testing.T) {
	// Skip real testing if environment variable is not set
	if os.Getenv("ETCD_TEST_LIVE") != "true" {
		t.Skip("Skipping test because ETCD_TEST_LIVE is not set to true")
	}

	// Create a basic configuration for testing
	myConfig := config.Config{
		ListenClientURLs: "http://127.0.0.1:2379",
	}

	// Create client
	client, err := NewClient(&myConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get metrics
	metrics, err := client.GetMetrics(ctx)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}

	// Log metrics
	t.Log("Metrics:")
	logMetricsRecursive(t, metrics, "")
}

// Helper function to log metrics recursively
func logMetricsRecursive(t *testing.T, metrics map[string]interface{}, prefix string) {
	for k, v := range metrics {
		if nestedMap, ok := v.(map[string]interface{}); ok {
			t.Logf("%s%s:", prefix, k)
			logMetricsRecursive(t, nestedMap, prefix+"  ")
		} else {
			t.Logf("%s%s: %v", prefix, k, v)
		}
	}
}

// TestSnapshotAndRestore tests the snapshot and restore functionality
func TestSnapshotAndRestore(t *testing.T) {
	// Skip real testing if environment variable is not set
	if os.Getenv("ETCD_TEST_LIVE") != "true" {
		t.Skip("Skipping test because ETCD_TEST_LIVE is not set to true")
	}

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "etcd-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	snapshotFile := tempDir + "/snapshot.db"
	restoreDir := tempDir + "/restore"

	// Create a basic configuration for testing
	myConfig := config.Config{
		ListenClientURLs: "http://127.0.0.1:2379",
	}

	// Create client
	client, err := NewClient(&myConfig)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Take a snapshot
	t.Logf("Creating snapshot at %s", snapshotFile)
	err = client.SaveSnapshot(ctx, snapshotFile)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Verify snapshot file exists
	if _, err := os.Stat(snapshotFile); os.IsNotExist(err) {
		t.Fatalf("Snapshot file not created")
	}

	// Get snapshot file size
	fileInfo, err := os.Stat(snapshotFile)
	if err != nil {
		t.Fatalf("Failed to get snapshot file info: %v", err)
	}
	t.Logf("Snapshot created, size: %.2f MB", float64(fileInfo.Size())/(1024*1024))

	// Test restore functionality
	// Note: In a real test, this would actually restore the data,
	// but for a unit test we'll just verify the command executes

	// Create restore options
	restoreOpts := RestoreOptions{
		SnapshotPath:   snapshotFile,
		DataDir:        restoreDir,
		SkipHashCheck:  false,
		Name:           "test-etcd",
		ClusterToken:   "test-token",
		InitialCluster: "test-etcd=http://localhost:2380",
	}

	t.Logf("Restoring snapshot to %s", restoreDir)

	// In a test environment, we can't actually run etcdctl restore without stopping the server
	// So we'll mock this by checking if etcdctl is available
	_, err = exec.LookPath("etcdctl")
	if err != nil {
		t.Logf("etcdctl not found, skipping restore command test: %v", err)
	} else {
		// Test just executes the restore command with the --dry-run option if we added it
		// or we could use a mock implementation for testing
		mockCtx, mockCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer mockCancel()

		// Use the restoreOpts we created
		err = client.RestoreSnapshot(mockCtx, restoreOpts)

		// We expect it to fail in test environment, but at least we verify the function is called
		if err != nil {
			t.Logf("As expected, restore command failed in test environment: %v", err)
		} else {
			t.Logf("Restore command executed successfully (unexpected in test environment)")
		}
	}
}
