// Package observability provides metrics collection and tracing for the API proxy.
package observability

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects and exposes Prometheus-compatible metrics.
type MetricsCollector struct {
	mu sync.RWMutex

	// Request metrics
	requestsTotal     map[string]*uint64 // model:status -> count
	requestDurations  map[string]*histogram // model -> latency histogram
	tokensTotal       map[string]*uint64 // model:type -> count
	activeRequests    int64

	// Provider metrics
	providerHealth    map[string]*providerMetrics
	
	// Cache metrics
	cacheHits         uint64
	cacheMisses       uint64
	cacheLatencySum   uint64
	cacheLatencyCount uint64

	// Scheduler metrics
	schedulerQueueSize    map[string]*int64
	schedulerWaitTimeSum  uint64
	schedulerWaitTimeCount uint64

	// System metrics
	startTime time.Time

	config MetricsConfig
}

type providerMetrics struct {
	requests     uint64
	errors       uint64
	latencySum   uint64
	latencyCount uint64
	lastError    time.Time
	healthy      bool
}

type histogram struct {
	buckets []uint64 // count per bucket
	sum     uint64
	count   uint64
}

// MetricsConfig configures the metrics collector.
type MetricsConfig struct {
	// Enabled controls whether metrics collection is active.
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Path is the HTTP path for the metrics endpoint.
	Path string `yaml:"path" json:"path"`
	// Namespace is the Prometheus namespace prefix.
	Namespace string `yaml:"namespace" json:"namespace"`
	// Subsystem is the Prometheus subsystem prefix.
	Subsystem string `yaml:"subsystem" json:"subsystem"`
	// HistogramBuckets defines latency histogram buckets in milliseconds.
	HistogramBuckets []float64 `yaml:"histogram-buckets" json:"histogram_buckets"`
}

// DefaultMetricsConfig returns sensible defaults.
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Enabled:   true,
		Path:      "/metrics",
		Namespace: "shinapi",
		Subsystem: "proxy",
		HistogramBuckets: []float64{
			5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000,
		},
	}
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector(cfg MetricsConfig) *MetricsCollector {
	if cfg.Namespace == "" {
		cfg.Namespace = "shinapi"
	}
	if cfg.Subsystem == "" {
		cfg.Subsystem = "proxy"
	}
	if len(cfg.HistogramBuckets) == 0 {
		cfg.HistogramBuckets = DefaultMetricsConfig().HistogramBuckets
	}

	return &MetricsCollector{
		requestsTotal:      make(map[string]*uint64),
		requestDurations:   make(map[string]*histogram),
		tokensTotal:        make(map[string]*uint64),
		providerHealth:     make(map[string]*providerMetrics),
		schedulerQueueSize: make(map[string]*int64),
		startTime:          time.Now(),
		config:             cfg,
	}
}

// RecordRequest records a completed request.
func (m *MetricsCollector) RecordRequest(model, status string, durationMs float64, tokens int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Increment request counter
	key := model + ":" + status
	if m.requestsTotal[key] == nil {
		var v uint64
		m.requestsTotal[key] = &v
	}
	atomic.AddUint64(m.requestsTotal[key], 1)

	// Record duration histogram
	if m.requestDurations[model] == nil {
		m.requestDurations[model] = &histogram{
			buckets: make([]uint64, len(m.config.HistogramBuckets)+1),
		}
	}
	h := m.requestDurations[model]
	h.sum += uint64(durationMs)
	h.count++

	// Find bucket
	for i, bound := range m.config.HistogramBuckets {
		if durationMs <= bound {
			h.buckets[i]++
			break
		}
		if i == len(m.config.HistogramBuckets)-1 {
			h.buckets[i+1]++
		}
	}

	// Record tokens
	if tokens > 0 {
		tokenKey := model + ":total"
		if m.tokensTotal[tokenKey] == nil {
			var v uint64
			m.tokensTotal[tokenKey] = &v
		}
		atomic.AddUint64(m.tokensTotal[tokenKey], uint64(tokens))
	}
}

// RecordProviderRequest records a provider request.
func (m *MetricsCollector) RecordProviderRequest(provider string, durationMs float64, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.providerHealth[provider] == nil {
		m.providerHealth[provider] = &providerMetrics{healthy: true}
	}
	pm := m.providerHealth[provider]
	pm.requests++
	pm.latencySum += uint64(durationMs)
	pm.latencyCount++

	if !success {
		pm.errors++
		pm.lastError = time.Now()
		// Mark unhealthy if error rate > 50% in last 10 requests
		if pm.requests >= 10 {
			errorRate := float64(pm.errors) / float64(pm.requests)
			pm.healthy = errorRate < 0.5
		}
	} else {
		pm.healthy = true
	}
}

