/*Copyright [2025] [Alejandro Escanero Blanco <aescanero@disasterproject.com>]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.*/

package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aescanero/etcd-node/client"
	"github.com/aescanero/etcd-node/config"
	"github.com/aescanero/etcd-node/utils"
	"github.com/spf13/cobra"
)

var (
	// Restore flags
	snapshotFileFlag   string
	dataDirFlag        string
	skipHashCheckFlag  bool
	forceFlag          bool
	restoreTimeoutFlag int
)

func init() {
	// Default data directory
	defaultDataDir := utils.GetEnv("ETCD_DATA_DIR", "/var/run/etcd")

	restoreCmd.Flags().StringVarP(&snapshotFileFlag, "snapshot-file", "f", "", "Snapshot file to restore from (required)")
	restoreCmd.Flags().StringVarP(&dataDirFlag, "data-dir", "d", defaultDataDir, "Data directory for restored data")
	restoreCmd.Flags().BoolVar(&skipHashCheckFlag, "skip-hash-check", false, "Skip integrity checking of snapshot")
	restoreCmd.Flags().BoolVar(&forceFlag, "force", false, "Force restore even if data directory exists")
	restoreCmd.Flags().IntVarP(&restoreTimeoutFlag, "timeout", "t", 120, "Timeout in seconds for the restore operation")

	restoreCmd.MarkFlagRequired("snapshot-file")
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore etcd from snapshot",
	Long:  `Restore etcd database from a snapshot file`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if snapshot file exists
		if _, err := os.Stat(snapshotFileFlag); os.IsNotExist(err) {
			log.Fatalf("Snapshot file %s does not exist", snapshotFileFlag)
		}

		// Check if data directory is suitable for restore
		if _, err := os.Stat(dataDirFlag); err == nil {
			// Directory exists
			empty, err := isDirEmpty(dataDirFlag)
			if err != nil {
				log.Fatalf("Error checking data directory: %v", err)
			}

			if !empty && !forceFlag {
				log.Fatalf("Data directory %s is not empty. Use --force to overwrite", dataDirFlag)
			}

			if !empty && forceFlag {
				log.Printf("Warning: Forcing restore to non-empty directory %s", dataDirFlag)
			}
		} else if os.IsNotExist(err) {
			// Directory doesn't exist, create it
			if err := os.MkdirAll(dataDirFlag, 0755); err != nil {
				log.Fatalf("Failed to create data directory %s: %v", dataDirFlag, err)
			}
		} else {
			log.Fatalf("Error checking data directory: %v", err)
		}

		// Load configuration
		myConfig := config.Config{
			DataDir: dataDirFlag,
		}

		// Create client (for restore we don't need a connection,
		// but we use the client struct for consistency)
		etcdClient, err := client.NewClient(&myConfig)
		if err != nil {
			log.Printf("Warning: Error creating etcd client, continuing with restore: %v", err)
		}

		// Perform the restore operation
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(restoreTimeoutFlag)*time.Second)
		defer cancel()

		restoreOpts := client.RestoreOptions{
			SnapshotPath:   snapshotFileFlag,
			DataDir:        dataDirFlag,
			SkipHashCheck:  skipHashCheckFlag,
			Name:           Name,
			ClusterToken:   InitialClusterToken,
			InitialCluster: InitialCluster,
		}

		log.Printf("Restoring snapshot %s to %s", snapshotFileFlag, dataDirFlag)

		startTime := time.Now()
		if err = etcdClient.RestoreSnapshot(ctx, restoreOpts); err != nil {
			log.Fatalf("Restore failed: %v", err)
		}
		duration := time.Since(startTime)

		fmt.Printf("Successfully restored etcd from snapshot %s (Duration: %v)\n", snapshotFileFlag, duration)
		fmt.Println("Note: You must restart etcd for the changes to take effect")
	},
}

// Helper function to check if directory is empty
func isDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// Read just one entry
	_, err = f.Readdirnames(1)
	if err == nil {
		// Not empty
		return false, nil
	} else if err.Error() == "EOF" {
		// Empty directory
		return true, nil
	}

	return false, err
}
