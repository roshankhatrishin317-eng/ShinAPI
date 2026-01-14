package management

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ProviderStatus represents the health status of a provider.
type ProviderStatus struct {
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Healthy      bool      `json:"healthy"`
	Requests     int64     `json:"requests"`
	Errors       int64     `json:"errors"`
	ErrorRate    float64   `json:"error_rate_percent"`
	AvgLatencyMs float64   `json:"avg_latency_ms"`
	P95LatencyMs float64   `json:"p95_latency_ms"`
	LastError    time.Time `json:"last_error_time,omitempty"`
	LastSuccess  time.Time `json:"last_success_time,omitempty"`
	Credentials  int       `json:"credentials"`
	RateLimited  bool      `json:"rate_limited"`
}

// ProviderHealthResponse is the response for the provider health endpoint.
type ProviderHealthResponse struct {
	Providers []ProviderStatus `json:"providers"`
	Overall   struct {
		Healthy        bool    `json:"healthy"`
		TotalProviders int     `json:"total_providers"`
		HealthyCount   int     `json:"healthy_count"`
		UnhealthyCount int     `json:"unhealthy_count"`
		AvgErrorRate   float64 `json:"avg_error_rate_percent"`
	} `json:"overall"`
	Timestamp int64 `json:"timestamp"`
}

// ProviderHealthTracker tracks provider health metrics.
type ProviderHealthTracker struct {
	mu        sync.RWMutex
	providers map[string]*providerStats
}

type providerStats struct {
	name           string
	providerType   string
	requests       int64
	errors         int64
	latencies      []float64
	latencyIndex   int
	lastError      time.Time
	lastSuccess    time.Time
	credentials    int
	rateLimited    bool
	rateLimitUntil time.Time
}

var (
	globalProviderHealth     *ProviderHealthTracker
	globalProviderHealthOnce sync.Once
)

// GetProviderHealthTracker returns the global provider health tracker.
func GetProviderHealthTracker() *ProviderHealthTracker {
	globalProviderHealthOnce.Do(func() {
		globalProviderHealth = &ProviderHealthTracker{
			providers: make(map[string]*providerStats),
		}
	})
	return globalProviderHealth
}

// RecordRequest records a provider request.
func (t *ProviderHealthTracker) RecordRequest(provider, providerType string, latencyMs float64, success bool, credentials int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.providers[provider] == nil {
		t.providers[provider] = &providerStats{
			name:         provider,
			providerType: providerType,
			latencies:    make([]float64, 100),
		}
	}

	ps := t.providers[provider]
	ps.requests++
	ps.credentials = credentials

	// Record latency
	ps.latencies[ps.latencyIndex%100] = latencyMs
	ps.latencyIndex++

	if success {
		ps.lastSuccess = time.Now()
		// Clear rate limit if successful
		if time.Now().After(ps.rateLimitUntil) {
			ps.rateLimited = false
		}
	} else {
		ps.errors++
		ps.lastError = time.Now()
	}
}

// RecordRateLimit records that a provider is rate limited.
func (t *ProviderHealthTracker) RecordRateLimit(provider string, duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.providers[provider] != nil {
		t.providers[provider].rateLimited = true
		t.providers[provider].rateLimitUntil = time.Now().Add(duration)
	}
}

// GetStatus returns the health status of all providers.
func (t *ProviderHealthTracker) GetStatus() ProviderHealthResponse {
	t.mu.RLock()
	defer t.mu.RUnlock()

	resp := ProviderHealthResponse{
		Providers: make([]ProviderStatus, 0, len(t.providers)),
		Timestamp: time.Now().Unix(),
	}

	var totalErrorRate float64
	for _, ps := range t.providers {
		status := ProviderStatus{
			Name:        ps.name,
			Type:        ps.providerType,
			Requests:    ps.requests,
			Errors:      ps.errors,
			LastError:   ps.lastError,
			LastSuccess: ps.lastSuccess,
			Credentials: ps.credentials,
			RateLimited: ps.rateLimited && time.Now().Before(ps.rateLimitUntil),
		}

		// Calculate error rate
		if ps.requests > 0 {
			status.ErrorRate = float64(ps.errors) / float64(ps.requests) * 100
		}

		// Calculate latencies
		count := ps.latencyIndex
		if count > 100 {
			count = 100
		}
		if count > 0 {
			var sum float64
			latencies := make([]float64, count)
			copy(latencies, ps.latencies[:count])
			for _, l := range latencies {
				sum += l
			}
			status.AvgLatencyMs = sum / float64(count)

			// P95 (simple sort)
			for i := 0; i < len(latencies)-1; i++ {
				for j := i + 1; j < len(latencies); j++ {
					if latencies[i] > latencies[j] {
						latencies[i], latencies[j] = latencies[j], latencies[i]
					}
				}
			}
			p95Index := int(float64(count) * 0.95)
			if p95Index >= count {
				p95Index = count - 1
			}
			status.P95LatencyMs = latencies[p95Index]
		}

		// Determine health
		status.Healthy = status.ErrorRate < 50 && !status.RateLimited
		if status.Healthy {
			resp.Overall.HealthyCount++
		} else {
			resp.Overall.UnhealthyCount++
		}

		totalErrorRate += status.ErrorRate
		resp.Providers = append(resp.Providers, status)
	}

	resp.Overall.TotalProviders = len(t.providers)
	resp.Overall.Healthy = resp.Overall.UnhealthyCount == 0
	if resp.Overall.TotalProviders > 0 {
		resp.Overall.AvgErrorRate = totalErrorRate / float64(resp.Overall.TotalProviders)
	}

	return resp
}

