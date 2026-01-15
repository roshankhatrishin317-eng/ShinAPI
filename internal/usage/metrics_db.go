// Package usage provides usage tracking and metrics persistence for the CLI Proxy API server.
package usage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	log "github.com/sirupsen/logrus"
)

// MetricsDB provides PostgreSQL-backed metrics persistence.
type MetricsDB struct {
	pool   *pgxpool.Pool
	config config.MetricsDBConfig

	// Buffer for batching writes
	mu          sync.Mutex
	buffer      []MetricRecord
	lastFlush   time.Time
	flushTicker *time.Ticker
	flushCh     chan struct{}
	done        chan struct{}
	closeOnce   sync.Once
	wg          sync.WaitGroup
}

// MetricRecord represents a single metrics record to be persisted.
type MetricRecord struct {
	Timestamp    time.Time
	Granularity  string // "second", "minute", "hour", "day"
	Requests     int64
	Tokens       int64
	InputTokens  int64
	OutputTokens int64
	AvgLatencyMs float64
	SuccessCount int64
	FailureCount int64
	ModelMetrics map[string]ModelMetricRecord
}

// ModelMetricRecord represents per-model metrics.
type ModelMetricRecord struct {
	ModelName    string
	Requests     int64
	Tokens       int64
	InputTokens  int64
	OutputTokens int64
	AvgLatencyMs float64
}

var (
	globalMetricsDB     *MetricsDB
	globalMetricsDBOnce sync.Once
	globalMetricsDBMu   sync.RWMutex
)

// InitMetricsDB initializes the global metrics database connection.
func InitMetricsDB(cfg config.MetricsDBConfig) error {
	if !cfg.Enabled || cfg.DSN == "" {
		log.Info("Metrics database is disabled or DSN not configured")
		return nil
	}

	globalMetricsDBMu.Lock()
	defer globalMetricsDBMu.Unlock()

	db, err := NewMetricsDB(cfg)
	if err != nil {
		return err
	}

	globalMetricsDB = db
	log.Info("Metrics database initialized successfully")
	return nil
}

// GetMetricsDB returns the global metrics database instance.
func GetMetricsDB() *MetricsDB {
	globalMetricsDBMu.RLock()
	defer globalMetricsDBMu.RUnlock()
	return globalMetricsDB
}