// RecordCacheAccess records a cache access.
func (m *MetricsCollector) RecordCacheAccess(hit bool, latencyMs float64) {
	if hit {
		atomic.AddUint64(&m.cacheHits, 1)
	} else {
		atomic.AddUint64(&m.cacheMisses, 1)
	}
	atomic.AddUint64(&m.cacheLatencySum, uint64(latencyMs*1000))
	atomic.AddUint64(&m.cacheLatencyCount, 1)
}

// RecordSchedulerQueue records scheduler queue size.
func (m *MetricsCollector) RecordSchedulerQueue(apiKey string, size int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.schedulerQueueSize[apiKey] == nil {
		var v int64
		m.schedulerQueueSize[apiKey] = &v
	}
	atomic.StoreInt64(m.schedulerQueueSize[apiKey], size)
}

// RecordSchedulerWait records scheduler wait time.
func (m *MetricsCollector) RecordSchedulerWait(durationMs float64) {
	atomic.AddUint64(&m.schedulerWaitTimeSum, uint64(durationMs*1000))
	atomic.AddUint64(&m.schedulerWaitTimeCount, 1)
}

// IncrementActiveRequests increments active request count.
func (m *MetricsCollector) IncrementActiveRequests() {
	atomic.AddInt64(&m.activeRequests, 1)
}

// DecrementActiveRequests decrements active request count.
func (m *MetricsCollector) DecrementActiveRequests() {
	atomic.AddInt64(&m.activeRequests, -1)
}

// GetProviderHealth returns health status for all providers.
func (m *MetricsCollector) GetProviderHealth() map[string]ProviderHealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]ProviderHealthStatus)
	for provider, pm := range m.providerHealth {
		var avgLatency float64
		if pm.latencyCount > 0 {
			avgLatency = float64(pm.latencySum) / float64(pm.latencyCount)
		}
		var errorRate float64
		if pm.requests > 0 {
			errorRate = float64(pm.errors) / float64(pm.requests) * 100
		}

		result[provider] = ProviderHealthStatus{
			Healthy:       pm.healthy,
			Requests:      pm.requests,
			Errors:        pm.errors,
			ErrorRate:     errorRate,
			AvgLatencyMs:  avgLatency,
			LastErrorTime: pm.lastError,
		}
	}

	return result
}

// ProviderHealthStatus represents provider health.
type ProviderHealthStatus struct {
	Healthy       bool      `json:"healthy"`
	Requests      uint64    `json:"requests"`
	Errors        uint64    `json:"errors"`
	ErrorRate     float64   `json:"error_rate_percent"`
	AvgLatencyMs  float64   `json:"avg_latency_ms"`
	LastErrorTime time.Time `json:"last_error_time,omitempty"`
}

// Handler returns an HTTP handler for the metrics endpoint.
func (m *MetricsCollector) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Write([]byte(m.Export()))
	})
}

