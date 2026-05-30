package monitoring

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zyy125/im-system/config"
	"github.com/zyy125/im-system/internal/repository"
	"github.com/zyy125/im-system/internal/ws"
)

type Runtime struct {
	registry *prometheus.Registry

	httpInflight *prometheus.GaugeVec
	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec

	redisOperations *prometheus.CounterVec
	redisDuration   *prometheus.HistogramVec
	redisMetrics    repository.RedisMetricsRecorder
	hubAttached     bool
}

func NewRuntime(hub *ws.Hub, sqlDB *sql.DB) *Runtime {
	registry := prometheus.NewRegistry()
	r := &Runtime{
		registry: registry,
		httpInflight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "im",
				Subsystem: "http",
				Name:      "requests_in_flight",
				Help:      "Current number of in-flight HTTP requests.",
			},
			[]string{"method", "route"},
		),
		httpRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "im",
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests handled by route, method, and status.",
			},
			[]string{"method", "route", "status"},
		),
		httpDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "im",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds by route and method.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "route"},
		),
		redisOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "im",
				Subsystem: "redis",
				Name:      "operations_total",
				Help:      "Total number of Redis operations performed by business module, operation, and result.",
			},
			[]string{"module", "op", "result"},
		),
		redisDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "im",
				Subsystem: "redis",
				Name:      "operation_duration_seconds",
				Help:      "Redis operation duration in seconds by business module and operation.",
				Buckets:   []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
			[]string{"module", "op"},
		),
	}
	r.redisMetrics = &redisMetricsRecorder{
		operations: r.redisOperations,
		duration:   r.redisDuration,
	}

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{Namespace: "im"}),
		r.httpInflight,
		r.httpRequests,
		r.httpDuration,
		r.redisOperations,
		r.redisDuration,
	)
	if hub != nil {
		registry.MustRegister(newHubCollector(hub))
		r.hubAttached = true
	}
	if sqlDB != nil {
		registry.MustRegister(newDBStatsCollector(sqlDB))
	}

	return r
}

func (r *Runtime) RedisMetrics() repository.RedisMetricsRecorder {
	if r == nil || r.redisMetrics == nil {
		return noopRedisMetricsRecorder{}
	}
	return r.redisMetrics
}

func (r *Runtime) AttachHub(hub *ws.Hub) {
	if r == nil || r.registry == nil || hub == nil || r.hubAttached {
		return
	}
	r.registry.MustRegister(newHubCollector(hub))
	r.hubAttached = true
}

func (r *Runtime) HTTPMiddleware() gin.HandlerFunc {
	if r == nil {
		return nil
	}

	return func(c *gin.Context) {
		method := c.Request.Method
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		r.httpInflight.WithLabelValues(method, route).Inc()
		start := time.Now()
		defer func() {
			r.httpInflight.WithLabelValues(method, route).Dec()
			if finalRoute := c.FullPath(); finalRoute != "" {
				route = finalRoute
			}
			status := strconv.Itoa(c.Writer.Status())
			r.httpRequests.WithLabelValues(method, route, status).Inc()
			r.httpDuration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
		}()

		c.Next()
	}
}

func (r *Runtime) MetricsHandler() http.Handler {
	if r == nil || r.registry == nil {
		return promhttp.Handler()
	}
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{})
}

func NewServer(cfg config.Monitor, runtime *Runtime, hub *ws.Hub) *http.Server {
	if !cfg.EnableMetrics && !cfg.EnableDebugHub && !cfg.EnablePprof {
		return nil
	}

	mux := http.NewServeMux()
	if cfg.EnableMetrics && runtime != nil {
		mux.Handle("/metrics", runtime.MetricsHandler())
	}
	if cfg.EnableDebugHub && hub != nil {
		mux.HandleFunc("/debug/hub", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(hub.Snapshot())
		})
	}
	if cfg.EnablePprof {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	return &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
	}
}

type hubCollector struct {
	hub *ws.Hub

	users                *prometheus.Desc
	connections          *prometheus.Desc
	registerQueueLen     *prometheus.Desc
	registerQueueCap     *prometheus.Desc
	unregisterQueueLen   *prometheus.Desc
	unregisterQueueCap   *prometheus.Desc
	forwardQueueLen      *prometheus.Desc
	forwardQueueCap      *prometheus.Desc
	lifecycleForwardLen  *prometheus.Desc
	lifecycleForwardCap  *prometheus.Desc
	markSyncQueueLen     *prometheus.Desc
	markSyncQueueCap     *prometheus.Desc
	bootstrappedQueueLen *prometheus.Desc
	bootstrappedQueueCap *prometheus.Desc

	registerTotal              *prometheus.Desc
	unregisterTotal            *prometheus.Desc
	forwardQueueFullTotal      *prometheus.Desc
	pendingQueueFullTotal      *prometheus.Desc
	sendQueueFullTotal         *prometheus.Desc
	syncRequiredEmittedTotal   *prometheus.Desc
	bootstrapTotal             *prometheus.Desc
	bootstrapFailedTotal       *prometheus.Desc
	syncDeliveryFailCloseTotal *prometheus.Desc
}

