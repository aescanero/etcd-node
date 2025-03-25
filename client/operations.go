package client

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	pb "go.etcd.io/etcd/api/v3/etcdserverpb"
)

// GetCurrentRevision returns the current revision of the etcd cluster
func (c *Client) GetCurrentRevision(ctx context.Context) (int64, error) {
	// Get status of an endpoint to get the current revision
	var revision int64 = 0
	endpoint := c.client.Endpoints()[0]

	status, err := c.client.Status(ctx, endpoint)
	if err != nil {
		return 0, err
	}

	revision = status.Header.Revision
	return revision, nil
}

// Compact performs logical compaction in etcd
func (c *Client) Compact(ctx context.Context, revision int64, physical bool) error {
	// Perform logical compaction
	_, err := c.client.Compact(ctx, revision)
	if err != nil {
		return fmt.Errorf("logical compaction failed: %v", err)
	}

	// If physical compaction is requested
	if physical {
		// Physical compaction is done by defragmenting
		for _, ep := range c.client.Endpoints() {
			_, err := c.client.Defragment(ctx, ep)
			if err != nil {
				return fmt.Errorf("physical compaction (defragment) failed on %s: %v", ep, err)
			}
		}
	}

	return nil
}

// SaveSnapshot creates a snapshot of the etcd database
func (c *Client) SaveSnapshot(ctx context.Context, filePath string) error {
	// Create snapshot using etcdctl command
	// This is more reliable than the client API for snapshots
	args := []string{"--endpoints", strings.Join(c.client.Endpoints(), ","), "snapshot", "save", filePath}

	// Add authentication if configured
	if c.config.RootUser != "" && c.config.RootPassword != "" {
		args = append([]string{"--user", fmt.Sprintf("%s:%s", c.config.RootUser, c.config.RootPassword)}, args...)
	}

	// Add TLS arguments if configured
	if c.config.CertFile != "" && c.config.KeyFile != "" && c.config.TrustedCAFile != "" {
		args = append([]string{
			"--cacert", c.config.TrustedCAFile,
			"--cert", c.config.CertFile,
			"--key", c.config.KeyFile,
		}, args...)
	}

	// Execute etcdctl command
	cmd := exec.CommandContext(ctx, "etcdctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("snapshot failed: %v, output: %s", err, string(output))
	}

	return nil
}

// RestoreOptions contains options for snapshot restore
type RestoreOptions struct {
	SnapshotPath   string
	DataDir        string
	SkipHashCheck  bool
	Name           string
	ClusterToken   string
	InitialCluster string
}

// RestoreSnapshot restores from a snapshot file
func (c *Client) RestoreSnapshot(ctx context.Context, opts RestoreOptions) error {
	// Build args for etcdctl snapshot restore command
	args := []string{"snapshot", "restore", opts.SnapshotPath}

	// Add data directory
	args = append(args, "--data-dir", opts.DataDir)

	// Add other options
	if opts.SkipHashCheck {
		args = append(args, "--skip-hash-check")
	}

	if opts.Name != "" {
		args = append(args, "--name", opts.Name)
	}

	if opts.ClusterToken != "" {
		args = append(args, "--initial-cluster-token", opts.ClusterToken)
	}

	if opts.InitialCluster != "" {
		args = append(args, "--initial-cluster", opts.InitialCluster)
	}

	// Always force new cluster when restoring
	args = append(args, "--initial-cluster-state", "new")

	// Execute etcdctl command
	cmd := exec.CommandContext(ctx, "etcdctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restore failed: %v, output: %s", err, string(output))
	}

	return nil
}

// ListAlarms lists all alarms in the etcd cluster
func (c *Client) ListAlarms(ctx context.Context) ([]string, error) {
	alarmResponse, err := c.client.AlarmList(ctx)
	if err != nil {
		return nil, err
	}

	var alarms []string
	for _, alarm := range alarmResponse.Alarms {
		switch alarm.Alarm {
		case pb.AlarmType_NOSPACE:
			alarms = append(alarms, "NOSPACE - Database space exceeded")
		case pb.AlarmType_CORRUPT:
			alarms = append(alarms, "CORRUPT - Database corruption detected")
		default:
			alarms = append(alarms, fmt.Sprintf("UNKNOWN(%d)", alarm.Alarm))
		}
	}

	return alarms, nil
}

