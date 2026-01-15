// Package observability provides metrics collection and tracing for the API proxy.
// This file provides an official Prometheus client implementation that coexists
// with the existing custom MetricsCollector for gradual migration.
package observability

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMetrics provides an official Prometheus client implementation.
// This coexists with the existing MetricsCollector for gradual migration.
type PrometheusMetrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	tokensTotal     *prometheus.CounterVec
	activeRequests  prometheus.Gauge
	providerHealth  *prometheus.GaugeVec
	providerErrors  *prometheus.CounterVec
	cacheHits       prometheus.Counter
	cacheMisses     prometheus.Counter

	// Agentic metrics
	agentIterations    *prometheus.CounterVec
	agentToolCalls     *prometheus.CounterVec
	agentToolDuration  *prometheus.HistogramVec
	agentThinkingTokens *prometheus.CounterVec
	agentLoopDuration  *prometheus.HistogramVec
	agentLoopState     *prometheus.GaugeVec
}

// PrometheusConfig configures the official Prometheus metrics collector.
type PrometheusConfig struct {
	Namespace        string
	Subsystem        string
	HistogramBuckets []float64
}

// DefaultPrometheusConfig returns sensible defaults for Prometheus metrics.
func DefaultPrometheusConfig() PrometheusConfig {
	return PrometheusConfig{
		Namespace:        "shinapi",
		Subsystem:        "proxy",
		HistogramBuckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}
}

// NewPrometheusMetrics creates a new official Prometheus metrics collector.
func NewPrometheusMetrics(cfg PrometheusConfig) *PrometheusMetrics {
	if cfg.Namespace == "" {
		cfg.Namespace = "shinapi"
	}
	if cfg.Subsystem == "" {
		cfg.Subsystem = "proxy"
	}
	if len(cfg.HistogramBuckets) == 0 {
		cfg.HistogramBuckets = DefaultPrometheusConfig().HistogramBuckets
	}

	return &PrometheusMetrics{
		requestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "requests_total",
			Help:      "Total number of requests by model, provider, and status",
		}, []string{"model", "provider", "status"}),

		requestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "request_duration_seconds",
			Help:      "Request duration in seconds",
			Buckets:   cfg.HistogramBuckets,
		}, []string{"model", "provider"}),

		tokensTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "tokens_total",
			Help:      "Total tokens processed by model and type (prompt/completion/reasoning)",
		}, []string{"model", "type"}),

		activeRequests: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "active_requests",
			Help:      "Current number of active requests",
		}),

		providerHealth: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "provider_healthy",
			Help:      "Provider health status (1=healthy, 0=unhealthy)",
		}, []string{"provider"}),

		providerErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "provider_errors_total",
			Help:      "Total provider errors",
		}, []string{"provider"}),

		cacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "cache_hits_total",
			Help:      "Total cache hits",
		}),

		cacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "cache_misses_total",
			Help:      "Total cache misses",
		}),

		// Agentic metrics
		agentIterations: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: "agent",
			Name:      "iterations_total",
			Help:      "Total agent loop iterations by model and outcome",
		}, []string{"model", "outcome"}),

		agentToolCalls: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: "agent",
			Name:      "tool_calls_total",
			Help:      "Total tool calls by tool name and status",
		}, []string{"tool", "status"}),

		agentToolDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: "agent",
			Name:      "tool_duration_seconds",
			Help:      "Tool execution duration in seconds",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30},
		}, []string{"tool"}),

		agentThinkingTokens: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: "agent",
			Name:      "thinking_tokens_total",
			Help:      "Total thinking/reasoning tokens by model",
		}, []string{"model"}),

		agentLoopDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: "agent",
			Name:      "loop_duration_seconds",
			Help:      "Total agent loop duration in seconds",
			Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600},
		}, []string{"model"}),

		agentLoopState: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: "agent",
			Name:      "loop_state",
			Help:      "Current state of agent loops (1=active in that state)",
		}, []string{"state"}),
	}
}