// NewMetricsDB creates a new metrics database connection.
func NewMetricsDB(cfg config.MetricsDBConfig) (*MetricsDB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("metrics database DSN is required")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	// Set connection pool limits
	maxConns := cfg.MaxConnections
	if maxConns <= 0 {
		maxConns = 10
	}
	poolConfig.MaxConns = int32(maxConns)
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db := &MetricsDB{
		pool:      pool,
		config:    cfg,
		buffer:    make([]MetricRecord, 0, cfg.BatchSize),
		lastFlush: time.Now(),
		flushCh:   make(chan struct{}, 1),
		done:      make(chan struct{}),
	}

	// Initialize schema
	if err := db.initSchema(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Start background flush ticker
	flushInterval := cfg.FlushIntervalSeconds
	if flushInterval <= 0 {
		flushInterval = 5
	}
	db.flushTicker = time.NewTicker(time.Duration(flushInterval) * time.Second)
	db.wg.Add(1)
	go func() {
		defer db.wg.Done()
		db.flushLoop()
	}()

	// Start retention cleanup
	db.wg.Add(1)
	go func() {
		defer db.wg.Done()
		db.cleanupLoop()
	}()

	return db, nil
}

// initSchema creates the database tables if they don't exist.
func (db *MetricsDB) initSchema(ctx context.Context) error {
	schema := `
		CREATE TABLE IF NOT EXISTS metrics_snapshots (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			granularity VARCHAR(20) NOT NULL,
			requests BIGINT NOT NULL DEFAULT 0,
			tokens BIGINT NOT NULL DEFAULT 0,
			input_tokens BIGINT NOT NULL DEFAULT 0,
			output_tokens BIGINT NOT NULL DEFAULT 0,
			success_count BIGINT NOT NULL DEFAULT 0,
			failure_count BIGINT NOT NULL DEFAULT 0,
			avg_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_metrics_snapshots_timestamp 
			ON metrics_snapshots(timestamp DESC);

		CREATE INDEX IF NOT EXISTS idx_metrics_snapshots_granularity 
			ON metrics_snapshots(granularity, timestamp DESC);

		CREATE TABLE IF NOT EXISTS model_metrics (
			id SERIAL PRIMARY KEY,
			snapshot_id INTEGER REFERENCES metrics_snapshots(id) ON DELETE CASCADE,
			model_name VARCHAR(255) NOT NULL,
			requests BIGINT NOT NULL DEFAULT 0,
			tokens BIGINT NOT NULL DEFAULT 0,
			input_tokens BIGINT NOT NULL DEFAULT 0,
			output_tokens BIGINT NOT NULL DEFAULT 0,
			avg_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0
		);

		CREATE INDEX IF NOT EXISTS idx_model_metrics_snapshot 
			ON model_metrics(snapshot_id);

		CREATE INDEX IF NOT EXISTS idx_model_metrics_model 
			ON model_metrics(model_name);

		CREATE TABLE IF NOT EXISTS hourly_aggregates (
			id SERIAL PRIMARY KEY,
			hour_start TIMESTAMPTZ NOT NULL UNIQUE,
			total_requests BIGINT NOT NULL DEFAULT 0,
			total_tokens BIGINT NOT NULL DEFAULT 0,
			total_input_tokens BIGINT NOT NULL DEFAULT 0,
			total_output_tokens BIGINT NOT NULL DEFAULT 0,
			success_count BIGINT NOT NULL DEFAULT 0,
			failure_count BIGINT NOT NULL DEFAULT 0,
			avg_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_hourly_aggregates_hour 
			ON hourly_aggregates(hour_start DESC);

		CREATE TABLE IF NOT EXISTS daily_aggregates (
			id SERIAL PRIMARY KEY,
			date DATE NOT NULL UNIQUE,
			total_requests BIGINT NOT NULL DEFAULT 0,
			total_tokens BIGINT NOT NULL DEFAULT 0,
			total_input_tokens BIGINT NOT NULL DEFAULT 0,
			total_output_tokens BIGINT NOT NULL DEFAULT 0,
			success_count BIGINT NOT NULL DEFAULT 0,
			failure_count BIGINT NOT NULL DEFAULT 0,
			avg_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
			peak_tps DOUBLE PRECISION NOT NULL DEFAULT 0,
			peak_tpm BIGINT NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_daily_aggregates_date 
			ON daily_aggregates(date DESC);
	`

	_, err := db.pool.Exec(ctx, schema)
	return err
}

// Record adds a metric record to the buffer for batch insertion.
func (db *MetricsDB) Record(record MetricRecord) {
	if db == nil || db.pool == nil {
		return
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	db.buffer = append(db.buffer, record)

	// Flush if buffer is full
	batchSize := db.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	if len(db.buffer) >= batchSize {
		select {
		case db.flushCh <- struct{}{}:
		default:
		}
	}
}

// flushLoop periodically flushes buffered metrics.
func (db *MetricsDB) flushLoop() {
	for {
		select {
		case <-db.flushTicker.C:
			db.flush()
		case <-db.flushCh:
			db.flush()
		case <-db.done:
			db.flush() // Final flush
			return
		}
	}
}

// flush writes buffered metrics to the database.
func (db *MetricsDB) flush() {
	db.mu.Lock()
	if len(db.buffer) == 0 {
		db.mu.Unlock()
		return
	}
	records := db.buffer
	db.buffer = make([]MetricRecord, 0, db.config.BatchSize)
	db.lastFlush = time.Now()
	db.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	batch := &pgx.Batch{}
	for _, record := range records {
		batch.Queue(`
			INSERT INTO metrics_snapshots (
				timestamp, granularity, requests, tokens, input_tokens, output_tokens,
				success_count, failure_count, avg_latency_ms
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id
		`, record.Timestamp, record.Granularity, record.Requests, record.Tokens,
			record.InputTokens, record.OutputTokens, record.SuccessCount,
			record.FailureCount, record.AvgLatencyMs)
	}

	results := db.pool.SendBatch(ctx, batch)
	defer results.Close()

	for i, record := range records {
		var snapshotID int64
		if err := results.QueryRow().Scan(&snapshotID); err != nil {
			log.WithError(err).Error("Failed to insert metrics snapshot")
			continue
		}

		// Insert model metrics
		if len(record.ModelMetrics) > 0 {
			modelBatch := &pgx.Batch{}
			for _, model := range record.ModelMetrics {
				modelBatch.Queue(`
					INSERT INTO model_metrics (
						snapshot_id, model_name, requests, tokens, input_tokens,
						output_tokens, avg_latency_ms
					) VALUES ($1, $2, $3, $4, $5, $6, $7)
				`, snapshotID, model.ModelName, model.Requests, model.Tokens,
					model.InputTokens, model.OutputTokens, model.AvgLatencyMs)
			}
			modelResults := db.pool.SendBatch(ctx, modelBatch)
			modelResults.Close()
		}

		// Update aggregates for minute/hour granularity
		if record.Granularity == "minute" || record.Granularity == "hour" {
			db.updateHourlyAggregate(ctx, record)
		}
		if record.Granularity == "hour" || record.Granularity == "day" {
			db.updateDailyAggregate(ctx, record)
		}

		_ = i // Suppress unused warning
	}
}

// updateHourlyAggregate upserts hourly aggregate data.
func (db *MetricsDB) updateHourlyAggregate(ctx context.Context, record MetricRecord) {
	hourStart := record.Timestamp.Truncate(time.Hour)

	_, err := db.pool.Exec(ctx, `
		INSERT INTO hourly_aggregates (
			hour_start, total_requests, total_tokens, total_input_tokens,
			total_output_tokens, success_count, failure_count, avg_latency_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (hour_start) DO UPDATE SET
			total_requests = hourly_aggregates.total_requests + EXCLUDED.total_requests,
			total_tokens = hourly_aggregates.total_tokens + EXCLUDED.total_tokens,
			total_input_tokens = hourly_aggregates.total_input_tokens + EXCLUDED.total_input_tokens,
			total_output_tokens = hourly_aggregates.total_output_tokens + EXCLUDED.total_output_tokens,
			success_count = hourly_aggregates.success_count + EXCLUDED.success_count,
			failure_count = hourly_aggregates.failure_count + EXCLUDED.failure_count,
			avg_latency_ms = (hourly_aggregates.avg_latency_ms * 0.9 + EXCLUDED.avg_latency_ms * 0.1),
			updated_at = NOW()
	`, hourStart, record.Requests, record.Tokens, record.InputTokens,
		record.OutputTokens, record.SuccessCount, record.FailureCount, record.AvgLatencyMs)

	if err != nil {
		log.WithError(err).Error("Failed to update hourly aggregate")
	}
}

// updateDailyAggregate upserts daily aggregate data.
func (db *MetricsDB) updateDailyAggregate(ctx context.Context, record MetricRecord) {
	date := record.Timestamp.Truncate(24 * time.Hour)

	_, err := db.pool.Exec(ctx, `
		INSERT INTO daily_aggregates (
			date, total_requests, total_tokens, total_input_tokens,
			total_output_tokens, success_count, failure_count, avg_latency_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (date) DO UPDATE SET
			total_requests = daily_aggregates.total_requests + EXCLUDED.total_requests,
			total_tokens = daily_aggregates.total_tokens + EXCLUDED.total_tokens,
			total_input_tokens = daily_aggregates.total_input_tokens + EXCLUDED.total_input_tokens,
			total_output_tokens = daily_aggregates.total_output_tokens + EXCLUDED.total_output_tokens,
			success_count = daily_aggregates.success_count + EXCLUDED.success_count,
			failure_count = daily_aggregates.failure_count + EXCLUDED.failure_count,
			avg_latency_ms = (daily_aggregates.avg_latency_ms * 0.9 + EXCLUDED.avg_latency_ms * 0.1),
			updated_at = NOW()
	`, date, record.Requests, record.Tokens, record.InputTokens,
		record.OutputTokens, record.SuccessCount, record.FailureCount, record.AvgLatencyMs)

	if err != nil {
		log.WithError(err).Error("Failed to update daily aggregate")
	}
}

// cleanupLoop periodically removes old data based on retention policy.
func (db *MetricsDB) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			db.cleanup()
		case <-db.done:
			return
		}
	}
}

