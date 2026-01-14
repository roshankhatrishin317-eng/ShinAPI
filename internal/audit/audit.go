// Package audit provides audit logging functionality for the CLI Proxy API.
// It captures and stores request/response metadata for debugging and compliance.
package audit

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// LogLevel indicates the severity/detail level of an audit entry.
type LogLevel string

const (
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelDebug   LogLevel = "debug"
)

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	ID           string            `json:"id"`
	Timestamp    time.Time         `json:"timestamp"`
	Level        LogLevel          `json:"level"`
	Provider     string            `json:"provider"`
	Model        string            `json:"model"`
	AuthID       string            `json:"auth_id,omitempty"`
	AuthLabel    string            `json:"auth_label,omitempty"`
	Endpoint     string            `json:"endpoint"`
	Method       string            `json:"method"`
	StatusCode   int               `json:"status_code"`
	Latency      time.Duration     `json:"latency_ms"`
	InputTokens  int64             `json:"input_tokens,omitempty"`
	OutputTokens int64             `json:"output_tokens,omitempty"`
	Error        string            `json:"error,omitempty"`
	ClientIP     string            `json:"client_ip,omitempty"`
	UserAgent    string            `json:"user_agent,omitempty"`
	RequestID    string            `json:"request_id,omitempty"`
	Streaming    bool              `json:"streaming"`
	Cached       bool              `json:"cached"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// AuditConfig configures audit logging behavior.
type AuditConfig struct {
	Enabled        bool `yaml:"enabled" json:"enabled"`
	MaxEntries     int  `yaml:"max-entries" json:"max_entries"`
	RetentionHours int  `yaml:"retention-hours" json:"retention_hours"`
	LogRequests    bool `yaml:"log-requests" json:"log_requests"`
	LogResponses   bool `yaml:"log-responses" json:"log_responses"`
	LogErrors      bool `yaml:"log-errors" json:"log_errors"`
	LogHeaders     bool `yaml:"log-headers" json:"log_headers"`
}

// DefaultAuditConfig returns sensible defaults.
func DefaultAuditConfig() AuditConfig {
	return AuditConfig{
		Enabled:        true,
		MaxEntries:     10000,
		RetentionHours: 24,
		LogRequests:    true,
		LogResponses:   true,
		LogErrors:      true,
		LogHeaders:     false,
	}
}

// AuditLogger manages audit log entries.
type AuditLogger struct {
	mu      sync.RWMutex
	entries []AuditEntry
	config  AuditConfig
	idGen   uint64
}

var (
	globalAuditLogger     *AuditLogger
	globalAuditLoggerOnce sync.Once
)

// GetAuditLogger returns the global audit logger singleton.
func GetAuditLogger() *AuditLogger {
	globalAuditLoggerOnce.Do(func() {
		globalAuditLogger = NewAuditLogger(DefaultAuditConfig())
	})
	return globalAuditLogger
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(cfg AuditConfig) *AuditLogger {
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = 10000
	}
	al := &AuditLogger{
		entries: make([]AuditEntry, 0, cfg.MaxEntries),
		config:  cfg,
	}
	go al.cleanupLoop()
	return al
}

// Configure updates the audit logger configuration.
func (al *AuditLogger) Configure(cfg AuditConfig) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.config = cfg
}

// IsEnabled returns whether audit logging is enabled.
func (al *AuditLogger) IsEnabled() bool {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return al.config.Enabled
}

// Log adds a new audit entry.
func (al *AuditLogger) Log(entry AuditEntry) {
	if !al.IsEnabled() {
		return
	}

	al.mu.Lock()
	defer al.mu.Unlock()

	// Generate ID
	al.idGen++
	entry.ID = generateAuditID(al.idGen, entry.Timestamp)

	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Set default level
	if entry.Level == "" {
		if entry.Error != "" {
			entry.Level = LogLevelError
		} else if entry.StatusCode >= 400 {
			entry.Level = LogLevelWarning
		} else {
			entry.Level = LogLevelInfo
		}
	}

	// Trim to max size
	if len(al.entries) >= al.config.MaxEntries {
		// Remove oldest 10%
		removeCount := al.config.MaxEntries / 10
		if removeCount < 1 {
			removeCount = 1
		}
		al.entries = al.entries[removeCount:]
	}

	al.entries = append(al.entries, entry)
}

// LogRequest logs an API request.
func (al *AuditLogger) LogRequest(req *http.Request, provider, model, authID, authLabel string) {
	if !al.IsEnabled() || !al.config.LogRequests {
		return
	}

	entry := AuditEntry{
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Provider:  provider,
		Model:     model,
		AuthID:    authID,
		AuthLabel: authLabel,
		Endpoint:  req.URL.Path,
		Method:    req.Method,
		ClientIP:  getClientIP(req),
		UserAgent: req.UserAgent(),
		RequestID: req.Header.Get("X-Request-ID"),
	}

	al.Log(entry)
}

// LogResponse logs an API response.
func (al *AuditLogger) LogResponse(
	provider, model, authID, authLabel, endpoint, method string,
	statusCode int, latency time.Duration, inputTokens, outputTokens int64,
	streaming, cached bool, err error,
) {
	if !al.IsEnabled() {
		return
	}

	if statusCode >= 400 && !al.config.LogErrors {
		return
	}
	if statusCode < 400 && !al.config.LogResponses {
		return
	}

	entry := AuditEntry{
		Timestamp:    time.Now(),
		Provider:     provider,
		Model:        model,
		AuthID:       authID,
		AuthLabel:    authLabel,
		Endpoint:     endpoint,
		Method:       method,
		StatusCode:   statusCode,
		Latency:      latency,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Streaming:    streaming,
		Cached:       cached,
	}

	if err != nil {
		entry.Error = err.Error()
		entry.Level = LogLevelError
	}

	al.Log(entry)
}

// GetEntries returns audit entries with optional filtering.
func (al *AuditLogger) GetEntries(filter AuditFilter) []AuditEntry {
	al.mu.RLock()
	defer al.mu.RUnlock()

	result := make([]AuditEntry, 0)

	for i := len(al.entries) - 1; i >= 0; i-- {
		entry := al.entries[i]

		// Apply filters
		if filter.Level != "" && entry.Level != filter.Level {
			continue
		}
		if filter.Provider != "" && entry.Provider != filter.Provider {
			continue
		}
		if filter.Model != "" && entry.Model != filter.Model {
			continue
		}
		if filter.AuthID != "" && entry.AuthID != filter.AuthID {
			continue
		}
		if !filter.Since.IsZero() && entry.Timestamp.Before(filter.Since) {
			continue
		}
		if !filter.Until.IsZero() && entry.Timestamp.After(filter.Until) {
			continue
		}
		if filter.ErrorsOnly && entry.Error == "" {
			continue
		}
		if filter.MinLatencyMs > 0 && entry.Latency.Milliseconds() < filter.MinLatencyMs {
			continue
		}

		result = append(result, entry)

		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}

	return result
}

// GetStats returns aggregate statistics.
func (al *AuditLogger) GetStats() AuditStats {
	al.mu.RLock()
	defer al.mu.RUnlock()

	stats := AuditStats{
		TotalEntries:   len(al.entries),
		ProviderCounts: make(map[string]int),
		ModelCounts:    make(map[string]int),
		StatusCounts:   make(map[int]int),
		LevelCounts:    make(map[LogLevel]int),
	}

	var totalLatency time.Duration
	for _, entry := range al.entries {
		stats.ProviderCounts[entry.Provider]++
		stats.ModelCounts[entry.Model]++
		stats.StatusCounts[entry.StatusCode]++
		stats.LevelCounts[entry.Level]++
		stats.TotalTokens += entry.InputTokens + entry.OutputTokens

		if entry.Error != "" {
			stats.ErrorCount++
		}
		totalLatency += entry.Latency
	}

	if len(al.entries) > 0 {
		stats.AvgLatencyMs = totalLatency.Milliseconds() / int64(len(al.entries))
		stats.OldestEntry = al.entries[0].Timestamp
		stats.NewestEntry = al.entries[len(al.entries)-1].Timestamp
	}

	return stats
}

// Clear removes all audit entries.
func (al *AuditLogger) Clear() {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.entries = make([]AuditEntry, 0, al.config.MaxEntries)
}

// cleanupLoop periodically removes old entries.
func (al *AuditLogger) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		al.cleanup()
	}
}

func (al *AuditLogger) cleanup() {
	al.mu.Lock()
	defer al.mu.Unlock()

	if al.config.RetentionHours <= 0 {
		return
	}

	cutoff := time.Now().Add(-time.Duration(al.config.RetentionHours) * time.Hour)
	newEntries := make([]AuditEntry, 0, len(al.entries))

	for _, entry := range al.entries {
		if entry.Timestamp.After(cutoff) {
			newEntries = append(newEntries, entry)
		}
	}

	al.entries = newEntries
}

// AuditFilter specifies filtering criteria for audit entries.
type AuditFilter struct {
	Level        LogLevel  `json:"level,omitempty"`
	Provider     string    `json:"provider,omitempty"`
	Model        string    `json:"model,omitempty"`
	AuthID       string    `json:"auth_id,omitempty"`
	Since        time.Time `json:"since,omitempty"`
	Until        time.Time `json:"until,omitempty"`
	ErrorsOnly   bool      `json:"errors_only,omitempty"`
	MinLatencyMs int64     `json:"min_latency_ms,omitempty"`
	Limit        int       `json:"limit,omitempty"`
}

// AuditStats contains aggregate audit statistics.
type AuditStats struct {
	TotalEntries   int              `json:"total_entries"`
	ErrorCount     int              `json:"error_count"`
	TotalTokens    int64            `json:"total_tokens"`
	AvgLatencyMs   int64            `json:"avg_latency_ms"`
	OldestEntry    time.Time        `json:"oldest_entry,omitempty"`
	NewestEntry    time.Time        `json:"newest_entry,omitempty"`
	ProviderCounts map[string]int   `json:"provider_counts"`
	ModelCounts    map[string]int   `json:"model_counts"`
	StatusCounts   map[int]int      `json:"status_counts"`
	LevelCounts    map[LogLevel]int `json:"level_counts"`
}

// Export exports audit entries as JSON.
func (al *AuditLogger) Export() ([]byte, error) {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return json.Marshal(al.entries)
}

// Helper functions

func generateAuditID(seq uint64, t time.Time) string {
	return time.Now().Format("20060102150405") + "-" + uintToBase36(seq)
}

func uintToBase36(n uint64) string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	if n == 0 {
		return "0"
	}
	result := make([]byte, 0, 8)
	for n > 0 {
		result = append(result, chars[n%36])
		n /= 36
	}
	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}

func getClientIP(req *http.Request) string {
	if ip := req.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := req.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return req.RemoteAddr
}