// RecordRequest records a completed request with the given parameters.
func (p *PrometheusMetrics) RecordRequest(model, provider, status string, durationSeconds float64, tokens int64) {
	p.requestsTotal.WithLabelValues(model, provider, status).Inc()
	p.requestDuration.WithLabelValues(model, provider).Observe(durationSeconds)
	if tokens > 0 {
		p.tokensTotal.WithLabelValues(model, "total").Add(float64(tokens))
	}
}

// RecordTokens records token usage by type.
func (p *PrometheusMetrics) RecordTokens(model, tokenType string, count int64) {
	if count > 0 {
		p.tokensTotal.WithLabelValues(model, tokenType).Add(float64(count))
	}
}

// IncrementActiveRequests increments the active requests gauge.
func (p *PrometheusMetrics) IncrementActiveRequests() {
	p.activeRequests.Inc()
}

// DecrementActiveRequests decrements the active requests gauge.
func (p *PrometheusMetrics) DecrementActiveRequests() {
	p.activeRequests.Dec()
}

// SetProviderHealth sets the health status for a provider.
func (p *PrometheusMetrics) SetProviderHealth(provider string, healthy bool) {
	val := 0.0
	if healthy {
		val = 1.0
	}
	p.providerHealth.WithLabelValues(provider).Set(val)
}

// RecordProviderError records a provider error.
func (p *PrometheusMetrics) RecordProviderError(provider string) {
	p.providerErrors.WithLabelValues(provider).Inc()
}

// RecordCacheHit records a cache hit.
func (p *PrometheusMetrics) RecordCacheHit() {
	p.cacheHits.Inc()
}

// RecordCacheMiss records a cache miss.
func (p *PrometheusMetrics) RecordCacheMiss() {
	p.cacheMisses.Inc()
}

// RecordAgentIteration records an agent loop iteration.
func (p *PrometheusMetrics) RecordAgentIteration(model, outcome string) {
	p.agentIterations.WithLabelValues(model, outcome).Inc()
}

// RecordAgentToolCall records a tool call execution.
func (p *PrometheusMetrics) RecordAgentToolCall(tool, status string, durationSeconds float64) {
	p.agentToolCalls.WithLabelValues(tool, status).Inc()
	p.agentToolDuration.WithLabelValues(tool).Observe(durationSeconds)
}

// RecordAgentThinkingTokens records thinking/reasoning tokens.
func (p *PrometheusMetrics) RecordAgentThinkingTokens(model string, tokens int64) {
	if tokens > 0 {
		p.agentThinkingTokens.WithLabelValues(model).Add(float64(tokens))
	}
}

// RecordAgentLoopDuration records the total duration of an agent loop.
func (p *PrometheusMetrics) RecordAgentLoopDuration(model string, durationSeconds float64) {
	p.agentLoopDuration.WithLabelValues(model).Observe(durationSeconds)
}

// SetAgentLoopState sets the current state of agent loops.
func (p *PrometheusMetrics) SetAgentLoopState(state string, active bool) {
	val := 0.0
	if active {
		val = 1.0
	}
	p.agentLoopState.WithLabelValues(state).Set(val)
}

// Handler returns an HTTP handler for the official Prometheus metrics endpoint.
func (p *PrometheusMetrics) Handler() http.Handler {
	return promhttp.Handler()
}

// Global official Prometheus metrics instance.
var (
	globalPrometheusMetrics     *PrometheusMetrics
	globalPrometheusMetricsOnce sync.Once
)

// GetPrometheusMetrics returns the global official Prometheus metrics collector.
func GetPrometheusMetrics() *PrometheusMetrics {
	globalPrometheusMetricsOnce.Do(func() {
		globalPrometheusMetrics = NewPrometheusMetrics(DefaultPrometheusConfig())
	})
	return globalPrometheusMetrics
}

// InitPrometheusMetrics initializes the global official Prometheus metrics collector with custom config.
func InitPrometheusMetrics(cfg PrometheusConfig) *PrometheusMetrics {
	globalPrometheusMetricsOnce.Do(func() {
		globalPrometheusMetrics = NewPrometheusMetrics(cfg)
	})
	return globalPrometheusMetrics
}