func newHubCollector(hub *ws.Hub) *hubCollector {
	return &hubCollector{
		hub:                   hub,
		users:                 prometheus.NewDesc("im_hub_users", "Current number of distinct online users tracked by the Hub.", nil, nil),
		connections:           prometheus.NewDesc("im_hub_connections", "Current number of active WebSocket connections tracked by the Hub.", nil, nil),
		registerQueueLen:      prometheus.NewDesc("im_hub_register_queue_length", "Current number of pending client register requests.", nil, nil),
		registerQueueCap:      prometheus.NewDesc("im_hub_register_queue_capacity", "Configured capacity of the client register queue.", nil, nil),
		unregisterQueueLen:    prometheus.NewDesc("im_hub_unregister_queue_length", "Current number of pending client unregister requests.", nil, nil),
		unregisterQueueCap:    prometheus.NewDesc("im_hub_unregister_queue_capacity", "Configured capacity of the client unregister queue.", nil, nil),
		forwardQueueLen:       prometheus.NewDesc("im_hub_forward_queue_length", "Current number of pending Hub forward messages.", nil, nil),
		forwardQueueCap:       prometheus.NewDesc("im_hub_forward_queue_capacity", "Configured capacity of the Hub forward queue.", nil, nil),
		lifecycleForwardLen:   prometheus.NewDesc("im_hub_lifecycle_forward_queue_length", "Current number of pending lifecycle forward messages.", nil, nil),
		lifecycleForwardCap:   prometheus.NewDesc("im_hub_lifecycle_forward_queue_capacity", "Configured capacity of the lifecycle forward queue.", nil, nil),
		markSyncQueueLen:      prometheus.NewDesc("im_hub_mark_sync_queue_length", "Current number of pending mark-sync requests.", nil, nil),
		markSyncQueueCap:      prometheus.NewDesc("im_hub_mark_sync_queue_capacity", "Configured capacity of the mark-sync queue.", nil, nil),
		bootstrappedQueueLen:  prometheus.NewDesc("im_hub_bootstrapped_queue_length", "Current number of pending client bootstrap results.", nil, nil),
		bootstrappedQueueCap:  prometheus.NewDesc("im_hub_bootstrapped_queue_capacity", "Configured capacity of the client bootstrap result queue.", nil, nil),
		registerTotal:         prometheus.NewDesc("im_hub_register_total", "Total number of client registrations handled by the Hub.", nil, nil),
		unregisterTotal:       prometheus.NewDesc("im_hub_unregister_total", "Total number of client unregistrations handled by the Hub.", nil, nil),
		forwardQueueFullTotal: prometheus.NewDesc("im_hub_forward_queue_full_total", "Total number of forward messages dropped because the Hub forward queue was full.", nil, nil),
		pendingQueueFullTotal: prometheus.NewDesc("im_hub_pending_queue_full_total", "Total number of times a connection pending queue filled up before the client became ready.", nil, nil),
		sendQueueFullTotal:    prometheus.NewDesc("im_hub_send_queue_full_total", "Total number of times a client send queue was full during live delivery.", nil, nil),
		syncRequiredEmittedTotal: prometheus.NewDesc(
			"im_hub_sync_required_emitted_total",
			"Total number of sync_required events emitted by the Hub.",
			nil,
			nil,
		),
		bootstrapTotal:             prometheus.NewDesc("im_hub_bootstrap_total", "Total number of client bootstrap attempts.", nil, nil),
		bootstrapFailedTotal:       prometheus.NewDesc("im_hub_bootstrap_failed_total", "Total number of failed client bootstrap attempts.", nil, nil),
		syncDeliveryFailCloseTotal: prometheus.NewDesc("im_hub_sync_delivery_fail_close_total", "Total number of connections closed because sync_required could not be delivered.", nil, nil),
	}
}

