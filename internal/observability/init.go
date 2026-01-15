// Package observability provides metrics collection and tracing for the API proxy.
package observability

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ObservabilityConfig holds all observability configuration.
type ObservabilityConfig struct {
	// Metrics configures Prometheus metrics.
	Metrics MetricsConfig `yaml:"metrics" json:"metrics"`
	// Tracing configures OpenTelemetry tracing.
	Tracing TracerConfig `yaml:"tracing" json:"tracing"`
}

// DefaultObservabilityConfig returns sensible defaults.
func DefaultObservabilityConfig() ObservabilityConfig {
	return ObservabilityConfig{
		Metrics: DefaultMetricsConfig(),
		Tracing: DefaultTracerConfig(),
	}
}

// RegisterGinRoutes registers observability routes on a Gin engine.
func RegisterGinRoutes(r *gin.Engine, cfg ObservabilityConfig) {
	RegisterGinRoutesWithOptions(r, cfg, false)
}

// RegisterGinRoutesWithOptions registers observability routes with optional official Prometheus client.
// When useOfficialClient is true, uses promhttp.Handler() from prometheus/client_golang.
func RegisterGinRoutesWithOptions(r *gin.Engine, cfg ObservabilityConfig, useOfficialClient bool) {
	if cfg.Metrics.Enabled {
		path := cfg.Metrics.Path
		if path == "" {
			path = "/metrics"
		}

		if useOfficialClient {
			// Use official prometheus/client_golang handler
			r.GET(path, gin.WrapH(promhttp.Handler()))
		} else {
			// Use custom MetricsCollector handler
			metrics := GetMetrics()
			r.GET(path, gin.WrapH(metrics.Handler()))
		}
	}

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		health := GetHealthStatus()
		status := http.StatusOK
		if !health.Healthy {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, health)
	})

	// Provider health endpoint
	r.GET("/health/providers", func(c *gin.Context) {
		providers := GetMetrics().GetProviderHealth()
		c.JSON(http.StatusOK, providers)
	})
}

// HealthStatus represents the overall health of the service.
type HealthStatus struct {
	Healthy   bool                          `json:"healthy"`
	Uptime    string                        `json:"uptime"`
	Providers map[string]ProviderHealthStatus `json:"providers,omitempty"`
	Cache     *CacheHealthStatus            `json:"cache,omitempty"`
	Scheduler *SchedulerHealthStatus        `json:"scheduler,omitempty"`
}

// CacheHealthStatus represents cache health.
type CacheHealthStatus struct {
	Healthy  bool    `json:"healthy"`
	HitRate  float64 `json:"hit_rate_percent"`
	Size     int     `json:"size"`
	Capacity int     `json:"capacity"`
}

// SchedulerHealthStatus represents scheduler health.
type SchedulerHealthStatus struct {
	Healthy      bool  `json:"healthy"`
	QueuedJobs   int64 `json:"queued_jobs"`
	ActiveJobs   int64 `json:"active_jobs"`
	RejectedJobs int64 `json:"rejected_jobs"`
}

// GetHealthStatus returns the current health status.
func GetHealthStatus() HealthStatus {
	metrics := GetMetrics()
	providers := metrics.GetProviderHealth()

	healthy := true
	for _, p := range providers {
		if !p.Healthy {
			healthy = false
			break
		}
	}

	return HealthStatus{
		Healthy:   healthy,
		Uptime:    formatUptime(metrics.startTime),
		Providers: providers,
	}
}

func formatUptime(start interface{}) string {
	// This is a placeholder - would format actual uptime
	return "running"
}
