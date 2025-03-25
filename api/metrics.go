package api

import (
	"context"
	"time"

	"github.com/aescanero/etcd-node/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector maneja las métricas de Prometheus para etcd
type MetricsCollector struct {
	etcdClient *client.Client
	processID  int
	dataDir    string

	// Métricas básicas
	up              prometheus.Gauge
	processRunning  prometheus.Gauge
	socketAvailable prometheus.Gauge
	socketBlocked   prometheus.Gauge
	dataValid       prometheus.Gauge
	hasLeader       prometheus.Gauge
	isLeader        prometheus.Gauge
	clusterSize     prometheus.Gauge
	databaseSize    prometheus.Gauge

	// Métricas de operaciones
	operationsTotal   *prometheus.CounterVec
	operationErrors   *prometheus.CounterVec
	operationDuration *prometheus.HistogramVec

	// Métricas de compactación
	lastCompactionTime prometheus.Gauge
	compactionTotal    prometheus.Counter
}

// NewMetricsCollector crea un nuevo colector de métricas
func NewMetricsCollector(etcdClient *client.Client, processID int, dataDir string) *MetricsCollector {
	return &MetricsCollector{
		etcdClient: etcdClient,
		processID:  processID,
		dataDir:    dataDir,

		// Registro de métricas
		up: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "etcd_node_up",
			Help: "1 if the etcd node is up and healthy, 0 otherwise",
		}),

		processRunning: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "etcd_node_process_running",
			Help: "1 if the etcd process is running, 0 otherwise",
		}),

		socketAvailable: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "etcd_node_socket_available",
			Help: "1 if the etcd socket is available, 0 otherwise",
		}),

		socketBlocked: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "etcd_node_socket_blocked",
			Help: "1 if the etcd socket is blocked, 0 otherwise",
		}),

		dataValid: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "etcd_node_data_valid",
			Help: "1 if the etcd data is valid, 0 otherwise",
		}),

		hasLeader: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "etcd_node_has_leader",
			Help: "1 if the etcd cluster has a leader, 0 otherwise",
		}),

		isLeader: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "etcd_node_is_leader",
			Help: "1 if this etcd node is the leader, 0 otherwise",
		}),

		clusterSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "etcd_node_cluster_size",
			Help: "Number of members in the etcd cluster",
		}),

		databaseSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "etcd_node_database_size_bytes",
			Help: "Size of the etcd database in bytes",
		}),

		operationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "etcd_node_operations_total",
				Help: "Total number of etcd operations by type",
			},
			[]string{"operation"},
		),

		operationErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "etcd_node_operation_errors_total",
				Help: "Total number of etcd operation errors by type",
			},
			[]string{"operation"},
		),

		operationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "etcd_node_operation_duration_seconds",
				Help:    "Duration of etcd operations in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation"},
		),

		lastCompactionTime: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "etcd_node_last_compaction_timestamp",
			Help: "Timestamp of the last etcd compaction",
		}),

		compactionTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "etcd_node_compactions_total",
			Help: "Total number of etcd compactions",
		}),
	}
}

// UpdateMetrics actualiza todas las métricas
func (c *MetricsCollector) UpdateMetrics() {
	// Realizar comprobación de salud
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := c.etcdClient.CheckEtcdHealth(ctx, c.processID, c.dataDir)

	// Actualizar métricas básicas
	if err != nil || health == nil {
		c.up.Set(0)
		return
	}

	// Actualizar métricas de disponibilidad
	setGaugeFromBool(c.processRunning, health.ProcessRunning)
	setGaugeFromBool(c.socketAvailable, health.IsSocketAvailable)
	setGaugeFromBool(c.socketBlocked, health.IsSocketBlocked)
	setGaugeFromBool(c.dataValid, health.DataValid)
	setGaugeFromBool(c.hasLeader, health.HasLeader)

	// Si todas las métricas clave son positivas, el nodo está "up"
	nodeIsUp := health.ProcessRunning && health.IsSocketAvailable && !health.IsSocketBlocked && health.DataValid && health.HasLeader
	if nodeIsUp {
		c.up.Set(1)
	} else {
		c.up.Set(0)
	}

	// Obtener métricas adicionales si el nodo está disponible
	if nodeIsUp {
		c.collectAdditionalMetrics(ctx)
	}
}

// collectAdditionalMetrics recopila métricas adicionales que requieren consultas a etcd
func (c *MetricsCollector) collectAdditionalMetrics(ctx interface{}) {
	// Esta función debe implementarse para recopilar métricas
	// adicionales como tamaño de la base de datos, estado del liderazgo, etc.

	// Ejemplo (pseudo-código):
	// status, err := c.etcdClient.GetStatus(ctx)
	// if err == nil {
	//     c.isLeader.Set(boolToFloat64(status.IsLeader))
	//     c.clusterSize.Set(float64(status.MemberCount))
	//     c.databaseSize.Set(float64(status.DbSize))
	// }
}

// TrackOperation registra una operación y su duración
func (c *MetricsCollector) TrackOperation(operation string, duration time.Duration, success bool) {
	c.operationsTotal.WithLabelValues(operation).Inc()

	if !success {
		c.operationErrors.WithLabelValues(operation).Inc()
	}

	c.operationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// TrackCompaction registra una operación de compactación
func (c *MetricsCollector) TrackCompaction() {
	c.compactionTotal.Inc()
	c.lastCompactionTime.Set(float64(time.Now().Unix()))
}

// Helper para convertir bool a float64 para métricas
func setGaugeFromBool(gauge prometheus.Gauge, value bool) {
	if value {
		gauge.Set(1)
	} else {
		gauge.Set(0)
	}
}
