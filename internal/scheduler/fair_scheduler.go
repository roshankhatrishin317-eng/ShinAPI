// Package scheduler provides fair scheduling and weighted queuing for API requests.
package scheduler

import (
	"container/heap"
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// FairScheduler implements weighted fair queuing for API requests.
// It ensures that API keys with higher weights get proportionally more bandwidth
// while preventing starvation of lower-priority keys.
type FairScheduler struct {
	mu            sync.Mutex
	queues        map[string]*requestQueue
	weights       map[string]int
	defaultWeight int
	maxQueueSize  int
	metrics       *SchedulerMetrics

	// Virtual time for fair scheduling
	virtualTime atomic.Int64

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// requestQueue holds pending requests for a single API key.
type requestQueue struct {
	apiKey      string
	weight      int
	virtualTime int64
	requests    []*scheduledRequest
	totalTokens int64
}

// scheduledRequest represents a queued request.
type scheduledRequest struct {
	ctx        context.Context
	priority   int
	tokens     int64 // estimated tokens for this request
	enqueuedAt time.Time
	callback   func() error
	done       chan error
}

// SchedulerConfig configures the fair scheduler.
type SchedulerConfig struct {
	// DefaultWeight is the default weight for API keys without explicit config
	DefaultWeight int
	// MaxQueueSize is the maximum number of pending requests per queue
	MaxQueueSize int
	// MaxConcurrent is the maximum number of concurrent requests
	MaxConcurrent int
	// QueueTimeout is the maximum time a request can wait in queue
	QueueTimeout time.Duration
}

// DefaultSchedulerConfig returns sensible defaults.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		DefaultWeight: 100,
		MaxQueueSize:  1000,
		MaxConcurrent: 50,
		QueueTimeout:  60 * time.Second,
	}
}

// NewFairScheduler creates a new fair scheduler.
func NewFairScheduler(cfg SchedulerConfig) *FairScheduler {
	if cfg.DefaultWeight <= 0 {
		cfg.DefaultWeight = 100
	}
	if cfg.MaxQueueSize <= 0 {
		cfg.MaxQueueSize = 1000
	}

	fs := &FairScheduler{
		queues:        make(map[string]*requestQueue),
		weights:       make(map[string]int),
		defaultWeight: cfg.DefaultWeight,
		maxQueueSize:  cfg.MaxQueueSize,
		metrics:       NewSchedulerMetrics(),
		stopCh:        make(chan struct{}),
	}

	return fs
}

// SetWeight sets the weight for an API key.
// Higher weights get more bandwidth.
func (fs *FairScheduler) SetWeight(apiKey string, weight int) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if weight <= 0 {
		weight = fs.defaultWeight
	}
	fs.weights[apiKey] = weight

	if q, exists := fs.queues[apiKey]; exists {
		q.weight = weight
	}
}

// GetWeight returns the weight for an API key.
func (fs *FairScheduler) GetWeight(apiKey string) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if weight, exists := fs.weights[apiKey]; exists {
		return weight
	}
	return fs.defaultWeight
}

// Schedule queues a request for execution with fair scheduling.
// Returns an error if the queue is full or the context is cancelled.
func (fs *FairScheduler) Schedule(ctx context.Context, apiKey string, estimatedTokens int64, callback func() error) error {
	fs.mu.Lock()

	q, exists := fs.queues[apiKey]
	if !exists {
		weight := fs.defaultWeight
		if w, ok := fs.weights[apiKey]; ok {
			weight = w
		}
		q = &requestQueue{
			apiKey:   apiKey,
			weight:   weight,
			requests: make([]*scheduledRequest, 0, 100),
		}
		fs.queues[apiKey] = q
	}

	if len(q.requests) >= fs.maxQueueSize {
		fs.mu.Unlock()
		fs.metrics.RecordRejection(apiKey)
		return ErrQueueFull
	}

	req := &scheduledRequest{
		ctx:        ctx,
		tokens:     estimatedTokens,
		enqueuedAt: time.Now(),
		callback:   callback,
		done:       make(chan error, 1),
	}

	q.requests = append(q.requests, req)
	q.totalTokens += estimatedTokens
	fs.metrics.RecordEnqueue(apiKey)

	fs.mu.Unlock()

	// Wait for execution
	select {
	case err := <-req.done:
		return err
	case <-ctx.Done():
		fs.removeRequest(apiKey, req)
		return ctx.Err()
	}
}