// GetProviderHealth handles the GET /v0/management/providers/health endpoint.
func (h *Handler) GetProviderHealth(c *gin.Context) {
	tracker := GetProviderHealthTracker()
	resp := tracker.GetStatus()
	c.JSON(http.StatusOK, resp)
}

// CacheStatsResponse is the response for cache statistics.
type CacheStatsResponse struct {
	LRU struct {
		Hits     uint64  `json:"hits"`
		Misses   uint64  `json:"misses"`
		Size     int     `json:"size"`
		Capacity int     `json:"capacity"`
		HitRate  float64 `json:"hit_rate_percent"`
	} `json:"lru"`
	Semantic *struct {
		Enabled    bool    `json:"enabled"`
		Hits       uint64  `json:"hits"`
		Misses     uint64  `json:"misses"`
		IndexSize  int     `json:"index_size"`
		HitRate    float64 `json:"hit_rate_percent"`
	} `json:"semantic,omitempty"`
	Streaming *struct {
		Enabled     bool    `json:"enabled"`
		Entries     int     `json:"entries"`
		TotalEvents int     `json:"total_events"`
		TotalSize   int64   `json:"total_size_bytes"`
		HitRate     float64 `json:"hit_rate_percent"`
	} `json:"streaming,omitempty"`
	Redis *struct {
		Enabled       bool    `json:"enabled"`
		Connected     bool    `json:"connected"`
		Hits          uint64  `json:"hits"`
		Misses        uint64  `json:"misses"`
		Errors        uint64  `json:"errors"`
		HitRate       float64 `json:"hit_rate_percent"`
		LastLatencyMs float64 `json:"last_latency_ms"`
	} `json:"redis,omitempty"`
	Timestamp int64 `json:"timestamp"`
}

// GetCacheStats handles the GET /v0/management/cache/stats endpoint.
func (h *Handler) GetCacheStats(c *gin.Context) {
	// This would integrate with the actual cache instances
	// For now, return a placeholder response
	resp := CacheStatsResponse{
		Timestamp: time.Now().Unix(),
	}
	resp.LRU.Hits = 0
	resp.LRU.Misses = 0
	resp.LRU.Size = 0
	resp.LRU.Capacity = 1000
	resp.LRU.HitRate = 0

	c.JSON(http.StatusOK, resp)
}

// SchedulerStatsResponse is the response for scheduler statistics.
type SchedulerStatsResponse struct {
	Enabled      bool `json:"enabled"`
	TotalPending int  `json:"total_pending"`
	Queues       []struct {
		APIKey          string  `json:"api_key"`
		PendingRequests int     `json:"pending_requests"`
		Weight          int     `json:"weight"`
		TotalTokens     int64   `json:"total_tokens"`
	} `json:"queues"`
	Metrics struct {
		TotalEnqueued   int64 `json:"total_enqueued"`
		TotalDequeued   int64 `json:"total_dequeued"`
		TotalExecuted   int64 `json:"total_executed"`
		TotalRejected   int64 `json:"total_rejected"`
		TotalCancelled  int64 `json:"total_cancelled"`
		TotalSuccessful int64 `json:"total_successful"`
		TotalFailed     int64 `json:"total_failed"`
	} `json:"metrics"`
	Timestamp int64 `json:"timestamp"`
}

// GetSchedulerStats handles the GET /v0/management/scheduler/stats endpoint.
func (h *Handler) GetSchedulerStats(c *gin.Context) {
	resp := SchedulerStatsResponse{
		Enabled:   false,
		Timestamp: time.Now().Unix(),
	}
	c.JSON(http.StatusOK, resp)
}