// cleanup removes old metrics data.
func (db *MetricsDB) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	retentionDays := db.config.RetentionDays
	if retentionDays <= 0 {
		retentionDays = 30
	}

	// Delete second-granularity data older than 1 hour
	_, _ = db.pool.Exec(ctx, `
		DELETE FROM metrics_snapshots 
		WHERE granularity = 'second' 
		AND timestamp < NOW() - INTERVAL '1 hour'
	`)

	// Delete minute-granularity data older than 24 hours
	_, _ = db.pool.Exec(ctx, `
		DELETE FROM metrics_snapshots 
		WHERE granularity = 'minute' 
		AND timestamp < NOW() - INTERVAL '24 hours'
	`)

	// Delete hour-granularity data older than 7 days
	_, _ = db.pool.Exec(ctx, `
		DELETE FROM metrics_snapshots 
		WHERE granularity = 'hour' 
		AND timestamp < NOW() - INTERVAL '7 days'
	`)

	// Delete daily data older than retention period
	_, _ = db.pool.Exec(ctx, fmt.Sprintf(`
		DELETE FROM metrics_snapshots 
		WHERE granularity = 'day' 
		AND timestamp < NOW() - INTERVAL '%d days'
	`, retentionDays))

	_, _ = db.pool.Exec(ctx, fmt.Sprintf(`
		DELETE FROM hourly_aggregates 
		WHERE hour_start < NOW() - INTERVAL '%d days'
	`, retentionDays))

	_, _ = db.pool.Exec(ctx, fmt.Sprintf(`
		DELETE FROM daily_aggregates 
		WHERE date < CURRENT_DATE - INTERVAL '%d days'
	`, retentionDays))

	log.Debug("Metrics cleanup completed")
}