// EndpointsStatus returns status for all endpoints
func (c *Client) EndpointsStatus(ctx context.Context) (map[string]interface{}, error) {
	endpoints := make(map[string]interface{})

	for _, ep := range c.client.Endpoints() {
		// Get endpoint status
		status, err := c.client.Status(ctx, ep)
		if err != nil {
			endpoints[ep] = map[string]string{
				"error": err.Error(),
			}
			continue
		}

		// Add endpoint status
		endpoints[ep] = map[string]interface{}{
			"version":     status.Version,
			"dbSize":      status.DbSize,
			"leader":      status.Leader,
			"raftIndex":   status.RaftIndex,
			"raftTerm":    status.RaftTerm,
			"raftApplied": status.RaftAppliedIndex,
			"dbSizeInUse": status.DbSizeInUse,
		}
	}

	return endpoints, nil
}

// MemberList returns list of members in the cluster
func (c *Client) MemberList(ctx context.Context) ([]interface{}, error) {
	resp, err := c.client.MemberList(ctx)
	if err != nil {
		return nil, err
	}

	var members []interface{}
	for _, member := range resp.Members {
		members = append(members, map[string]interface{}{
			"id":         member.ID,
			"name":       member.Name,
			"peerURLs":   member.PeerURLs,
			"clientURLs": member.ClientURLs,
			"isLearner":  member.IsLearner,
		})
	}

	return members, nil
}

// GetLeader returns information about the current leader
func (c *Client) GetLeader(ctx context.Context) (interface{}, error) {
	// Get members
	membersResp, err := c.client.MemberList(ctx)
	if err != nil {
		return nil, err
	}

	// We need the status to get the leader ID
	statusResp, err := c.client.Status(ctx, c.client.Endpoints()[0])
	if err != nil {
		return nil, err
	}

	if statusResp.Leader == 0 {
		return map[string]interface{}{
			"error": "No leader elected",
		}, nil
	}

	// Find the leader in the member list
	var leaderInfo map[string]interface{}
	for _, member := range membersResp.Members {
		if member.ID == statusResp.Leader {
			leaderInfo = map[string]interface{}{
				"id":         member.ID,
				"name":       member.Name,
				"peerURLs":   member.PeerURLs,
				"clientURLs": member.ClientURLs,
			}
			break
		}
	}

	if leaderInfo == nil {
		return map[string]interface{}{
			"id":    statusResp.Leader,
			"error": "Leader ID found but member not found in member list",
		}, nil
	}

	return leaderInfo, nil
}

// GetMetrics returns basic metrics from etcd
func (c *Client) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	// Currently, the etcd client v3 doesn't have a direct API for metrics
	// We'll build a limited set of metrics from available information
	metrics := make(map[string]interface{})

	// Get status for an endpoint
	statusResp, err := c.client.Status(ctx, c.client.Endpoints()[0])
	if err != nil {
		return nil, err
	}

	// Get members count
	membersResp, err := c.client.MemberList(ctx)
	if err != nil {
		return nil, err
	}

	// Build basic metrics
	dbMetrics := map[string]interface{}{
		"size":      statusResp.DbSize,
		"sizeInUse": statusResp.DbSizeInUse,
	}

	raftMetrics := map[string]interface{}{
		"index":   statusResp.RaftIndex,
		"term":    statusResp.RaftTerm,
		"applied": statusResp.RaftAppliedIndex,
	}

	clusterMetrics := map[string]interface{}{
		"members": len(membersResp.Members),
		"version": statusResp.Version,
	}

	metrics["db"] = dbMetrics
	metrics["raft"] = raftMetrics
	metrics["cluster"] = clusterMetrics

	return metrics, nil
}

// GetDatabaseSize returns the size of the etcd database in bytes
func (c *Client) GetDatabaseSize(ctx context.Context) (int64, error) {
	statusResp, err := c.client.Status(ctx, c.client.Endpoints()[0])
	if err != nil {
		return 0, err
	}

	return statusResp.DbSize, nil
}
