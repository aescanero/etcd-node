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
	"path/filepath"
	"time"

	"github.com/aescanero/etcd-node/client"
	"github.com/aescanero/etcd-node/config"
	"github.com/aescanero/etcd-node/utils"
	"github.com/spf13/cobra"
)

var (
	// Snapshot flags
	outputFileFlag      string
	snapshotTimeoutFlag int
)

func init() {
	// Get default snapshot location
	defaultSnapshotDir := utils.GetEnv("ETCD_SNAPSHOT_DIR", "/var/run/etcd/snapshots")
	defaultFilename := fmt.Sprintf("etcd-snapshot-%s.db", time.Now().Format("20060102-150405"))
	defaultOutput := filepath.Join(defaultSnapshotDir, defaultFilename)

	snapshotCmd.Flags().StringVarP(&outputFileFlag, "output", "o", defaultOutput, "Output file for snapshot")
	snapshotCmd.Flags().IntVarP(&snapshotTimeoutFlag, "timeout", "t", 60, "Timeout in seconds for the snapshot operation")
}

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Create an etcd snapshot",
	Long:  `Create a point-in-time snapshot of the etcd database`,
	Run: func(cmd *cobra.Command, args []string) {
		// Ensure the snapshot directory exists
		snapshotDir := filepath.Dir(outputFileFlag)
		if err := os.MkdirAll(snapshotDir, 0755); err != nil {
			log.Fatalf("Failed to create snapshot directory %s: %v", snapshotDir, err)
		}

		// Load configuration
		myConfig := config.Config{
			CertFile:         CertFile,
			KeyFile:          KeyFile,
			TrustedCAFile:    TrustedCAFile,
			ListenClientURLs: ListenClientURLs,
			RootUser:         RootUser,
			RootPassword:     RootPassword,
		}

		// Create client
		etcdClient, err := client.NewClient(&myConfig)
		if err != nil {
			log.Fatalf("Error creating etcd client: %v", err)
		}
		defer etcdClient.Close()

		// Perform the snapshot operation
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(snapshotTimeoutFlag)*time.Second)
		defer cancel()

		log.Printf("Creating snapshot at %s", outputFileFlag)

		startTime := time.Now()
		if err = etcdClient.SaveSnapshot(ctx, outputFileFlag); err != nil {
			log.Fatalf("Snapshot failed: %v", err)
		}
		duration := time.Since(startTime)

		// Get file size
		fileInfo, err := os.Stat(outputFileFlag)
		if err != nil {
			log.Printf("Warning: Could not get snapshot file info: %v", err)
		}

		fileSize := "unknown"
		if err == nil {
			fileSize = fmt.Sprintf("%.2f MB", float64(fileInfo.Size())/(1024*1024))
		}

		fmt.Printf("Successfully saved snapshot to %s (Size: %s, Duration: %v)\n", outputFileFlag, fileSize, duration)
	},
}