// removeRequest removes a cancelled request from the queue.
func (fs *FairScheduler) removeRequest(apiKey string, req *scheduledRequest) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	q, exists := fs.queues[apiKey]
	if !exists {
		return
	}

	for i, r := range q.requests {
		if r == req {
			q.requests = append(q.requests[:i], q.requests[i+1:]...)
			q.totalTokens -= req.tokens
			fs.metrics.RecordCancellation(apiKey)
			break
		}
	}
}

// NextRequest returns the next request to execute based on fair scheduling.
// Uses weighted fair queuing where virtual time advances slower for higher-weight keys.
func (fs *FairScheduler) NextRequest() (*scheduledRequest, string, bool) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	var bestQueue *requestQueue
	var bestVirtualFinish int64 = -1

	globalVTime := fs.virtualTime.Load()

	for _, q := range fs.queues {
		if len(q.requests) == 0 {
			continue
		}

		// Calculate virtual finish time for the next request
		// Lower weight = higher virtual time advancement = less priority
		req := q.requests[0]
		virtualStart := max(q.virtualTime, globalVTime)
		virtualFinish := virtualStart + (req.tokens * 1000 / int64(q.weight))

		if bestQueue == nil || virtualFinish < bestVirtualFinish {
			bestQueue = q
			bestVirtualFinish = virtualFinish
		}
	}

	if bestQueue == nil {
		return nil, "", false
	}

	// Pop the request
	req := bestQueue.requests[0]
	bestQueue.requests = bestQueue.requests[1:]
	bestQueue.totalTokens -= req.tokens
	bestQueue.virtualTime = bestVirtualFinish

	// Update global virtual time
	fs.virtualTime.Store(bestVirtualFinish)

	fs.metrics.RecordDequeue(bestQueue.apiKey)

	return req, bestQueue.apiKey, true
}

// ExecuteNext executes the next scheduled request.
func (fs *FairScheduler) ExecuteNext() bool {
	req, apiKey, ok := fs.NextRequest()
	if !ok {
		return false
	}

	// Check if context is still valid
	if req.ctx.Err() != nil {
		req.done <- req.ctx.Err()
		return true
	}

	start := time.Now()
	err := req.callback()
	duration := time.Since(start)

	fs.metrics.RecordExecution(apiKey, duration, err == nil)
	req.done <- err

	return true
}

// RunWorker starts a worker that processes requests continuously.
func (fs *FairScheduler) RunWorker(ctx context.Context) {
	fs.wg.Add(1)
	defer fs.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fs.stopCh:
			return
		default:
			if !fs.ExecuteNext() {
				// No requests, sleep briefly
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
}

// Start starts the scheduler with the specified number of workers.
func (fs *FairScheduler) Start(ctx context.Context, workers int) {
	if workers <= 0 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		go fs.RunWorker(ctx)
	}
}

// Stop stops all workers.
func (fs *FairScheduler) Stop() {
	close(fs.stopCh)
	fs.wg.Wait()
}

// Stats returns scheduler statistics.
func (fs *FairScheduler) Stats() SchedulerStats {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	stats := SchedulerStats{
		Queues:      make(map[string]QueueStats),
		VirtualTime: fs.virtualTime.Load(),
	}

	for apiKey, q := range fs.queues {
		stats.Queues[apiKey] = QueueStats{
			PendingRequests: len(q.requests),
			TotalTokens:     q.totalTokens,
			Weight:          q.weight,
			VirtualTime:     q.virtualTime,
		}
		stats.TotalPending += len(q.requests)
	}

	stats.Metrics = fs.metrics.Snapshot()
	return stats
}

// SchedulerStats holds scheduler statistics.
type SchedulerStats struct {
	Queues       map[string]QueueStats `json:"queues"`
	TotalPending int                   `json:"total_pending"`
	VirtualTime  int64                 `json:"virtual_time"`
	Metrics      MetricsSnapshot       `json:"metrics"`
}

// QueueStats holds statistics for a single queue.
type QueueStats struct {
	PendingRequests int   `json:"pending_requests"`
	TotalTokens     int64 `json:"total_tokens"`
	Weight          int   `json:"weight"`
	VirtualTime     int64 `json:"virtual_time"`
}

// ErrQueueFull is returned when a queue is at capacity.
var ErrQueueFull = &SchedulerError{Message: "queue is full"}

// SchedulerError represents a scheduler error.
type SchedulerError struct {
	Message string
}

func (e *SchedulerError) Error() string {
	return e.Message
}

