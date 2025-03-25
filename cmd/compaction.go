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
	"strconv"
	"time"

	"github.com/aescanero/etcd-node/client"
	"github.com/aescanero/etcd-node/config"
	"github.com/spf13/cobra"
)

var (
	// Compaction flags
	revisionFlag string
	timeoutFlag  int
	dryRunFlag   bool
	physicalFlag bool
)

func init() {
	compactionCmd.Flags().StringVarP(&revisionFlag, "revision", "r", "", "Revision to compact to (default is '0' which means compact to the most recent revision)")
	compactionCmd.Flags().IntVarP(&timeoutFlag, "timeout", "t", 30, "Timeout in seconds for the compaction operation")
	compactionCmd.Flags().BoolVarP(&dryRunFlag, "dry-run", "d", false, "Show what would be compacted without actually compacting")
	compactionCmd.Flags().BoolVarP(&physicalFlag, "physical", "p", false, "Perform physical compaction after logical compaction")
}

var compactionCmd = &cobra.Command{
	Use:   "compaction",
	Short: "Compact etcd storage",
	Long:  `Compact etcd storage to free up space by removing historical revisions`,
	Run: func(cmd *cobra.Command, args []string) {
		// Parse revision
		var revision int64 = 0
		var err error
		if revisionFlag != "" {
			revision, err = strconv.ParseInt(revisionFlag, 10, 64)
			if err != nil {
				log.Fatalf("Invalid revision number: %v", err)
			}
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

		// If revision is 0, get the current revision
		if revision == 0 {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutFlag)*time.Second)
			defer cancel()

			status, err := etcdClient.GetCurrentRevision(ctx)
			if err != nil {
				log.Fatalf("Error getting current revision: %v", err)
			}
			revision = status
			log.Printf("Current revision is %d", revision)
		}

		if dryRunFlag {
			log.Printf("Dry run: Would compact to revision %d", revision)
			return
		}

		// Perform the compaction
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutFlag)*time.Second)
		defer cancel()

		log.Printf("Starting compaction to revision %d", revision)
		if err = etcdClient.Compact(ctx, revision, physicalFlag); err != nil {
			log.Fatalf("Compaction failed: %v", err)
		}

		fmt.Printf("Successfully compacted etcd to revision %d\n", revision)
	},
}
