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
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aescanero/etcd-node/client"
	"github.com/aescanero/etcd-node/config"
	"github.com/spf13/cobra"
)

var (
	// Check command flags
	checkTimeoutFlag int
	verboseFlag      bool
	jsonOutputFlag   bool
	healthCheckFlag  bool
	alarmCheckFlag   bool
	endpointsFlag    bool
	membersFlag      bool
	leaderCheckFlag  bool
	metricsFlag      bool
	dbSizeFlag       bool
)

func init() {
	checkCmd.Flags().IntVarP(&checkTimeoutFlag, "timeout", "t", 30, "Timeout in seconds for check operations")
	checkCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")
	checkCmd.Flags().BoolVarP(&jsonOutputFlag, "json", "j", false, "Output results in JSON format")
	checkCmd.Flags().BoolVarP(&healthCheckFlag, "health", "", true, "Check cluster health")
	checkCmd.Flags().BoolVarP(&alarmCheckFlag, "alarms", "", true, "Check for cluster alarms")
	checkCmd.Flags().BoolVarP(&endpointsFlag, "endpoints", "", true, "Check endpoints status")
	checkCmd.Flags().BoolVarP(&membersFlag, "members", "", true, "List cluster members")
	checkCmd.Flags().BoolVarP(&leaderCheckFlag, "leader", "", true, "Check for leader presence")
	checkCmd.Flags().BoolVarP(&metricsFlag, "metrics", "", false, "Show metrics (if available)")
	checkCmd.Flags().BoolVarP(&dbSizeFlag, "db-size", "", true, "Show database size")
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check etcd cluster status",
	Long:  `Perform comprehensive checks on etcd cluster health, configuration, and performance`,
	Run: func(cmd *cobra.Command, args []string) {
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

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(checkTimeoutFlag)*time.Second)
		defer cancel()

		// Define data structure to hold check results
		type CheckResults struct {
			Timestamp     string                 `json:"timestamp"`
			Health        *client.EtcdHealth     `json:"health,omitempty"`
			Alarms        []string               `json:"alarms,omitempty"`
			Endpoints     map[string]interface{} `json:"endpoints,omitempty"`
			Members       []interface{}          `json:"members,omitempty"`
			Leader        interface{}            `json:"leader,omitempty"`
			Metrics       map[string]interface{} `json:"metrics,omitempty"`
			DatabaseSize  int64                  `json:"database_size,omitempty"`
			ClusterStatus string                 `json:"cluster_status"`
		}

		results := CheckResults{
			Timestamp:     time.Now().Format(time.RFC3339),
			ClusterStatus: "unknown",
		}

		// Check health if enabled
		if healthCheckFlag {
			if verboseFlag {
				fmt.Println("Checking cluster health...")
			}

			health, err := etcdClient.CheckEtcdHealth(ctx, -1, "")
			if err != nil {
				log.Printf("Health check error: %v", err)
				results.Health = &client.EtcdHealth{
					ErrorMessage: err.Error(),
				}
			} else {
				results.Health = health
				if health.ProcessRunning && health.IsSocketAvailable &&
					!health.IsSocketBlocked && health.DataValid && health.HasLeader {
					results.ClusterStatus = "healthy"
				} else {
					results.ClusterStatus = "unhealthy"
				}
			}
		}

		// Check for alarms if enabled
		if alarmCheckFlag {
			if verboseFlag {
				fmt.Println("Checking for alarms...")
			}

			alarms, err := etcdClient.ListAlarms(ctx)
			if err != nil {
				log.Printf("Alarm check error: %v", err)
				results.Alarms = []string{fmt.Sprintf("Error checking alarms: %v", err)}
			} else {
				results.Alarms = alarms
				if len(alarms) > 0 {
					results.ClusterStatus = "unhealthy"
				}
			}
		}

		// Check endpoints if enabled
		if endpointsFlag {
			if verboseFlag {
				fmt.Println("Checking endpoints...")
			}

			endpoints, err := etcdClient.EndpointsStatus(ctx)
			if err != nil {
				log.Printf("Endpoints check error: %v", err)
				if results.Endpoints == nil {
					results.Endpoints = make(map[string]interface{})
				}
				results.Endpoints["error"] = err.Error()
			} else {
				results.Endpoints = endpoints
			}
		}

		// List members if enabled
		if membersFlag {
			if verboseFlag {
				fmt.Println("Listing cluster members...")
			}

			members, err := etcdClient.MemberList(ctx)
			if err != nil {
				log.Printf("Member list error: %v", err)
				results.Members = []interface{}{map[string]string{"error": err.Error()}}
			} else {
				results.Members = members
			}
		}

		// Check leader if enabled
		if leaderCheckFlag {
			if verboseFlag {
				fmt.Println("Checking for leader...")
			}

			leader, err := etcdClient.GetLeader(ctx)
			if err != nil {
				log.Printf("Leader check error: %v", err)
				results.Leader = map[string]string{"error": err.Error()}
			} else {
				results.Leader = leader
				if leader == nil || leader.(map[string]interface{})["id"] == nil {
					results.ClusterStatus = "unhealthy"
				}
			}
		}

		// Get metrics if enabled
		if metricsFlag {
			if verboseFlag {
				fmt.Println("Collecting metrics...")
			}

			metrics, err := etcdClient.GetMetrics(ctx)
			if err != nil {
				log.Printf("Metrics collection error: %v", err)
				if results.Metrics == nil {
					results.Metrics = make(map[string]interface{})
				}
				results.Metrics["error"] = err.Error()
			} else {
				results.Metrics = metrics
			}
		}

		// Get database size if enabled
		if dbSizeFlag {
			if verboseFlag {
				fmt.Println("Checking database size...")
			}

			size, err := etcdClient.GetDatabaseSize(ctx)
			if err != nil {
				log.Printf("Database size check error: %v", err)
			} else {
				results.DatabaseSize = size
			}
		}

		// Display results
		if jsonOutputFlag {
			// Output in JSON format
			jsonData, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				log.Fatalf("Error marshaling results to JSON: %v", err)
			}
			fmt.Println(string(jsonData))
		} else {
			// Output in human-readable format
			fmt.Printf("Etcd Cluster Check Results (as of %s):\n", results.Timestamp)
			fmt.Printf("Overall Status: %s\n\n", results.ClusterStatus)

			if results.Health != nil {
				fmt.Println("=== Health ===")
				fmt.Printf("Process Running: %v\n", results.Health.ProcessRunning)
				fmt.Printf("Socket Available: %v\n", results.Health.IsSocketAvailable)
				fmt.Printf("Socket Blocked: %v\n", results.Health.IsSocketBlocked)
				fmt.Printf("Data Valid: %v\n", results.Health.DataValid)
				fmt.Printf("Has Leader: %v\n", results.Health.HasLeader)
				if results.Health.ErrorMessage != "" {
					fmt.Printf("Error: %s\n", results.Health.ErrorMessage)
				}
				fmt.Println()
			}

			if results.Alarms != nil {
				fmt.Println("=== Alarms ===")
				if len(results.Alarms) == 0 {
					fmt.Println("No alarms active")
				} else {
					for _, alarm := range results.Alarms {
						fmt.Printf("- %s\n", alarm)
					}
				}
				fmt.Println()
			}

			if results.Members != nil {
				fmt.Println("=== Cluster Members ===")
				for i, member := range results.Members {
					if m, ok := member.(map[string]interface{}); ok {
						fmt.Printf("Member %d:\n", i+1)
						if id, ok := m["id"]; ok {
							fmt.Printf("  ID: %v\n", id)
						}
						if name, ok := m["name"]; ok {
							fmt.Printf("  Name: %v\n", name)
						}
						if peerURLs, ok := m["peerURLs"]; ok {
							fmt.Printf("  Peer URLs: %v\n", peerURLs)
						}
						if clientURLs, ok := m["clientURLs"]; ok {
							fmt.Printf("  Client URLs: %v\n", clientURLs)
						}
						fmt.Println()
					}
				}
			}

			if results.Leader != nil {
				fmt.Println("=== Leader Information ===")
				if m, ok := results.Leader.(map[string]interface{}); ok {
					if id, ok := m["id"]; ok {
						fmt.Printf("Leader ID: %v\n", id)
					}
					if name, ok := m["name"]; ok {
						fmt.Printf("Leader Name: %v\n", name)
					}
				}
				fmt.Println()
			}

			if results.DatabaseSize > 0 {
				fmt.Println("=== Database ===")
				fmt.Printf("Size: %.2f MB\n", float64(results.DatabaseSize)/(1024*1024))
				fmt.Println()
			}

			// Print verbose metrics only if requested and available
			if verboseFlag && results.Metrics != nil && len(results.Metrics) > 0 {
				fmt.Println("=== Metrics ===")
				printMap(results.Metrics, "")
				fmt.Println()
			}
		}
	},
}

// Helper function to print nested maps recursively
func printMap(m map[string]interface{}, indent string) {
	for k, v := range m {
		if nestedMap, ok := v.(map[string]interface{}); ok {
			fmt.Printf("%s%s:\n", indent, k)
			printMap(nestedMap, indent+"  ")
		} else {
			fmt.Printf("%s%s: %v\n", indent, k, v)
		}
	}
}