// GetTPSData retrieves TPS data from the database.
func (db *MetricsDB) GetTPSData(ctx context.Context, limit int) ([]MetricBucket, float64, error) {
	if db == nil || db.pool == nil {
		return nil, 0, fmt.Errorf("database not initialized")
	}

	rows, err := db.pool.Query(ctx, `
		SELECT timestamp, requests, tokens, avg_latency_ms, success_count, failure_count
		FROM metrics_snapshots
		WHERE granularity = 'second'
		ORDER BY timestamp DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var buckets []MetricBucket
	for rows.Next() {
		var b MetricBucket
		if err := rows.Scan(&b.Timestamp, &b.Requests, &b.Tokens, &b.AvgLatency,
			&b.SuccessCount, &b.FailureCount); err != nil {
			continue
		}
		b.ByModel = make(map[string]ModelBucket)
		buckets = append(buckets, b)
	}

	// Reverse to get chronological order
	for i, j := 0, len(buckets)-1; i < j; i, j = i+1, j-1 {
		buckets[i], buckets[j] = buckets[j], buckets[i]
	}

	// Calculate current TPS (average of last 10 seconds)
	var currentTPS float64
	if len(buckets) > 0 {
		count := 10
		if len(buckets) < count {
			count = len(buckets)
		}
		var total int64
		for i := len(buckets) - count; i < len(buckets); i++ {
			total += buckets[i].Requests
		}
		currentTPS = float64(total) / float64(count)
	}

	return buckets, currentTPS, nil
}

// GetTPMData retrieves TPM data from the database.
func (db *MetricsDB) GetTPMData(ctx context.Context, limit int) ([]MetricBucket, int64, error) {
	if db == nil || db.pool == nil {
		return nil, 0, fmt.Errorf("database not initialized")
	}

	rows, err := db.pool.Query(ctx, `
		SELECT timestamp, requests, tokens, input_tokens, output_tokens, 
			avg_latency_ms, success_count, failure_count
		FROM metrics_snapshots
		WHERE granularity = 'minute'
		ORDER BY timestamp DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var buckets []MetricBucket
	for rows.Next() {
		var b MetricBucket
		if err := rows.Scan(&b.Timestamp, &b.Requests, &b.Tokens, &b.InputTokens,
			&b.OutputTokens, &b.AvgLatency, &b.SuccessCount, &b.FailureCount); err != nil {
			continue
		}
		b.ByModel = make(map[string]ModelBucket)
		buckets = append(buckets, b)
	}

	// Reverse to get chronological order
	for i, j := 0, len(buckets)-1; i < j; i, j = i+1, j-1 {
		buckets[i], buckets[j] = buckets[j], buckets[i]
	}

	var currentTPM int64
	if len(buckets) > 0 {
		currentTPM = buckets[len(buckets)-1].Tokens
	}

	return buckets, currentTPM, nil
}

// GetTPHData retrieves TPH data from the database.
func (db *MetricsDB) GetTPHData(ctx context.Context, limit int) ([]MetricBucket, int64, error) {
	if db == nil || db.pool == nil {
		return nil, 0, fmt.Errorf("database not initialized")
	}

	rows, err := db.pool.Query(ctx, `
		SELECT hour_start, total_requests, total_tokens, total_input_tokens,
			total_output_tokens, avg_latency_ms, success_count, failure_count
		FROM hourly_aggregates
		ORDER BY hour_start DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var buckets []MetricBucket
	for rows.Next() {
		var b MetricBucket
		if err := rows.Scan(&b.Timestamp, &b.Requests, &b.Tokens, &b.InputTokens,
			&b.OutputTokens, &b.AvgLatency, &b.SuccessCount, &b.FailureCount); err != nil {
			continue
		}
		b.ByModel = make(map[string]ModelBucket)
		buckets = append(buckets, b)
	}

	// Reverse to get chronological order
	for i, j := 0, len(buckets)-1; i < j; i, j = i+1, j-1 {
		buckets[i], buckets[j] = buckets[j], buckets[i]
	}

	var currentTPH int64
	if len(buckets) > 0 {
		currentTPH = buckets[len(buckets)-1].Tokens
	}

	return buckets, currentTPH, nil
}

// GetTPDData retrieves TPD data from the database.
func (db *MetricsDB) GetTPDData(ctx context.Context, limit int) ([]MetricBucket, int64, error) {
	if db == nil || db.pool == nil {
		return nil, 0, fmt.Errorf("database not initialized")
	}

	rows, err := db.pool.Query(ctx, `
		SELECT date, total_requests, total_tokens, total_input_tokens,
			total_output_tokens, avg_latency_ms, success_count, failure_count
		FROM daily_aggregates
		ORDER BY date DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var buckets []MetricBucket
	for rows.Next() {
		var b MetricBucket
		if err := rows.Scan(&b.Timestamp, &b.Requests, &b.Tokens, &b.InputTokens,
			&b.OutputTokens, &b.AvgLatency, &b.SuccessCount, &b.FailureCount); err != nil {
			continue
		}
		b.ByModel = make(map[string]ModelBucket)
		buckets = append(buckets, b)
	}

	// Reverse to get chronological order
	for i, j := 0, len(buckets)-1; i < j; i, j = i+1, j-1 {
		buckets[i], buckets[j] = buckets[j], buckets[i]
	}

	var currentTPD int64
	if len(buckets) > 0 {
		currentTPD = buckets[len(buckets)-1].Tokens
	}

	return buckets, currentTPD, nil
}

// Close shuts down the database connection.
func (db *MetricsDB) Close() {
	if db == nil {
		return
	}

	db.closeOnce.Do(func() {
		close(db.done)
		if db.flushTicker != nil {
			db.flushTicker.Stop()
		}
		db.wg.Wait()
		if db.pool != nil {
			db.pool.Close()
		}
	})
}

// IsEnabled returns true if the metrics database is initialized.
func (db *MetricsDB) IsEnabled() bool {
	return db != nil && db.pool != nil
}
