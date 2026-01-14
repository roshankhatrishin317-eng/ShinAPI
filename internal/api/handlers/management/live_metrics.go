package management

import (
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// LiveMetricsSnapshot represents current live metrics
type LiveMetricsSnapshot struct {
	// Rates
	RPM           int64   `json:"rpm"`            // Requests per minute
	TPM           int64   `json:"tpm"`            // Tokens per minute
	TPS           float64 `json:"tps"`            // Transactions per second
	
	// Totals
	TotalRequests int64   `json:"total_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalSuccess  int64   `json:"total_success"`
	TotalFailed   int64   `json:"total_failed"`
	SuccessRate   float64 `json:"success_rate"`
	
	// Latency stats (ms)
	AvgLatency    float64 `json:"avg_latency_ms"`
	P50Latency    float64 `json:"p50_latency_ms"`
	P95Latency    float64 `json:"p95_latency_ms"`
	P99Latency    float64 `json:"p99_latency_ms"`
	
	// Uptime
	UptimeSeconds int64   `json:"uptime_seconds"`
	
	// Per-model breakdown
	ModelStats    map[string]ModelMetrics `json:"model_stats"`
	
	// Timestamp
	Timestamp     int64   `json:"timestamp"`
}

// ModelMetrics contains per-model statistics
type ModelMetrics struct {
	Requests int64 `json:"requests"`
	Tokens   int64 `json:"tokens"`
}

var serverStartTime = time.Now()

// GetLiveMetrics returns real-time metrics as JSON
func (h *Handler) GetLiveMetrics(c *gin.Context) {
	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)
	tenSecondsAgo := now.Add(-10 * time.Second)
	
	snapshot := LiveMetricsSnapshot{
		Timestamp:     now.Unix(),
		UptimeSeconds: int64(now.Sub(serverStartTime).Seconds()),
		ModelStats:    make(map[string]ModelMetrics),
	}
	
	if h == nil || h.usageStats == nil {
		c.JSON(http.StatusOK, snapshot)
		return
	}
	
	usageSnapshot := h.usageStats.Snapshot()
	
	// Set totals from usage stats
	snapshot.TotalRequests = usageSnapshot.TotalRequests
	snapshot.TotalTokens = usageSnapshot.TotalTokens
	snapshot.TotalSuccess = usageSnapshot.SuccessCount
	snapshot.TotalFailed = usageSnapshot.FailureCount
	
	if usageSnapshot.TotalRequests > 0 {
		snapshot.SuccessRate = float64(usageSnapshot.SuccessCount) / float64(usageSnapshot.TotalRequests) * 100
	}
	
	// Calculate RPM, TPM, TPS from request details
	var requestsLastMinute int64
	var tokensLastMinute int64
	var requestsLast10Seconds int64
	
	for _, apiSnapshot := range usageSnapshot.APIs {
		for modelName, modelSnapshot := range apiSnapshot.Models {
			// Update model stats
			if _, exists := snapshot.ModelStats[modelName]; !exists {
				snapshot.ModelStats[modelName] = ModelMetrics{
					Requests: modelSnapshot.TotalRequests,
					Tokens:   modelSnapshot.TotalTokens,
				}
			} else {
				existing := snapshot.ModelStats[modelName]
				existing.Requests += modelSnapshot.TotalRequests
				existing.Tokens += modelSnapshot.TotalTokens
				snapshot.ModelStats[modelName] = existing
			}
			
			// Count recent requests for RPM/TPS
			for _, detail := range modelSnapshot.Details {
				if detail.Timestamp.After(oneMinuteAgo) {
					requestsLastMinute++
					tokensLastMinute += detail.Tokens.TotalTokens
				}
				if detail.Timestamp.After(tenSecondsAgo) {
					requestsLast10Seconds++
				}
			}
		}
	}
	
	snapshot.RPM = requestsLastMinute
	snapshot.TPM = tokensLastMinute
	snapshot.TPS = float64(requestsLast10Seconds) / 10.0
	
	c.JSON(http.StatusOK, snapshot)
}

// RealTimeTracker provides real-time request tracking with sub-second granularity
type RealTimeTracker struct {
	mu sync.RWMutex
	
	// Circular buffer for last 60 seconds of request counts
	requestCounts [60]int64
	tokenCounts   [60]int64
	lastSecond    int64
	
	// Total counters
	totalRequests int64
	totalTokens   int64
	totalSuccess  int64
	totalFailed   int64
	
	// Per-model stats
	modelStats map[string]*modelTracker
	
	// Latency tracking (last 1000 requests)
	latencies    []int64 // in milliseconds
	latencyIndex int
	latencyMu    sync.Mutex
	
	startTime time.Time
}

type modelTracker struct {
	requests int64
	tokens   int64
}

var (
	globalTracker     *RealTimeTracker
	globalTrackerOnce sync.Once
)

// GetRealTimeTracker returns the global real-time tracker
func GetRealTimeTracker() *RealTimeTracker {
	globalTrackerOnce.Do(func() {
		globalTracker = &RealTimeTracker{
			modelStats: make(map[string]*modelTracker),
			latencies:  make([]int64, 1000),
			startTime:  time.Now(),
		}
	})
	return globalTracker
}

// Record records a request to the real-time tracker
func (t *RealTimeTracker) Record(model string, tokens int64, latencyMs int64, success bool) {
	if t == nil {
		return
	}
	
	now := time.Now()
	currentSecond := now.Unix()
	
	t.mu.Lock()
	
	// Roll over to new second if needed
	if currentSecond != t.lastSecond {
		// Clear seconds between lastSecond and currentSecond
		if t.lastSecond > 0 {
			diff := currentSecond - t.lastSecond
			if diff > 60 {
				diff = 60
			}
			for i := int64(1); i <= diff; i++ {
				idx := (t.lastSecond + i) % 60
				t.requestCounts[idx] = 0
				t.tokenCounts[idx] = 0
			}
		}
		t.lastSecond = currentSecond
	}
	
	// Update current second
	idx := currentSecond % 60
	t.requestCounts[idx]++
	t.tokenCounts[idx] += tokens
	
	// Update totals
	t.totalRequests++
	t.totalTokens += tokens
	if success {
		t.totalSuccess++
	} else {
		t.totalFailed++
	}
	
	// Update model stats
	if t.modelStats[model] == nil {
		t.modelStats[model] = &modelTracker{}
	}
	t.modelStats[model].requests++
	t.modelStats[model].tokens += tokens
	
	t.mu.Unlock()
	
	// Record latency (separate lock to avoid blocking)
	t.latencyMu.Lock()
	t.latencies[t.latencyIndex%1000] = latencyMs
	t.latencyIndex++
	t.latencyMu.Unlock()
}

// Snapshot returns current metrics
func (t *RealTimeTracker) Snapshot() LiveMetricsSnapshot {
	now := time.Now()
	currentSecond := now.Unix()
	
	snapshot := LiveMetricsSnapshot{
		Timestamp:     now.Unix(),
		UptimeSeconds: int64(now.Sub(t.startTime).Seconds()),
		ModelStats:    make(map[string]ModelMetrics),
	}
	
	if t == nil {
		return snapshot
	}
	
	t.mu.RLock()
	
	// Calculate RPM (sum of last 60 seconds)
	var rpm int64
	var tpm int64
	for i := 0; i < 60; i++ {
		rpm += t.requestCounts[i]
		tpm += t.tokenCounts[i]
	}
	snapshot.RPM = rpm
	snapshot.TPM = tpm
	
	// Calculate TPS (average of last 10 seconds)
	var tps int64
	for i := 0; i < 10; i++ {
		idx := (currentSecond - int64(i) + 60) % 60
		tps += t.requestCounts[idx]
	}
	snapshot.TPS = float64(tps) / 10.0
	
	// Totals
	snapshot.TotalRequests = t.totalRequests
	snapshot.TotalTokens = t.totalTokens
	snapshot.TotalSuccess = t.totalSuccess
	snapshot.TotalFailed = t.totalFailed
	
	if t.totalRequests > 0 {
		snapshot.SuccessRate = float64(t.totalSuccess) / float64(t.totalRequests) * 100
	}
	
	// Model stats
	for model, stats := range t.modelStats {
		snapshot.ModelStats[model] = ModelMetrics{
			Requests: stats.requests,
			Tokens:   stats.tokens,
		}
	}
	
	t.mu.RUnlock()
	
	// Calculate latency percentiles
	t.latencyMu.Lock()
	count := t.latencyIndex
	if count > 1000 {
		count = 1000
	}
	if count > 0 {
		latencies := make([]int64, count)
		copy(latencies, t.latencies[:count])
		t.latencyMu.Unlock()
		
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})
		
		var sum int64
		for _, l := range latencies {
			sum += l
		}
		snapshot.AvgLatency = float64(sum) / float64(len(latencies))
		snapshot.P50Latency = float64(latencies[len(latencies)*50/100])
		snapshot.P95Latency = float64(latencies[len(latencies)*95/100])
		p99Idx := len(latencies) * 99 / 100
		if p99Idx >= len(latencies) {
			p99Idx = len(latencies) - 1
		}
		snapshot.P99Latency = float64(latencies[p99Idx])
	} else {
		t.latencyMu.Unlock()
	}
	
	return snapshot
}