// SchedulerMetrics tracks scheduler performance metrics.
type SchedulerMetrics struct {
	mu sync.RWMutex

	totalEnqueued   int64
	totalDequeued   int64
	totalExecuted   int64
	totalRejected   int64
	totalCancelled  int64
	totalSuccessful int64
	totalFailed     int64

	queueTimes    []time.Duration
	executeTimes  []time.Duration
	keyMetrics    map[string]*keyMetrics
}

type keyMetrics struct {
	enqueued   int64
	dequeued   int64
	executed   int64
	rejected   int64
	cancelled  int64
	successful int64
	failed     int64
}

// NewSchedulerMetrics creates a new metrics instance.
func NewSchedulerMetrics() *SchedulerMetrics {
	return &SchedulerMetrics{
		queueTimes:   make([]time.Duration, 0, 1000),
		executeTimes: make([]time.Duration, 0, 1000),
		keyMetrics:   make(map[string]*keyMetrics),
	}
}

func (m *SchedulerMetrics) getKeyMetrics(apiKey string) *keyMetrics {
	if km, exists := m.keyMetrics[apiKey]; exists {
		return km
	}
	km := &keyMetrics{}
	m.keyMetrics[apiKey] = km
	return km
}

// RecordEnqueue records a request being enqueued.
func (m *SchedulerMetrics) RecordEnqueue(apiKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalEnqueued++
	m.getKeyMetrics(apiKey).enqueued++
}

// RecordDequeue records a request being dequeued.
func (m *SchedulerMetrics) RecordDequeue(apiKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalDequeued++
	m.getKeyMetrics(apiKey).dequeued++
}

// RecordRejection records a request being rejected.
func (m *SchedulerMetrics) RecordRejection(apiKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalRejected++
	m.getKeyMetrics(apiKey).rejected++
}

// RecordCancellation records a request being cancelled.
func (m *SchedulerMetrics) RecordCancellation(apiKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalCancelled++
	m.getKeyMetrics(apiKey).cancelled++
}

// RecordExecution records a request execution.
func (m *SchedulerMetrics) RecordExecution(apiKey string, duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalExecuted++
	km := m.getKeyMetrics(apiKey)
	km.executed++

	if success {
		m.totalSuccessful++
		km.successful++
	} else {
		m.totalFailed++
		km.failed++
	}

	// Keep last 1000 execution times
	if len(m.executeTimes) >= 1000 {
		m.executeTimes = m.executeTimes[1:]
	}
	m.executeTimes = append(m.executeTimes, duration)
}

// Snapshot returns a copy of the current metrics.
func (m *SchedulerMetrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return MetricsSnapshot{
		TotalEnqueued:   m.totalEnqueued,
		TotalDequeued:   m.totalDequeued,
		TotalExecuted:   m.totalExecuted,
		TotalRejected:   m.totalRejected,
		TotalCancelled:  m.totalCancelled,
		TotalSuccessful: m.totalSuccessful,
		TotalFailed:     m.totalFailed,
	}
}

// MetricsSnapshot holds a snapshot of scheduler metrics.
type MetricsSnapshot struct {
	TotalEnqueued   int64 `json:"total_enqueued"`
	TotalDequeued   int64 `json:"total_dequeued"`
	TotalExecuted   int64 `json:"total_executed"`
	TotalRejected   int64 `json:"total_rejected"`
	TotalCancelled  int64 `json:"total_cancelled"`
	TotalSuccessful int64 `json:"total_successful"`
	TotalFailed     int64 `json:"total_failed"`
}

// PriorityQueue implements a priority queue for requests.
type PriorityQueue []*scheduledRequest

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// Higher priority first, then earlier enqueue time
	if pq[i].priority != pq[j].priority {
		return pq[i].priority > pq[j].priority
	}
	return pq[i].enqueuedAt.Before(pq[j].enqueuedAt)
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*scheduledRequest))
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

// Global scheduler instance
var (
	globalScheduler     *FairScheduler
	globalSchedulerOnce sync.Once
)

// GetScheduler returns the global fair scheduler.
func GetScheduler() *FairScheduler {
	globalSchedulerOnce.Do(func() {
		globalScheduler = NewFairScheduler(DefaultSchedulerConfig())
	})
	return globalScheduler
}

// InitScheduler initializes the global scheduler with custom config.
func InitScheduler(cfg SchedulerConfig) *FairScheduler {
	globalSchedulerOnce.Do(func() {
		globalScheduler = NewFairScheduler(cfg)
	})
	return globalScheduler
}

// Ensure PriorityQueue implements heap.Interface
var _ heap.Interface = (*PriorityQueue)(nil)
