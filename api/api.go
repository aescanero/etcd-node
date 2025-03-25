package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/aescanero/etcd-node/client"
	"github.com/aescanero/etcd-node/config"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HealthServer proporciona una API HTTP para monitorizar la salud de etcd
type HealthServer struct {
	etcdClient       *client.Client
	router           *gin.Engine
	config           *config.Config
	processID        int
	dataDir          string
	currentHealth    *client.EtcdHealth
	mutex            sync.RWMutex
	updateTicker     *time.Ticker
	stopChan         chan struct{}
	metricsCollector *MetricsCollector // Añadido: colector de métricas
}

// NewHealthServer crea un nuevo servidor de API de salud
func NewHealthServer(etcdClient *client.Client, cfg *config.Config, processID int, dataDir string) *HealthServer {
	router := gin.Default()

	// Crear el colector de métricas
	metricsCollector := NewMetricsCollector(etcdClient, processID, dataDir)

	server := &HealthServer{
		etcdClient:       etcdClient,
		router:           router,
		config:           cfg,
		processID:        processID,
		dataDir:          dataDir,
		metricsCollector: metricsCollector, // Añadido: asignar el colector
		currentHealth: &client.EtcdHealth{
			ProcessRunning:    false,
			IsSocketAvailable: false,
			IsSocketBlocked:   false,
			DataValid:         false,
			HasLeader:         false,
			ErrorMessage:      "Health check not yet performed",
		},
		stopChan: make(chan struct{}),
	}

	// Configurar rutas
	server.setupRoutes()

	return server
}

// setupRoutes sets up the API endpoints
func (s *HealthServer) setupRoutes() {
	// GET /health returns the current health status
	s.router.GET("/health", s.getHealth)

	// POST /health/refresh triggers a manual health check refresh
	s.router.POST("/health/refresh", s.refreshHealth)

	// GET /status returns status details including the EtcdHealth, plus system metrics
	s.router.GET("/status", s.getStatus)

	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

// getHealth returns the current health status
func (s *HealthServer) getHealth(c *gin.Context) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	c.JSON(http.StatusOK, s.currentHealth)
}

// refreshHealth manually triggers a health check update
func (s *HealthServer) refreshHealth(c *gin.Context) {
	s.updateHealthStatus()
	c.JSON(http.StatusOK, gin.H{
		"message": "Health status refreshed",
		"health":  s.currentHealth,
	})
}

// getStatus returns additional system status information
func (s *HealthServer) getStatus(c *gin.Context) {
	s.mutex.RLock()
	health := s.currentHealth
	s.mutex.RUnlock()

	// Get additional system metrics if needed
	uptime := "Unknown" // Placeholder for actual uptime calculation

	c.JSON(http.StatusOK, gin.H{
		"health":      health,
		"uptime":      uptime,
		"processID":   s.processID,
		"dataDir":     s.dataDir,
		"clusterName": s.config.InitialClusterToken,
		"nodeName":    s.config.Name,
		"timestamp":   time.Now().Format(time.RFC3339),
	})
}

// updateHealthStatus performs a health check and updates the current status
func (s *HealthServer) updateHealthStatus() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := s.etcdClient.CheckEtcdHealth(ctx, s.processID, s.dataDir)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err != nil {
		s.currentHealth = &client.EtcdHealth{
			ProcessRunning:    false,
			IsSocketAvailable: false,
			IsSocketBlocked:   true,
			DataValid:         false,
			HasLeader:         false,
			ErrorMessage:      err.Error(),
		}
		return
	}

	s.currentHealth = health
	s.metricsCollector.UpdateMetrics()
}

// StartMonitoring starts periodic health checks
func (s *HealthServer) StartMonitoring(interval time.Duration) {
	// Perform an initial health check
	s.updateHealthStatus()

	// Start periodic checks
	s.updateTicker = time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-s.updateTicker.C:
				s.updateHealthStatus()
			case <-s.stopChan:
				s.updateTicker.Stop()
				return
			}
		}
	}()
}

// StopMonitoring stops periodic health checks
func (s *HealthServer) StopMonitoring() {
	close(s.stopChan)
}

// Start begins serving the health API
func (s *HealthServer) Start(address string) error {
	return s.router.Run(address)
}

// StartAsync begins serving the health API in a goroutine
func (s *HealthServer) StartAsync(address string) {
	go func() {
		if err := s.Start(address); err != nil && err != http.ErrServerClosed {
			// Log error but don't crash
			// log.Printf("Health API server error: %v", err)
		}
	}()
}