// Export exports metrics in Prometheus format.
func (m *MetricsCollector) Export() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sb strings.Builder
	ns := m.config.Namespace
	ss := m.config.Subsystem
	prefix := ns + "_" + ss

	// Request counters
	sb.WriteString(fmt.Sprintf("# HELP %s_requests_total Total number of requests\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_requests_total counter\n", prefix))
	for key, count := range m.requestsTotal {
		parts := strings.SplitN(key, ":", 2)
		model, status := parts[0], "success"
		if len(parts) > 1 {
			status = parts[1]
		}
		sb.WriteString(fmt.Sprintf("%s_requests_total{model=\"%s\",status=\"%s\"} %d\n",
			prefix, model, status, atomic.LoadUint64(count)))
	}

	// Request duration histograms
	sb.WriteString(fmt.Sprintf("# HELP %s_request_duration_milliseconds Request duration histogram\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_request_duration_milliseconds histogram\n", prefix))
	for model, h := range m.requestDurations {
		var cumulative uint64
		for i, bucket := range m.config.HistogramBuckets {
			cumulative += h.buckets[i]
			sb.WriteString(fmt.Sprintf("%s_request_duration_milliseconds_bucket{model=\"%s\",le=\"%.0f\"} %d\n",
				prefix, model, bucket, cumulative))
		}
		cumulative += h.buckets[len(m.config.HistogramBuckets)]
		sb.WriteString(fmt.Sprintf("%s_request_duration_milliseconds_bucket{model=\"%s\",le=\"+Inf\"} %d\n",
			prefix, model, cumulative))
		sb.WriteString(fmt.Sprintf("%s_request_duration_milliseconds_sum{model=\"%s\"} %d\n",
			prefix, model, h.sum))
		sb.WriteString(fmt.Sprintf("%s_request_duration_milliseconds_count{model=\"%s\"} %d\n",
			prefix, model, h.count))
	}

	// Token counters
	sb.WriteString(fmt.Sprintf("# HELP %s_tokens_total Total tokens processed\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_tokens_total counter\n", prefix))
	for key, count := range m.tokensTotal {
		parts := strings.SplitN(key, ":", 2)
		model := parts[0]
		sb.WriteString(fmt.Sprintf("%s_tokens_total{model=\"%s\"} %d\n",
			prefix, model, atomic.LoadUint64(count)))
	}

	// Active requests gauge
	sb.WriteString(fmt.Sprintf("# HELP %s_active_requests Current number of active requests\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_active_requests gauge\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_active_requests %d\n", prefix, atomic.LoadInt64(&m.activeRequests)))

	// Provider health
	sb.WriteString(fmt.Sprintf("# HELP %s_provider_healthy Provider health status\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_provider_healthy gauge\n", prefix))
	sb.WriteString(fmt.Sprintf("# HELP %s_provider_requests_total Provider request count\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_provider_requests_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("# HELP %s_provider_errors_total Provider error count\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_provider_errors_total counter\n", prefix))
	
	providers := make([]string, 0, len(m.providerHealth))
	for p := range m.providerHealth {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	
	for _, provider := range providers {
		pm := m.providerHealth[provider]
		healthy := 0
		if pm.healthy {
			healthy = 1
		}
		sb.WriteString(fmt.Sprintf("%s_provider_healthy{provider=\"%s\"} %d\n", prefix, provider, healthy))
		sb.WriteString(fmt.Sprintf("%s_provider_requests_total{provider=\"%s\"} %d\n", prefix, provider, pm.requests))
		sb.WriteString(fmt.Sprintf("%s_provider_errors_total{provider=\"%s\"} %d\n", prefix, provider, pm.errors))
	}

	// Cache metrics
	sb.WriteString(fmt.Sprintf("# HELP %s_cache_hits_total Cache hits\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_cache_hits_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_cache_hits_total %d\n", prefix, atomic.LoadUint64(&m.cacheHits)))
	sb.WriteString(fmt.Sprintf("# HELP %s_cache_misses_total Cache misses\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_cache_misses_total counter\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_cache_misses_total %d\n", prefix, atomic.LoadUint64(&m.cacheMisses)))

	// Scheduler metrics
	sb.WriteString(fmt.Sprintf("# HELP %s_scheduler_queue_size Scheduler queue size per API key\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_scheduler_queue_size gauge\n", prefix))
	for apiKey, size := range m.schedulerQueueSize {
		// Hash API key for privacy
		keyHash := apiKey
		if len(keyHash) > 8 {
			keyHash = keyHash[:8] + "..."
		}
		sb.WriteString(fmt.Sprintf("%s_scheduler_queue_size{api_key=\"%s\"} %d\n",
			prefix, keyHash, atomic.LoadInt64(size)))
	}

	// Uptime
	sb.WriteString(fmt.Sprintf("# HELP %s_uptime_seconds Server uptime in seconds\n", prefix))
	sb.WriteString(fmt.Sprintf("# TYPE %s_uptime_seconds gauge\n", prefix))
	sb.WriteString(fmt.Sprintf("%s_uptime_seconds %.0f\n", prefix, time.Since(m.startTime).Seconds()))

	return sb.String()
}

// Global metrics collector
var (
	globalMetrics     *MetricsCollector
	globalMetricsOnce sync.Once
)

// GetMetrics returns the global metrics collector.
func GetMetrics() *MetricsCollector {
	globalMetricsOnce.Do(func() {
		globalMetrics = NewMetricsCollector(DefaultMetricsConfig())
	})
	return globalMetrics
}

// InitMetrics initializes the global metrics collector with custom config.
func InitMetrics(cfg MetricsConfig) *MetricsCollector {
	globalMetricsOnce.Do(func() {
		globalMetrics = NewMetricsCollector(cfg)
	})
	return globalMetrics
}
