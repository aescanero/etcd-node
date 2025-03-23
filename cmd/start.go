/*Copyright [2023] [Alejandro Escanero Blanco <aescanero@disasterproject.com>]

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
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aescanero/etcd-node/api"
	"github.com/aescanero/etcd-node/client"
	"github.com/aescanero/etcd-node/config"
	"github.com/aescanero/etcd-node/service"
	"github.com/aescanero/etcd-node/utils"
	"github.com/spf13/cobra"
)

var (
	app                service.EtcdLauncher
	CertFile           string = utils.GetEnv("ETCD_CERT_FILE", "tls.crt")
	KeyFile            string = utils.GetEnv("ETCD_KEY_FILE", "tls.key")
	TrustedCAFile      string = utils.GetEnv("ETCD_TRUSTED_CA_FILE", "ca.crt")
	PeerCertFile       string = utils.GetEnv("ETCD_PEER_CERT_FILE", "tls.crt")
	PeerKeyFile        string = utils.GetEnv("ETCD_PEER_KEY_FILE", "tls.key")
	PeerTrustedCAFile  string = utils.GetEnv("ETCD_PEER_TRUSTED_CA_FILE", "ca.crt")
	ClientCertAuth     bool   = utils.GetEnv("ETCD_CLIENT_CERT_AUTH", "true") == "true"
	PeerClientCertAuth bool   = utils.GetEnv("ETCD_PEER_CLIENT_CERT_AUTH", "true") == "true"

	// Node Configuration
	Name                     string = utils.GetEnv("ETCD_NAME", "etcd-0")
	InitialClusterState      string = utils.GetEnv("ETCD_INITIAL_CLUSTER_STATE", "new")
	InitialClusterToken      string = utils.GetEnv("ETCD_INITIAL_CLUSTER_TOKEN", "etcd-cluster-1")
	InitialCluster           string = utils.GetEnv("ETCD_INITIAL_CLUSTER", "etcd-0=http://etcd-0.etcd-headless.etcd.svc.cluster.local:2380,etcd-1=http://etcd-1.etcd-headless.etcd.svc.cluster.local:2380,etcd-2=http://etcd-2.etcd-headless.etcd.svc.cluster.local:2380")
	InitialAdvertisePeerURLs string = utils.GetEnv("ETCD_INITIAL_ADVERTISE_PEER_URLS", "http://$(ETCD_NAME).etcd-headless.etcd.svc.cluster.local:2380")
	ListenPeerURLs           string = utils.GetEnv("ETCD_LISTEN_PEER_URLS", "http://0.0.0.0:2380")
	AdvertiseClientURLs      string = utils.GetEnv("ETCD_ADVERTISE_CLIENT_URLS", "http://$(ETCD_NAME).etcd-headless.etcd.svc.cluster.local:2379")
	ListenClientURLs         string = utils.GetEnv("ETCD_LISTEN_CLIENT_URLS", "http://0.0.0.0:2380")

	// Performance Configuration
	AutoCompactionRetention string = utils.GetEnv("ETCD_AUTO_COMPACTION_RETENTION", "1")
	QuotaBackendBytes       string = utils.GetEnv("ETCD_QUOTA_BACKEND_BYTES", "8589934592")

	// Add root user configuration
	RootUser     string = utils.GetEnv("ETCD_ROOT_USER", "root")
	RootPassword string = utils.GetEnv("ETCD_ROOT_PASSWORD", "root")

	ReadUser     string = utils.GetEnv("ETCD_READ_USER", "read")
	ReadPassword string = utils.GetEnv("ETCD_READ_PASSWORD", "read")

	// Variables for health API
	HealthAPIEnabled  bool   = utils.GetEnv("ETCD_HEALTH_API_ENABLED", "true") == "true"
	HealthAPIAddress  string = utils.GetEnv("ETCD_HEALTH_API_ADDRESS", ":8080")
	HealthCheckPeriod int    = utils.GetEnvAsInt("ETCD_HEALTH_CHECK_PERIOD", 10) // seconds
	EtcdDataDir       string = utils.GetEnv("ETCD_DATA_DIR", "/var/lib/etcd")
)

func init() {
	startCmd.Flags().StringVarP(&CertFile, "cert-file", "", CertFile, "ETCD cert file path")
	startCmd.Flags().StringVarP(&KeyFile, "key-file", "", KeyFile, "ETCD key file path")
	startCmd.Flags().StringVarP(&TrustedCAFile, "trusted-ca-file", "", TrustedCAFile, "ETCD trusted CA file path")
	startCmd.Flags().StringVarP(&PeerCertFile, "peer-cert-file", "", PeerCertFile, "ETCD peer cert file path")
	startCmd.Flags().StringVarP(&PeerKeyFile, "peer-key-file", "", PeerKeyFile, "ETCD peer key file path")
	startCmd.Flags().StringVarP(&PeerTrustedCAFile, "peer-trusted-ca-file", "", PeerTrustedCAFile, "ETCD peer trusted CA file path")
	startCmd.Flags().BoolVarP(&ClientCertAuth, "client-cert-auth", "", ClientCertAuth, "Enable client cert auth")
	startCmd.Flags().BoolVarP(&PeerClientCertAuth, "peer-client-cert-auth", "", PeerClientCertAuth, "Enable peer client cert auth")

	startCmd.Flags().StringVarP(&Name, "name", "", Name, "ETCD node name")
	startCmd.Flags().StringVarP(&InitialClusterState, "initial-cluster-state", "", InitialClusterState, "Initial cluster state")
	startCmd.Flags().StringVarP(&InitialClusterToken, "initial-cluster-token", "", InitialClusterToken, "Initial cluster token")
	startCmd.Flags().StringVarP(&InitialCluster, "initial-cluster", "", InitialCluster, "Initial cluster configuration")
	startCmd.Flags().StringVarP(&InitialAdvertisePeerURLs, "initial-advertise-peer-urls", "", InitialAdvertisePeerURLs, "Initial advertise peer URLs")
	startCmd.Flags().StringVarP(&ListenPeerURLs, "listen-peer-urls", "", ListenPeerURLs, "Listen peer URLs")
	startCmd.Flags().StringVarP(&AdvertiseClientURLs, "advertise-client-urls", "", AdvertiseClientURLs, "Advertise client URLs")
	startCmd.Flags().StringVarP(&ListenClientURLs, "listen-client-urls", "", ListenClientURLs, "Listen client URLs")

	startCmd.Flags().StringVarP(&AutoCompactionRetention, "auto-compaction-retention", "", AutoCompactionRetention, "Auto compaction retention")
	startCmd.Flags().StringVarP(&QuotaBackendBytes, "quota-backend-bytes", "", QuotaBackendBytes, "Quota backend bytes")

	// Add root user flag
	startCmd.Flags().StringVarP(&RootUser, "root-user", "", RootUser, "ETCD root user name")
	// Add root user password flag
	startCmd.Flags().StringVarP(&RootPassword, "root-password", "", RootPassword, "ETCD root user password")
	startCmd.Flags().StringVarP(&ReadUser, "read-user", "", ReadUser, "ETCD read user name")
	startCmd.Flags().StringVarP(&ReadPassword, "read-password", "", ReadPassword, "ETCD read user password")
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Etcd Node",
	Long:  `Start Etcd Node`,
	Run: func(cmd *cobra.Command, args []string) {

		// Create config based on command line flags and environment variables
		myConfig := config.Config{
			// TLS Configuration
			CertFile:           CertFile,
			KeyFile:            KeyFile,
			TrustedCAFile:      TrustedCAFile,
			PeerCertFile:       PeerCertFile,
			PeerKeyFile:        PeerKeyFile,
			PeerTrustedCAFile:  PeerTrustedCAFile,
			ClientCertAuth:     ClientCertAuth,
			PeerClientCertAuth: PeerClientCertAuth,

			// Node Configuration
			Name:                     Name,
			InitialClusterState:      InitialClusterState,
			InitialClusterToken:      InitialClusterToken,
			InitialCluster:           InitialCluster,
			InitialAdvertisePeerURLs: InitialAdvertisePeerURLs,
			ListenPeerURLs:           ListenPeerURLs,
			AdvertiseClientURLs:      AdvertiseClientURLs,
			ListenClientURLs:         ListenClientURLs,

			// Performance Configuration
			AutoCompactionRetention: AutoCompactionRetention,
			QuotaBackendBytes:       QuotaBackendBytes,

			RootUser:     RootUser,
			RootPassword: RootPassword,
			ReadUser:     ReadUser,
			ReadPassword: ReadPassword,
		}

		// Start etcd server
		app := service.NewLauncher(myConfig)
		if err := app.Start(); err != nil {
			log.Fatalf("Error starting etcd: %v", err)
		}
		log.Println("Etcd started successfully")

		// Get etcd process ID for health monitoring
		etcdPid := app.GetProcessID()

		// Create bootstrap client
		client, err := client.NewClient(&myConfig)
		if err != nil {
			log.Fatalf("Error creating bootstrap client: %v", err)
		}
		log.Println("Bootstrap client created")
		defer client.Close()

		// Check if bootstrap is needed using non-blocking method
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		needsBootstrap, err := client.NeedsBootstrap(ctx, etcdPid)
		cancel()

		if err != nil {
			log.Printf("Bootstrap check warning: %v", err)
		}

		if needsBootstrap {
			bootstrapCtx, bootstrapCancel := context.WithTimeout(context.Background(), 60*time.Second)
			if err := client.Bootstrap(bootstrapCtx); err != nil {
				bootstrapCancel()
				log.Fatalf("Error during bootstrap: %v", err)
			}
			bootstrapCancel()
			log.Println("Bootstrap completed successfully")
		} else {
			log.Println("Server is already configured")
		}

		// Start health API server if enabled
		var healthServer *api.HealthServer
		if HealthAPIEnabled {
			healthServer = api.NewHealthServer(client, &myConfig, etcdPid, EtcdDataDir)
			healthServer.StartMonitoring(time.Duration(HealthCheckPeriod) * time.Second)
			healthServer.StartAsync(HealthAPIAddress)
			log.Printf("Health API server started on %s", HealthAPIAddress)
		}

		statusChan := app.Monitor(5 * time.Second)

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		done := make(chan struct{})

		go func() {
			<-sigChan
			log.Println("Señal de terminación recibida, deteniendo etcd...")
			if err := app.Stop(); err != nil {
				log.Printf("Error deteniendo etcd: %v", err)
			}
			close(done)
		}()

		for status := range statusChan {
			log.Printf("Estado del proceso etcd - PID: %d, Running: %v, Exit Code: %d",
				status.Pid, status.Running, status.ExitCode)

			if !status.Running {
				log.Printf("El proceso etcd se ha detenido con código de salida: %d", status.ExitCode)
				return
			}

			// Verificar si debemos terminar
			select {
			case <-done:
				return
			default:
				// Continuar monitoreando
			}
		}
	},
}
