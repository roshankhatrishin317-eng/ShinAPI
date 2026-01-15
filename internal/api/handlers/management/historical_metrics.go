package management

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
)

// HistoricalMetricsResponse is the API response for historical metrics.
type HistoricalMetricsResponse struct {
	Range   string                    `json:"range"`
	Data    usage.HistoricalSnapshot  `json:"data"`
	Summary HistoricalSummary         `json:"summary"`
}

// HistoricalSummary provides quick stats for the requested range.
type HistoricalSummary struct {
	TotalRequests int64   `json:"total_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	AvgTPS        float64 `json:"avg_tps"`
	AvgTPM        float64 `json:"avg_tpm"`
	PeakTPS       float64 `json:"peak_tps"`
	PeakTPM       int64   `json:"peak_tpm"`
	SuccessRate   float64 `json:"success_rate"`
	AvgLatency    float64 `json:"avg_latency_ms"`
}

// GetHistoricalMetrics returns historical metrics for a given time range.
// Query params:
//   - range: 1m, 1h, 24h, 7d, 30d (default: 1h)
func (h *Handler) GetHistoricalMetrics(c *gin.Context) {
	rangeParam := c.DefaultQuery("range", "1h")

	hm := usage.GetHistoricalMetrics()
	if hm == nil {
		c.JSON(http.StatusOK, HistoricalMetricsResponse{
			Range: rangeParam,
			Data:  usage.HistoricalSnapshot{},
		})
		return
	}

	var snapshot usage.HistoricalSnapshot
	var summary HistoricalSummary

	switch rangeParam {
	case "1m", "seconds":
		snapshot = hm.Snapshot(true, false, false, false)
		summary = calculateSummaryFromBuckets(snapshot.Seconds)
	case "1h", "minutes":
		snapshot = hm.Snapshot(false, true, false, false)
		summary = calculateSummaryFromBuckets(snapshot.Minutes)
	case "24h", "hours":
		snapshot = hm.Snapshot(false, false, true, false)
		summary = calculateSummaryFromBuckets(snapshot.Hours)
	case "7d":
		snapshot = hm.Snapshot(false, false, false, true)
		// Filter to last 7 days
		if len(snapshot.Days) > 7 {
			snapshot.Days = snapshot.Days[len(snapshot.Days)-7:]
		}
		summary = calculateSummaryFromBuckets(snapshot.Days)
	case "30d", "days":
		snapshot = hm.Snapshot(false, false, false, true)
		summary = calculateSummaryFromBuckets(snapshot.Days)
	default:
		snapshot = hm.Snapshot(false, true, false, false)
		summary = calculateSummaryFromBuckets(snapshot.Minutes)
		rangeParam = "1h"
	}

	c.JSON(http.StatusOK, HistoricalMetricsResponse{
		Range:   rangeParam,
		Data:    snapshot,
		Summary: summary,
	})
}

// GetTPSMetrics returns TPS-specific metrics.
func (h *Handler) GetTPSMetrics(c *gin.Context) {
	granularity := c.DefaultQuery("granularity", "second")

	// Try database first
	if db := usage.GetMetricsDB(); db != nil && db.IsEnabled() {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		data, currentTPS, err := db.GetTPSData(ctx, 60)
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"current_tps": currentTPS,
				"granularity": granularity,
				"data":        data,
				"source":      "database",
			})
			return
		}
	}

	// Fallback to in-memory
	hm := usage.GetHistoricalMetrics()
	if hm == nil {
		c.JSON(http.StatusOK, gin.H{
			"current_tps": 0,
			"data":        []usage.MetricBucket{},
		})
		return
	}

	var data []usage.MetricBucket

	switch granularity {
	case "second":
		snapshot := hm.Snapshot(true, false, false, false)
		data = snapshot.Seconds
	case "minute":
		snapshot := hm.Snapshot(false, true, false, false)
		data = snapshot.Minutes
	default:
		snapshot := hm.Snapshot(true, false, false, false)
		data = snapshot.Seconds
	}

	c.JSON(http.StatusOK, gin.H{
		"current_tps": hm.GetTPS(),
		"granularity": granularity,
		"data":        data,
		"source":      "memory",
	})
}

// GetTPMMetrics returns TPM-specific metrics.
func (h *Handler) GetTPMMetrics(c *gin.Context) {
	granularity := c.DefaultQuery("granularity", "minute")

	// Try database first
	if db := usage.GetMetricsDB(); db != nil && db.IsEnabled() {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		data, currentTPM, err := db.GetTPMData(ctx, 60)
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"current_tpm": currentTPM,
				"granularity": granularity,
				"data":        data,
				"source":      "database",
			})
			return
		}
	}

	// Fallback to in-memory
	hm := usage.GetHistoricalMetrics()
	if hm == nil {
		c.JSON(http.StatusOK, gin.H{
			"current_tpm": 0,
			"data":        []usage.MetricBucket{},
		})
		return
	}

	var data []usage.MetricBucket

	switch granularity {
	case "minute":
		snapshot := hm.Snapshot(false, true, false, false)
		data = snapshot.Minutes
	case "hour":
		snapshot := hm.Snapshot(false, false, true, false)
		data = snapshot.Hours
	default:
		snapshot := hm.Snapshot(false, true, false, false)
		data = snapshot.Minutes
	}

	c.JSON(http.StatusOK, gin.H{
		"current_tpm": hm.GetTPM(),
		"granularity": granularity,
		"data":        data,
		"source":      "memory",
	})
}

// GetTPHMetrics returns TPH-specific metrics.
func (h *Handler) GetTPHMetrics(c *gin.Context) {
	rangeParam := c.DefaultQuery("range", "24h")

	// Try database first
	if db := usage.GetMetricsDB(); db != nil && db.IsEnabled() {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		data, currentTPH, err := db.GetTPHData(ctx, 24)
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"current_tph": currentTPH,
				"range":       rangeParam,
				"data":        data,
				"source":      "database",
			})
			return
		}
	}

	// Fallback to in-memory
	hm := usage.GetHistoricalMetrics()
	if hm == nil {
		c.JSON(http.StatusOK, gin.H{
			"current_tph": 0,
			"data":        []usage.MetricBucket{},
		})
		return
	}

	snapshot := hm.Snapshot(false, false, true, false)

	c.JSON(http.StatusOK, gin.H{
		"current_tph": hm.GetTPH(),
		"range":       rangeParam,
		"data":        snapshot.Hours,
		"source":      "memory",
	})
}

// GetTPDMetrics returns TPD-specific metrics.
func (h *Handler) GetTPDMetrics(c *gin.Context) {
	rangeParam := c.DefaultQuery("range", "30d")

	// Try database first
	if db := usage.GetMetricsDB(); db != nil && db.IsEnabled() {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		limit := 30
		if rangeParam == "7d" {
			limit = 7
		}

		data, currentTPD, err := db.GetTPDData(ctx, limit)
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"current_tpd": currentTPD,
				"range":       rangeParam,
				"data":        data,
				"source":      "database",
			})
			return
		}
	}

	// Fallback to in-memory
	hm := usage.GetHistoricalMetrics()
	if hm == nil {
		c.JSON(http.StatusOK, gin.H{
			"current_tpd": 0,
			"data":        []usage.MetricBucket{},
		})
		return
	}

	snapshot := hm.Snapshot(false, false, false, true)
	data := snapshot.Days

	// Filter to requested range
	if rangeParam == "7d" && len(data) > 7 {
		data = data[len(data)-7:]
	}

	c.JSON(http.StatusOK, gin.H{
		"current_tpd": hm.GetTPD(),
		"range":       rangeParam,
		"data":        data,
		"source":      "memory",
	})
}

func calculateSummaryFromBuckets(buckets []usage.MetricBucket) HistoricalSummary {
	if len(buckets) == 0 {
		return HistoricalSummary{}
	}

	var summary HistoricalSummary
	var latencySum float64
	var latencyCount int64
	var peakRequests int64
	var peakTokens int64

	for _, b := range buckets {
		summary.TotalRequests += b.Requests
		summary.TotalTokens += b.Tokens

		if b.Requests > peakRequests {
			peakRequests = b.Requests
		}
		if b.Tokens > peakTokens {
			peakTokens = b.Tokens
		}

		if b.Requests > 0 {
			latencySum += b.AvgLatency * float64(b.Requests)
			latencyCount += b.Requests
		}
	}

	if latencyCount > 0 {
		summary.AvgLatency = latencySum / float64(latencyCount)
	}

	if len(buckets) > 0 {
		summary.AvgTPS = float64(summary.TotalRequests) / float64(len(buckets))
		summary.AvgTPM = float64(summary.TotalTokens) / float64(len(buckets))
		summary.PeakTPS = float64(peakRequests)
		summary.PeakTPM = peakTokens
	}

	totalAttempts := summary.TotalRequests
	var successCount int64
	for _, b := range buckets {
		successCount += b.SuccessCount
	}
	if totalAttempts > 0 {
		summary.SuccessRate = float64(successCount) / float64(totalAttempts) * 100
	}

	return summary
}