func (c *hubCollector) Describe(ch chan<- *prometheus.Desc) {
	descs := []*prometheus.Desc{
		c.users,
		c.connections,
		c.registerQueueLen,
		c.registerQueueCap,
		c.unregisterQueueLen,
		c.unregisterQueueCap,
		c.forwardQueueLen,
		c.forwardQueueCap,
		c.lifecycleForwardLen,
		c.lifecycleForwardCap,
		c.markSyncQueueLen,
		c.markSyncQueueCap,
		c.bootstrappedQueueLen,
		c.bootstrappedQueueCap,
		c.registerTotal,
		c.unregisterTotal,
		c.forwardQueueFullTotal,
		c.pendingQueueFullTotal,
		c.sendQueueFullTotal,
		c.syncRequiredEmittedTotal,
		c.bootstrapTotal,
		c.bootstrapFailedTotal,
		c.syncDeliveryFailCloseTotal,
	}
	for _, desc := range descs {
		ch <- desc
	}
}

func (c *hubCollector) Collect(ch chan<- prometheus.Metric) {
	if c == nil || c.hub == nil {
		return
	}

	snapshot := c.hub.Snapshot()
	ch <- prometheus.MustNewConstMetric(c.users, prometheus.GaugeValue, float64(snapshot.Users))
	ch <- prometheus.MustNewConstMetric(c.connections, prometheus.GaugeValue, float64(snapshot.Connections))
	ch <- prometheus.MustNewConstMetric(c.registerQueueLen, prometheus.GaugeValue, float64(snapshot.RegisterQueueLen))
	ch <- prometheus.MustNewConstMetric(c.registerQueueCap, prometheus.GaugeValue, float64(snapshot.RegisterQueueCap))
	ch <- prometheus.MustNewConstMetric(c.unregisterQueueLen, prometheus.GaugeValue, float64(snapshot.UnregisterQueueLen))
	ch <- prometheus.MustNewConstMetric(c.unregisterQueueCap, prometheus.GaugeValue, float64(snapshot.UnregisterQueueCap))
	ch <- prometheus.MustNewConstMetric(c.forwardQueueLen, prometheus.GaugeValue, float64(snapshot.ForwardQueueLen))
	ch <- prometheus.MustNewConstMetric(c.forwardQueueCap, prometheus.GaugeValue, float64(snapshot.ForwardQueueCap))
	ch <- prometheus.MustNewConstMetric(c.lifecycleForwardLen, prometheus.GaugeValue, float64(snapshot.LifecycleForwardLen))
	ch <- prometheus.MustNewConstMetric(c.lifecycleForwardCap, prometheus.GaugeValue, float64(snapshot.LifecycleForwardCap))
	ch <- prometheus.MustNewConstMetric(c.markSyncQueueLen, prometheus.GaugeValue, float64(snapshot.MarkSyncQueueLen))
	ch <- prometheus.MustNewConstMetric(c.markSyncQueueCap, prometheus.GaugeValue, float64(snapshot.MarkSyncQueueCap))
	ch <- prometheus.MustNewConstMetric(c.bootstrappedQueueLen, prometheus.GaugeValue, float64(snapshot.BootstrappedQueueLen))
	ch <- prometheus.MustNewConstMetric(c.bootstrappedQueueCap, prometheus.GaugeValue, float64(snapshot.BootstrappedQueueCap))

	ch <- prometheus.MustNewConstMetric(c.registerTotal, prometheus.CounterValue, float64(snapshot.RegisterTotal))
	ch <- prometheus.MustNewConstMetric(c.unregisterTotal, prometheus.CounterValue, float64(snapshot.UnregisterTotal))
	ch <- prometheus.MustNewConstMetric(c.forwardQueueFullTotal, prometheus.CounterValue, float64(snapshot.ForwardQueueFullTotal))
	ch <- prometheus.MustNewConstMetric(c.pendingQueueFullTotal, prometheus.CounterValue, float64(snapshot.PendingQueueFullTotal))
	ch <- prometheus.MustNewConstMetric(c.sendQueueFullTotal, prometheus.CounterValue, float64(snapshot.SendQueueFullTotal))
	ch <- prometheus.MustNewConstMetric(c.syncRequiredEmittedTotal, prometheus.CounterValue, float64(snapshot.SyncRequiredEmittedTotal))
	ch <- prometheus.MustNewConstMetric(c.bootstrapTotal, prometheus.CounterValue, float64(snapshot.BootstrapTotal))
	ch <- prometheus.MustNewConstMetric(c.bootstrapFailedTotal, prometheus.CounterValue, float64(snapshot.BootstrapFailedTotal))
	ch <- prometheus.MustNewConstMetric(c.syncDeliveryFailCloseTotal, prometheus.CounterValue, float64(snapshot.SyncDeliveryFailCloseTotal))
}

