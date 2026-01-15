// Package usage provides usage tracking and logging functionality for the CLI Proxy API server.
package usage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MetricBucket stores aggregated metrics for a time period.
type MetricBucket struct {
	Timestamp    time.Time              `json:"timestamp"`
	Requests     int64                  `json:"requests"`
	Tokens       int64                  `json:"tokens"`
	InputTokens  int64                  `json:"input_tokens"`
	OutputTokens int64                  `json:"output_tokens"`
	AvgLatency   float64                `json:"avg_latency_ms"`
	SuccessCount int64                  `json:"success_count"`
	FailureCount int64                  `json:"failure_count"`
	ByModel      map[string]ModelBucket `json:"by_model,omitempty"`
}

// ModelBucket stores per-model metrics.
type ModelBucket struct {
	Requests     int64   `json:"requests"`
	Tokens       int64   `json:"tokens"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	AvgLatency   float64 `json:"avg_latency_ms"`
}

// HistoricalMetrics maintains time-series metrics data with multiple granularities.
type HistoricalMetrics struct {
	mu sync.RWMutex

	// Circular buffers for different time granularities
	SecondBuckets [60]MetricBucket // Last 60 seconds (TPS)
	MinuteBuckets [60]MetricBucket // Last 60 minutes (TPM)
	HourBuckets   [24]MetricBucket // Last 24 hours (TPH)
	DayBuckets    [30]MetricBucket // Last 30 days (TPD)

	// Current bucket indices
	lastSecond int64
	lastMinute int64
	lastHour   int64
	lastDay    int64

	// Accumulator for current second
	currentSecond struct {
		requests     int64
		tokens       int64
		inputTokens  int64
		outputTokens int64
		latencySum   float64
		latencyCount int64
		successCount int64
		failureCount int64
		byModel      map[string]*modelAccumulator
	}

	// Persistence path
	persistPath string
}

type modelAccumulator struct {
	requests     int64
	tokens       int64
	inputTokens  int64
	outputTokens int64
	latencySum   float64
	latencyCount int64
}

var (
	globalHistoricalMetrics     *HistoricalMetrics
	globalHistoricalMetricsOnce sync.Once
)

// GetHistoricalMetrics returns the global historical metrics instance.
func GetHistoricalMetrics() *HistoricalMetrics {
	globalHistoricalMetricsOnce.Do(func() {
		globalHistoricalMetrics = NewHistoricalMetrics("")
		globalHistoricalMetrics.startTicker()
	})
	return globalHistoricalMetrics
}

// NewHistoricalMetrics creates a new historical metrics tracker.
func NewHistoricalMetrics(persistPath string) *HistoricalMetrics {
	hm := &HistoricalMetrics{
		persistPath: persistPath,
	}
	hm.currentSecond.byModel = make(map[string]*modelAccumulator)

	// Initialize all buckets with empty model maps
	now := time.Now()
	for i := range hm.SecondBuckets {
		hm.SecondBuckets[i].ByModel = make(map[string]ModelBucket)
		hm.SecondBuckets[i].Timestamp = now.Add(-time.Duration(60-i) * time.Second)
	}
	for i := range hm.MinuteBuckets {
		hm.MinuteBuckets[i].ByModel = make(map[string]ModelBucket)
		hm.MinuteBuckets[i].Timestamp = now.Add(-time.Duration(60-i) * time.Minute)
	}
	for i := range hm.HourBuckets {
		hm.HourBuckets[i].ByModel = make(map[string]ModelBucket)
		hm.HourBuckets[i].Timestamp = now.Add(-time.Duration(24-i) * time.Hour)
	}
	for i := range hm.DayBuckets {
		hm.DayBuckets[i].ByModel = make(map[string]ModelBucket)
		hm.DayBuckets[i].Timestamp = now.Add(-time.Duration(30-i) * 24 * time.Hour)
	}

	// Try to load persisted data
	if persistPath != "" {
		hm.load()
	}

	return hm
}

// Record records a request to the historical metrics.
func (hm *HistoricalMetrics) Record(model string, inputTokens, outputTokens int64, latencyMs float64, success bool) {
	if hm == nil {
		return
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	totalTokens := inputTokens + outputTokens

	hm.currentSecond.requests++
	hm.currentSecond.tokens += totalTokens
	hm.currentSecond.inputTokens += inputTokens
	hm.currentSecond.outputTokens += outputTokens
	hm.currentSecond.latencySum += latencyMs
	hm.currentSecond.latencyCount++

	if success {
		hm.currentSecond.successCount++
	} else {
		hm.currentSecond.failureCount++
	}

	// Update model stats
	if model != "" {
		if hm.currentSecond.byModel == nil {
			hm.currentSecond.byModel = make(map[string]*modelAccumulator)
		}
		acc, ok := hm.currentSecond.byModel[model]
		if !ok {
			acc = &modelAccumulator{}
			hm.currentSecond.byModel[model] = acc
		}
		acc.requests++
		acc.tokens += totalTokens
		acc.inputTokens += inputTokens
		acc.outputTokens += outputTokens
		acc.latencySum += latencyMs
		acc.latencyCount++
	}
}

// startTicker starts background ticker to roll over buckets.
func (hm *HistoricalMetrics) startTicker() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			hm.tick()
		}
	}()
}

// tick is called every second to roll over buckets.
func (hm *HistoricalMetrics) tick() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	now := time.Now()
	currentSecond := now.Unix()
	currentMinute := now.Unix() / 60
	currentHour := now.Unix() / 3600
	currentDay := now.Unix() / 86400

	// Roll second bucket
	idx := currentSecond % 60
	avgLatency := float64(0)
	if hm.currentSecond.latencyCount > 0 {
		avgLatency = hm.currentSecond.latencySum / float64(hm.currentSecond.latencyCount)
	}

	bucket := MetricBucket{
		Timestamp:    now,
		Requests:     hm.currentSecond.requests,
		Tokens:       hm.currentSecond.tokens,
		InputTokens:  hm.currentSecond.inputTokens,
		OutputTokens: hm.currentSecond.outputTokens,
		AvgLatency:   avgLatency,
		SuccessCount: hm.currentSecond.successCount,
		FailureCount: hm.currentSecond.failureCount,
		ByModel:      make(map[string]ModelBucket),
	}

	for model, acc := range hm.currentSecond.byModel {
		modelAvgLatency := float64(0)
		if acc.latencyCount > 0 {
			modelAvgLatency = acc.latencySum / float64(acc.latencyCount)
		}
		bucket.ByModel[model] = ModelBucket{
			Requests:     acc.requests,
			Tokens:       acc.tokens,
			InputTokens:  acc.inputTokens,
			OutputTokens: acc.outputTokens,
			AvgLatency:   modelAvgLatency,
		}
	}

	hm.SecondBuckets[idx] = bucket

	// Persist to database if available
	if db := GetMetricsDB(); db != nil && db.IsEnabled() {
		modelMetrics := make(map[string]ModelMetricRecord)
		for model, mb := range bucket.ByModel {
			modelMetrics[model] = ModelMetricRecord{
				ModelName:    model,
				Requests:     mb.Requests,
				Tokens:       mb.Tokens,
				InputTokens:  mb.InputTokens,
				OutputTokens: mb.OutputTokens,
				AvgLatencyMs: mb.AvgLatency,
			}
		}
		db.Record(MetricRecord{
			Timestamp:    bucket.Timestamp,
			Granularity:  "second",
			Requests:     bucket.Requests,
			Tokens:       bucket.Tokens,
			InputTokens:  bucket.InputTokens,
			OutputTokens: bucket.OutputTokens,
			AvgLatencyMs: bucket.AvgLatency,
			SuccessCount: bucket.SuccessCount,
			FailureCount: bucket.FailureCount,
			ModelMetrics: modelMetrics,
		})
	}

	// Reset current second accumulator
	hm.currentSecond.requests = 0
	hm.currentSecond.tokens = 0
	hm.currentSecond.inputTokens = 0
	hm.currentSecond.outputTokens = 0
	hm.currentSecond.latencySum = 0
	hm.currentSecond.latencyCount = 0
	hm.currentSecond.successCount = 0
	hm.currentSecond.failureCount = 0
	hm.currentSecond.byModel = make(map[string]*modelAccumulator)

	hm.lastSecond = currentSecond

	// Roll minute bucket (aggregate last 60 seconds)
	if currentMinute != hm.lastMinute {
		hm.rollMinuteBucket(now, currentMinute)
		hm.lastMinute = currentMinute
	}

	// Roll hour bucket (aggregate last 60 minutes)
	if currentHour != hm.lastHour {
		hm.rollHourBucket(now, currentHour)
		hm.lastHour = currentHour
	}

	// Roll day bucket (aggregate last 24 hours)
	if currentDay != hm.lastDay {
		hm.rollDayBucket(now, currentDay)
		hm.lastDay = currentDay

		// Persist on day rollover
		if hm.persistPath != "" {
			go hm.persist()
		}
	}
}

func (hm *HistoricalMetrics) rollMinuteBucket(now time.Time, currentMinute int64) {
	idx := currentMinute % 60
	bucket := hm.aggregateSeconds()
	bucket.Timestamp = now
	hm.MinuteBuckets[idx] = bucket

	// Persist minute bucket to database
	if db := GetMetricsDB(); db != nil && db.IsEnabled() {
		modelMetrics := make(map[string]ModelMetricRecord)
		for model, mb := range bucket.ByModel {
			modelMetrics[model] = ModelMetricRecord{
				ModelName:    model,
				Requests:     mb.Requests,
				Tokens:       mb.Tokens,
				InputTokens:  mb.InputTokens,
				OutputTokens: mb.OutputTokens,
				AvgLatencyMs: mb.AvgLatency,
			}
		}
		db.Record(MetricRecord{
			Timestamp:    bucket.Timestamp,
			Granularity:  "minute",
			Requests:     bucket.Requests,
			Tokens:       bucket.Tokens,
			InputTokens:  bucket.InputTokens,
			OutputTokens: bucket.OutputTokens,
			AvgLatencyMs: bucket.AvgLatency,
			SuccessCount: bucket.SuccessCount,
			FailureCount: bucket.FailureCount,
			ModelMetrics: modelMetrics,
		})
	}
}

func (hm *HistoricalMetrics) rollHourBucket(now time.Time, currentHour int64) {
	idx := currentHour % 24
	bucket := hm.aggregateMinutes()
	bucket.Timestamp = now
	hm.HourBuckets[idx] = bucket

	// Persist hour bucket to database
	if db := GetMetricsDB(); db != nil && db.IsEnabled() {
		modelMetrics := make(map[string]ModelMetricRecord)
		for model, mb := range bucket.ByModel {
			modelMetrics[model] = ModelMetricRecord{
				ModelName:    model,
				Requests:     mb.Requests,
				Tokens:       mb.Tokens,
				InputTokens:  mb.InputTokens,
				OutputTokens: mb.OutputTokens,
				AvgLatencyMs: mb.AvgLatency,
			}
		}
		db.Record(MetricRecord{
			Timestamp:    bucket.Timestamp,
			Granularity:  "hour",
			Requests:     bucket.Requests,
			Tokens:       bucket.Tokens,
			InputTokens:  bucket.InputTokens,
			OutputTokens: bucket.OutputTokens,
			AvgLatencyMs: bucket.AvgLatency,
			SuccessCount: bucket.SuccessCount,
			FailureCount: bucket.FailureCount,
			ModelMetrics: modelMetrics,
		})
	}
}

func (hm *HistoricalMetrics) rollDayBucket(now time.Time, currentDay int64) {
	idx := currentDay % 30
	bucket := hm.aggregateHours()
	bucket.Timestamp = now
	hm.DayBuckets[idx] = bucket

	// Persist day bucket to database
	if db := GetMetricsDB(); db != nil && db.IsEnabled() {
		modelMetrics := make(map[string]ModelMetricRecord)
		for model, mb := range bucket.ByModel {
			modelMetrics[model] = ModelMetricRecord{
				ModelName:    model,
				Requests:     mb.Requests,
				Tokens:       mb.Tokens,
				InputTokens:  mb.InputTokens,
				OutputTokens: mb.OutputTokens,
				AvgLatencyMs: mb.AvgLatency,
			}
		}
		db.Record(MetricRecord{
			Timestamp:    bucket.Timestamp,
			Granularity:  "day",
			Requests:     bucket.Requests,
			Tokens:       bucket.Tokens,
			InputTokens:  bucket.InputTokens,
			OutputTokens: bucket.OutputTokens,
			AvgLatencyMs: bucket.AvgLatency,
			SuccessCount: bucket.SuccessCount,
			FailureCount: bucket.FailureCount,
			ModelMetrics: modelMetrics,
		})
	}
}

func (hm *HistoricalMetrics) aggregateSeconds() MetricBucket {
	result := MetricBucket{ByModel: make(map[string]ModelBucket)}
	var latencySum float64
	var latencyCount int64
	modelLatencySum := make(map[string]float64)
	modelLatencyCount := make(map[string]int64)

	for _, b := range hm.SecondBuckets {
		result.Requests += b.Requests
		result.Tokens += b.Tokens
		result.InputTokens += b.InputTokens
		result.OutputTokens += b.OutputTokens
		result.SuccessCount += b.SuccessCount
		result.FailureCount += b.FailureCount
		if b.Requests > 0 {
			latencySum += b.AvgLatency * float64(b.Requests)
			latencyCount += b.Requests
		}

		for model, mb := range b.ByModel {
			existing := result.ByModel[model]
			existing.Requests += mb.Requests
			existing.Tokens += mb.Tokens
			existing.InputTokens += mb.InputTokens
			existing.OutputTokens += mb.OutputTokens
			if mb.Requests > 0 {
				modelLatencySum[model] += mb.AvgLatency * float64(mb.Requests)
				modelLatencyCount[model] += mb.Requests
			}
			result.ByModel[model] = existing
		}
	}

	if latencyCount > 0 {
		result.AvgLatency = latencySum / float64(latencyCount)
	}
	for model, existing := range result.ByModel {
		if count := modelLatencyCount[model]; count > 0 {
			existing.AvgLatency = modelLatencySum[model] / float64(count)
			result.ByModel[model] = existing
		}
	}

	return result
}

func (hm *HistoricalMetrics) aggregateMinutes() MetricBucket {
	result := MetricBucket{ByModel: make(map[string]ModelBucket)}
	var latencySum float64
	var latencyCount int64
	modelLatencySum := make(map[string]float64)
	modelLatencyCount := make(map[string]int64)

	for _, b := range hm.MinuteBuckets {
		result.Requests += b.Requests
		result.Tokens += b.Tokens
		result.InputTokens += b.InputTokens
		result.OutputTokens += b.OutputTokens
		result.SuccessCount += b.SuccessCount
		result.FailureCount += b.FailureCount
		if b.Requests > 0 {
			latencySum += b.AvgLatency * float64(b.Requests)
			latencyCount += b.Requests
		}

		for model, mb := range b.ByModel {
			existing := result.ByModel[model]
			existing.Requests += mb.Requests
			existing.Tokens += mb.Tokens
			existing.InputTokens += mb.InputTokens
			existing.OutputTokens += mb.OutputTokens
			if mb.Requests > 0 {
				modelLatencySum[model] += mb.AvgLatency * float64(mb.Requests)
				modelLatencyCount[model] += mb.Requests
			}
			result.ByModel[model] = existing
		}
	}

	if latencyCount > 0 {
		result.AvgLatency = latencySum / float64(latencyCount)
	}
	for model, existing := range result.ByModel {
		if count := modelLatencyCount[model]; count > 0 {
			existing.AvgLatency = modelLatencySum[model] / float64(count)
			result.ByModel[model] = existing
		}
	}

	return result
}

func (hm *HistoricalMetrics) aggregateHours() MetricBucket {
	result := MetricBucket{ByModel: make(map[string]ModelBucket)}
	var latencySum float64
	var latencyCount int64
	modelLatencySum := make(map[string]float64)
	modelLatencyCount := make(map[string]int64)

	for _, b := range hm.HourBuckets {
		result.Requests += b.Requests
		result.Tokens += b.Tokens
		result.InputTokens += b.InputTokens
		result.OutputTokens += b.OutputTokens
		result.SuccessCount += b.SuccessCount
		result.FailureCount += b.FailureCount
		if b.Requests > 0 {
			latencySum += b.AvgLatency * float64(b.Requests)
			latencyCount += b.Requests
		}

		for model, mb := range b.ByModel {
			existing := result.ByModel[model]
			existing.Requests += mb.Requests
			existing.Tokens += mb.Tokens
			existing.InputTokens += mb.InputTokens
			existing.OutputTokens += mb.OutputTokens
			if mb.Requests > 0 {
				modelLatencySum[model] += mb.AvgLatency * float64(mb.Requests)
				modelLatencyCount[model] += mb.Requests
			}
			result.ByModel[model] = existing
		}
	}

	if latencyCount > 0 {
		result.AvgLatency = latencySum / float64(latencyCount)
	}
	for model, existing := range result.ByModel {
		if count := modelLatencyCount[model]; count > 0 {
			existing.AvgLatency = modelLatencySum[model] / float64(count)
			result.ByModel[model] = existing
		}
	}

	return result
}

// HistoricalSnapshot represents a point-in-time view of historical metrics.
type HistoricalSnapshot struct {
	Seconds []MetricBucket `json:"seconds,omitempty"` // Last 60 seconds
	Minutes []MetricBucket `json:"minutes,omitempty"` // Last 60 minutes
	Hours   []MetricBucket `json:"hours,omitempty"`   // Last 24 hours
	Days    []MetricBucket `json:"days,omitempty"`    // Last 30 days
}

// Snapshot returns a copy of the historical metrics.
func (hm *HistoricalMetrics) Snapshot(includeSeconds, includeMinutes, includeHours, includeDays bool) HistoricalSnapshot {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	snapshot := HistoricalSnapshot{}

	now := time.Now()

	if includeSeconds {
		snapshot.Seconds = make([]MetricBucket, 60)
		currentSecond := now.Unix() % 60
		for i := 0; i < 60; i++ {
			idx := (currentSecond - int64(59-i) + 60) % 60
			snapshot.Seconds[i] = hm.copyBucket(hm.SecondBuckets[idx])
		}
	}

	if includeMinutes {
		snapshot.Minutes = make([]MetricBucket, 60)
		currentMinute := (now.Unix() / 60) % 60
		for i := 0; i < 60; i++ {
			idx := (currentMinute - int64(59-i) + 60) % 60
			snapshot.Minutes[i] = hm.copyBucket(hm.MinuteBuckets[idx])
		}
	}

	if includeHours {
		snapshot.Hours = make([]MetricBucket, 24)
		currentHour := (now.Unix() / 3600) % 24
		for i := 0; i < 24; i++ {
			idx := (currentHour - int64(23-i) + 24) % 24
			snapshot.Hours[i] = hm.copyBucket(hm.HourBuckets[idx])
		}
	}

	if includeDays {
		snapshot.Days = make([]MetricBucket, 30)
		currentDay := (now.Unix() / 86400) % 30
		for i := 0; i < 30; i++ {
			idx := (currentDay - int64(29-i) + 30) % 30
			snapshot.Days[i] = hm.copyBucket(hm.DayBuckets[idx])
		}
	}

	return snapshot
}

func (hm *HistoricalMetrics) copyBucket(b MetricBucket) MetricBucket {
	copy := MetricBucket{
		Timestamp:    b.Timestamp,
		Requests:     b.Requests,
		Tokens:       b.Tokens,
		InputTokens:  b.InputTokens,
		OutputTokens: b.OutputTokens,
		AvgLatency:   b.AvgLatency,
		SuccessCount: b.SuccessCount,
		FailureCount: b.FailureCount,
		ByModel:      make(map[string]ModelBucket, len(b.ByModel)),
	}
	for k, v := range b.ByModel {
		copy.ByModel[k] = v
	}
	return copy
}

// GetTPS returns the current transactions per second (average of last 10 seconds).
func (hm *HistoricalMetrics) GetTPS() float64 {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	now := time.Now()
	currentSecond := now.Unix() % 60
	var total int64

	for i := 0; i < 10; i++ {
		idx := (currentSecond - int64(i) + 60) % 60
		total += hm.SecondBuckets[idx].Requests
	}

	return float64(total) / 10.0
}

// GetTPM returns the current tokens per minute.
func (hm *HistoricalMetrics) GetTPM() int64 {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var total int64
	for _, b := range hm.SecondBuckets {
		total += b.Tokens
	}
	return total
}

// GetTPH returns the tokens for the last hour.
func (hm *HistoricalMetrics) GetTPH() int64 {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var total int64
	for _, b := range hm.MinuteBuckets {
		total += b.Tokens
	}
	return total
}

// GetTPD returns the tokens for the last day.
func (hm *HistoricalMetrics) GetTPD() int64 {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var total int64
	for _, b := range hm.HourBuckets {
		total += b.Tokens
	}
	return total
}

// persist saves the historical metrics to disk.
func (hm *HistoricalMetrics) persist() {
	if hm.persistPath == "" {
		return
	}

	hm.mu.RLock()
	data, err := json.Marshal(hm)
	hm.mu.RUnlock()

	if err != nil {
		return
	}

	dir := filepath.Dir(hm.persistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	_ = os.WriteFile(hm.persistPath, data, 0644)
}

// load restores historical metrics from disk.
func (hm *HistoricalMetrics) load() {
	if hm.persistPath == "" {
		return
	}

	data, err := os.ReadFile(hm.persistPath)
	if err != nil {
		return
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	_ = json.Unmarshal(data, hm)
}