type dbStatsCollector struct {
	sqlDB *sql.DB

	maxOpenConnections *prometheus.Desc
	openConnections    *prometheus.Desc
	inUseConnections   *prometheus.Desc
	idleConnections    *prometheus.Desc
	waitCountTotal     *prometheus.Desc
	waitDuration       *prometheus.Desc
	maxIdleClosedTotal *prometheus.Desc
	maxIdleTimeClosed  *prometheus.Desc
	maxLifeTimeClosed  *prometheus.Desc
}

func newDBStatsCollector(sqlDB *sql.DB) *dbStatsCollector {
	return &dbStatsCollector{
		sqlDB:              sqlDB,
		maxOpenConnections: prometheus.NewDesc("im_db_max_open_connections", "Configured maximum number of open DB connections.", nil, nil),
		openConnections:    prometheus.NewDesc("im_db_open_connections", "Current number of open DB connections.", nil, nil),
		inUseConnections:   prometheus.NewDesc("im_db_in_use_connections", "Current number of DB connections in use.", nil, nil),
		idleConnections:    prometheus.NewDesc("im_db_idle_connections", "Current number of idle DB connections.", nil, nil),
		waitCountTotal:     prometheus.NewDesc("im_db_wait_count_total", "Total number of waits for a DB connection.", nil, nil),
		waitDuration:       prometheus.NewDesc("im_db_wait_duration_seconds_total", "Total time blocked waiting for a DB connection.", nil, nil),
		maxIdleClosedTotal: prometheus.NewDesc("im_db_max_idle_closed_total", "Total connections closed because of SetMaxIdleConns.", nil, nil),
		maxIdleTimeClosed:  prometheus.NewDesc("im_db_max_idle_time_closed_total", "Total connections closed because of SetConnMaxIdleTime.", nil, nil),
		maxLifeTimeClosed:  prometheus.NewDesc("im_db_max_lifetime_closed_total", "Total connections closed because of SetConnMaxLifetime.", nil, nil),
	}
}

func (c *dbStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	descs := []*prometheus.Desc{
		c.maxOpenConnections,
		c.openConnections,
		c.inUseConnections,
		c.idleConnections,
		c.waitCountTotal,
		c.waitDuration,
		c.maxIdleClosedTotal,
		c.maxIdleTimeClosed,
		c.maxLifeTimeClosed,
	}
	for _, desc := range descs {
		ch <- desc
	}
}

func (c *dbStatsCollector) Collect(ch chan<- prometheus.Metric) {
	if c == nil || c.sqlDB == nil {
		return
	}

	stats := c.sqlDB.Stats()
	ch <- prometheus.MustNewConstMetric(c.maxOpenConnections, prometheus.GaugeValue, float64(stats.MaxOpenConnections))
	ch <- prometheus.MustNewConstMetric(c.openConnections, prometheus.GaugeValue, float64(stats.OpenConnections))
	ch <- prometheus.MustNewConstMetric(c.inUseConnections, prometheus.GaugeValue, float64(stats.InUse))
	ch <- prometheus.MustNewConstMetric(c.idleConnections, prometheus.GaugeValue, float64(stats.Idle))
	ch <- prometheus.MustNewConstMetric(c.waitCountTotal, prometheus.CounterValue, float64(stats.WaitCount))
	ch <- prometheus.MustNewConstMetric(c.waitDuration, prometheus.CounterValue, stats.WaitDuration.Seconds())
	ch <- prometheus.MustNewConstMetric(c.maxIdleClosedTotal, prometheus.CounterValue, float64(stats.MaxIdleClosed))
	ch <- prometheus.MustNewConstMetric(c.maxIdleTimeClosed, prometheus.CounterValue, float64(stats.MaxIdleTimeClosed))
	ch <- prometheus.MustNewConstMetric(c.maxLifeTimeClosed, prometheus.CounterValue, float64(stats.MaxLifetimeClosed))
}

type redisMetricsRecorder struct {
	operations *prometheus.CounterVec
	duration   *prometheus.HistogramVec
}

func (r *redisMetricsRecorder) ObserveOperation(module, op, result string, duration time.Duration) {
	if r == nil || r.operations == nil || r.duration == nil {
		return
	}
	r.operations.WithLabelValues(module, op, result).Inc()
	r.duration.WithLabelValues(module, op).Observe(duration.Seconds())
}

type noopRedisMetricsRecorder struct{}

func (noopRedisMetricsRecorder) ObserveOperation(string, string, string, time.Duration) {}
