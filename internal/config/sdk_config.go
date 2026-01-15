// Package config provides configuration management for the CLI Proxy API server.
// It handles loading and parsing YAML configuration files, and provides structured
// access to application settings including server port, authentication directory,
// debug settings, proxy configuration, and API keys.
package config

// SDKConfig represents the application's configuration, loaded from a YAML file.
type SDKConfig struct {
	// ProxyURL is the URL of an optional proxy server to use for outbound requests.
	ProxyURL string `yaml:"proxy-url" json:"proxy-url"`

	// ForceModelPrefix requires explicit model prefixes (e.g., "teamA/gemini-3-pro-preview")
	// to target prefixed credentials. When false, unprefixed model requests may use prefixed
	// credentials as well.
	ForceModelPrefix bool `yaml:"force-model-prefix" json:"force-model-prefix"`

	// RequestLog enables or disables detailed request logging functionality.
	RequestLog bool `yaml:"request-log" json:"request-log"`

	// APIKeys is a list of keys for authenticating clients to this proxy server.
	APIKeys []string `yaml:"api-keys" json:"api-keys"`

	// Access holds request authentication provider configuration.
	Access AccessConfig `yaml:"auth,omitempty" json:"auth,omitempty"`

	// Streaming configures server-side streaming behavior (keep-alives and safe bootstrap retries).
	Streaming StreamingConfig `yaml:"streaming" json:"streaming"`

	// NonStreamKeepAliveInterval controls how often blank lines are emitted for non-streaming responses.
	// <= 0 disables keep-alives. Value is in seconds.
	NonStreamKeepAliveInterval int `yaml:"nonstream-keepalive-interval,omitempty" json:"nonstream-keepalive-interval,omitempty"`

	// Cache configures response caching behavior.
	Cache CacheConfig `yaml:"cache,omitempty" json:"cache,omitempty"`

	// Scheduler configures fair scheduling and weighted queuing.
	Scheduler SchedulerConfig `yaml:"scheduler,omitempty" json:"scheduler,omitempty"`

	// Redis configures Redis-backed caching.
	Redis RedisCacheConfig `yaml:"redis,omitempty" json:"redis,omitempty"`

	// Observability configures metrics and tracing.
	Observability ObservabilityConfig `yaml:"observability,omitempty" json:"observability,omitempty"`

	// Performance configures HTTP connection pooling and streaming optimization.
	Performance PerformanceConfig `yaml:"performance,omitempty" json:"performance,omitempty"`

	// MetricsDB configures PostgreSQL database for metrics persistence.
	MetricsDB MetricsDBConfig `yaml:"metrics-db,omitempty" json:"metrics-db,omitempty"`

	// Tools configures tool calling format conversion.
	Tools ToolsConfig `yaml:"tools,omitempty" json:"tools,omitempty"`

	// Reasoning configures extended thinking/reasoning support.
	Reasoning ReasoningConfig `yaml:"reasoning,omitempty" json:"reasoning,omitempty"`

	// Agent configures agentic loop orchestration.
	Agent AgentConfig `yaml:"agent,omitempty" json:"agent,omitempty"`

	// Context configures context window management.
	Context ContextConfig `yaml:"context,omitempty" json:"context,omitempty"`

	// Retry configures retry behavior with exponential backoff.
	Retry RetryConfig `yaml:"retry,omitempty" json:"retry,omitempty"`
}

// CacheConfig holds response caching configuration.
type CacheConfig struct {
	// Enabled controls whether response caching is enabled.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// DefaultTTLSeconds is the default TTL for cached responses.
	DefaultTTLSeconds int `yaml:"default-ttl-seconds" json:"default_ttl_seconds"`

	// MaxEntries is the maximum number of cached responses.
	MaxEntries int `yaml:"max-entries" json:"max_entries"`

	// SemanticCache configures semantic (similarity-based) caching.
	SemanticCache SemanticCacheConfig `yaml:"semantic,omitempty" json:"semantic,omitempty"`

	// StreamingCache configures streaming response caching.
	StreamingCache StreamingCacheConfig `yaml:"streaming,omitempty" json:"streaming,omitempty"`

	// CacheKey configures how cache keys are generated.
	CacheKey CacheKeyConfig `yaml:"cache-key,omitempty" json:"cache_key,omitempty"`

	// ModelConfigs holds per-model cache configuration overrides.
	ModelConfigs []ModelCacheConfigEntry `yaml:"models,omitempty" json:"models,omitempty"`
}

// SemanticCacheConfig configures semantic caching behavior.
type SemanticCacheConfig struct {
	// Enabled controls whether semantic caching is enabled.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// SimilarityThreshold is the minimum Jaccard similarity (0.0-1.0) for a cache hit.
	SimilarityThreshold float64 `yaml:"similarity-threshold" json:"similarity_threshold"`

	// NGramSize is the size of n-grams for similarity calculation.
	NGramSize int `yaml:"ngram-size" json:"ngram_size"`

	// NormalizeCase lowercases text for comparison.
	NormalizeCase bool `yaml:"normalize-case" json:"normalize_case"`

	// NormalizeWhitespace collapses whitespace for comparison.
	NormalizeWhitespace bool `yaml:"normalize-whitespace" json:"normalize_whitespace"`
}

// StreamingCacheConfig configures streaming response caching.
type StreamingCacheConfig struct {
	// Enabled controls whether streaming response caching is enabled.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// MaxEntries is the maximum number of cached streaming responses.
	MaxEntries int `yaml:"max-entries" json:"max_entries"`

	// MaxEventSizeBytes is the maximum size of a single event in bytes.
	MaxEventSizeBytes int64 `yaml:"max-event-size-bytes" json:"max_event_size_bytes"`

	// MaxTotalSizeBytes is the maximum total size of events per response.
	MaxTotalSizeBytes int64 `yaml:"max-total-size-bytes" json:"max_total_size_bytes"`

	// PreserveTimings preserves original timing between events on replay.
	PreserveTimings bool `yaml:"preserve-timings" json:"preserve_timings"`
}

// CacheKeyConfig configures how cache keys are generated.
type CacheKeyConfig struct {
	// IncludeModel includes model name in cache key.
	IncludeModel bool `yaml:"include-model" json:"include_model"`

	// IncludeSystemPrompt includes system prompt in cache key.
	IncludeSystemPrompt bool `yaml:"include-system-prompt" json:"include_system_prompt"`

	// IncludeTemperature includes temperature in cache key.
	IncludeTemperature bool `yaml:"include-temperature" json:"include_temperature"`

	// IncludeMaxTokens includes max_tokens in cache key.
	IncludeMaxTokens bool `yaml:"include-max-tokens" json:"include_max_tokens"`

	// IncludeTools includes tools/functions in cache key.
	IncludeTools bool `yaml:"include-tools" json:"include_tools"`

	// ExcludeFields lists field names to exclude from cache key.
	ExcludeFields []string `yaml:"exclude-fields" json:"exclude_fields"`
}

// ModelCacheConfigEntry holds per-model cache configuration.
type ModelCacheConfigEntry struct {
	// Model is the model name or pattern (supports * and ? wildcards).
	Model string `yaml:"model" json:"model"`

	// TTLSeconds is the cache TTL in seconds for this model.
	TTLSeconds int `yaml:"ttl-seconds" json:"ttl_seconds"`

	// Enabled controls whether caching is enabled for this model.
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`

	// MaxEntries overrides the max cache entries for this model.
	MaxEntries int `yaml:"max-entries,omitempty" json:"max_entries,omitempty"`

	// SimilarityThreshold overrides semantic cache threshold for this model.
	SimilarityThreshold float64 `yaml:"similarity-threshold,omitempty" json:"similarity_threshold,omitempty"`
}

// SchedulerConfig holds fair scheduling configuration.
type SchedulerConfig struct {
	// Enabled controls whether fair scheduling is enabled.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// DefaultWeight is the default weight for API keys without explicit config.
	DefaultWeight int `yaml:"default-weight" json:"default_weight"`

	// MaxQueueSize is the maximum number of pending requests per queue.
	MaxQueueSize int `yaml:"max-queue-size" json:"max_queue_size"`

	// MaxConcurrent is the maximum number of concurrent requests.
	MaxConcurrent int `yaml:"max-concurrent" json:"max_concurrent"`

	// QueueTimeoutSeconds is the maximum time a request can wait in queue.
	QueueTimeoutSeconds int `yaml:"queue-timeout-seconds" json:"queue_timeout_seconds"`

	// APIKeyWeights maps API keys to their scheduling weights.
	APIKeyWeights []APIKeyWeight `yaml:"api-key-weights,omitempty" json:"api_key_weights,omitempty"`
}

// RedisCacheConfig holds Redis cache configuration.
type RedisCacheConfig struct {
	// Enabled controls whether Redis caching is active.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Address is the Redis server address (host:port).
	Address string `yaml:"address" json:"address"`

	// Password is the Redis password (optional).
	Password string `yaml:"password" json:"password"`

	// Database is the Redis database number.
	Database int `yaml:"database" json:"database"`

	// KeyPrefix is prepended to all cache keys.
	KeyPrefix string `yaml:"key-prefix" json:"key_prefix"`

	// DefaultTTLSeconds is the default TTL for cached items.
	DefaultTTLSeconds int `yaml:"default-ttl-seconds" json:"default_ttl_seconds"`

	// MaxRetries is the maximum number of retries for failed operations.
	MaxRetries int `yaml:"max-retries" json:"max_retries"`

	// PoolSize is the maximum number of connections.
	PoolSize int `yaml:"pool-size" json:"pool_size"`

	// DialTimeoutMs is the timeout for establishing new connections.
	DialTimeoutMs int `yaml:"dial-timeout-ms" json:"dial_timeout_ms"`

	// ReadTimeoutMs is the timeout for read operations.
	ReadTimeoutMs int `yaml:"read-timeout-ms" json:"read_timeout_ms"`

	// WriteTimeoutMs is the timeout for write operations.
	WriteTimeoutMs int `yaml:"write-timeout-ms" json:"write_timeout_ms"`

	// EnableTLS enables TLS for Redis connections.
	EnableTLS bool `yaml:"enable-tls" json:"enable_tls"`
}

// ObservabilityConfig holds observability configuration.
type ObservabilityConfig struct {
	// Metrics configures Prometheus metrics.
	Metrics MetricsConfig `yaml:"metrics" json:"metrics"`

	// Tracing configures OpenTelemetry tracing.
	Tracing TracingConfig `yaml:"tracing" json:"tracing"`
}

// MetricsConfig configures Prometheus metrics.
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

	// UseOfficialClient enables the official prometheus/client_golang library
	// instead of the custom implementation. When enabled, the /metrics endpoint
	// uses promhttp.Handler() for standard Prometheus scraping. Default: false.
	UseOfficialClient bool `yaml:"use-official-client" json:"use_official_client"`
}

// TracingConfig configures OpenTelemetry tracing.
type TracingConfig struct {
	// Enabled controls whether tracing is active.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// ServiceName is the name of this service in traces.
	ServiceName string `yaml:"service-name" json:"service_name"`

	// ServiceVersion is the version of this service.
	ServiceVersion string `yaml:"service-version" json:"service_version"`

	// SamplingRate is the fraction of traces to sample (0.0-1.0).
	SamplingRate float64 `yaml:"sampling-rate" json:"sampling_rate"`

	// ExporterType is the trace exporter type (otlp, jaeger, zipkin, stdout, none).
	ExporterType string `yaml:"exporter-type" json:"exporter_type"`

	// ExporterEndpoint is the endpoint for the trace exporter.
	ExporterEndpoint string `yaml:"exporter-endpoint" json:"exporter_endpoint"`

	// Headers are additional headers for the exporter.
	Headers map[string]string `yaml:"headers" json:"headers"`

	// Insecure disables TLS for the exporter connection.
	Insecure bool `yaml:"insecure" json:"insecure"`
}

// APIKeyWeight defines the scheduling weight for an API key.
type APIKeyWeight struct {
	// APIKey is the API key pattern (supports * and ? wildcards).
	APIKey string `yaml:"api-key" json:"api_key"`

	// Weight is the scheduling weight (higher = more bandwidth).
	Weight int `yaml:"weight" json:"weight"`
}

// StreamingConfig holds server streaming behavior configuration.
type StreamingConfig struct {
	// KeepAliveSeconds controls how often the server emits SSE heartbeats (": keep-alive\n\n").
	// <= 0 disables keep-alives. Default is 0.
	KeepAliveSeconds int `yaml:"keepalive-seconds,omitempty" json:"keepalive-seconds,omitempty"`

	// BootstrapRetries controls how many times the server may retry a streaming request before any bytes are sent,
	// to allow auth rotation / transient recovery.
	// <= 0 disables bootstrap retries. Default is 0.
	BootstrapRetries int `yaml:"bootstrap-retries,omitempty" json:"bootstrap-retries,omitempty"`
}

// AccessConfig groups request authentication providers.
type AccessConfig struct {
	// Providers lists configured authentication providers.
	Providers []AccessProvider `yaml:"providers,omitempty" json:"providers,omitempty"`
}

// AccessProvider describes a request authentication provider entry.
type AccessProvider struct {
	// Name is the instance identifier for the provider.
	Name string `yaml:"name" json:"name"`

	// Type selects the provider implementation registered via the SDK.
	Type string `yaml:"type" json:"type"`

	// SDK optionally names a third-party SDK module providing this provider.
	SDK string `yaml:"sdk,omitempty" json:"sdk,omitempty"`

	// APIKeys lists inline keys for providers that require them.
	APIKeys []string `yaml:"api-keys,omitempty" json:"api-keys,omitempty"`

	// Config passes provider-specific options to the implementation.
	Config map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
}

const (
	// AccessProviderTypeConfigAPIKey is the built-in provider validating inline API keys.
	AccessProviderTypeConfigAPIKey = "config-api-key"

	// DefaultAccessProviderName is applied when no provider name is supplied.
	DefaultAccessProviderName = "config-inline"
)

// ConfigAPIKeyProvider returns the first inline API key provider if present.
func (c *SDKConfig) ConfigAPIKeyProvider() *AccessProvider {
	if c == nil {
		return nil
	}
	for i := range c.Access.Providers {
		if c.Access.Providers[i].Type == AccessProviderTypeConfigAPIKey {
			if c.Access.Providers[i].Name == "" {
				c.Access.Providers[i].Name = DefaultAccessProviderName
			}
			return &c.Access.Providers[i]
		}
	}
	return nil
}

// MakeInlineAPIKeyProvider constructs an inline API key provider configuration.
// It returns nil when no keys are supplied.
func MakeInlineAPIKeyProvider(keys []string) *AccessProvider {
	if len(keys) == 0 {
		return nil
	}
	provider := &AccessProvider{
		Name:    DefaultAccessProviderName,
		Type:    AccessProviderTypeConfigAPIKey,
		APIKeys: append([]string(nil), keys...),
	}
	return provider
}

// PerformanceConfig holds HTTP connection pooling and streaming optimization settings.
type PerformanceConfig struct {
	// HTTPPool configures HTTP/2 connection pooling.
	HTTPPool HTTPPoolConfig `yaml:"http-pool,omitempty" json:"http_pool,omitempty"`

	// StreamFanout configures SSE stream fan-out for parallel streaming.
	StreamFanout StreamFanoutConfig `yaml:"stream-fanout,omitempty" json:"stream_fanout,omitempty"`
}

// HTTPPoolConfig configures HTTP/2 connection pooling behavior.
type HTTPPoolConfig struct {
	// MaxIdleConns is the maximum number of idle connections across all hosts.
	MaxIdleConns int `yaml:"max-idle-conns" json:"max_idle_conns"`

	// MaxIdleConnsPerHost is the maximum idle connections per host.
	MaxIdleConnsPerHost int `yaml:"max-idle-conns-per-host" json:"max_idle_conns_per_host"`

	// MaxConnsPerHost is the maximum total connections per host.
	MaxConnsPerHost int `yaml:"max-conns-per-host" json:"max_conns_per_host"`

	// IdleConnTimeoutSeconds is how long idle connections stay open.
	IdleConnTimeoutSeconds int `yaml:"idle-conn-timeout-seconds" json:"idle_conn_timeout_seconds"`

	// ForceHTTP2 enables HTTP/2 for all connections.
	ForceHTTP2 bool `yaml:"force-http2" json:"force_http2"`
}

// StreamFanoutConfig configures SSE stream fan-out behavior.
type StreamFanoutConfig struct {
	// Enabled controls whether stream fan-out is active.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// BufferSize is the number of events to buffer for late joiners.
	BufferSize int `yaml:"buffer-size" json:"buffer_size"`

	// DedupWindowSeconds is the time window for detecting duplicate requests.
	DedupWindowSeconds int `yaml:"dedup-window-seconds" json:"dedup_window_seconds"`
}

// DefaultPerformanceConfig returns sensible defaults for performance settings.
func DefaultPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		HTTPPool: HTTPPoolConfig{
			MaxIdleConns:           100,
			MaxIdleConnsPerHost:    10,
			MaxConnsPerHost:        100,
			IdleConnTimeoutSeconds: 90,
			ForceHTTP2:             true,
		},
		StreamFanout: StreamFanoutConfig{
			Enabled:            true,
			BufferSize:         50,
			DedupWindowSeconds: 5,
		},
	}
}

// MetricsDBConfig configures PostgreSQL database for metrics persistence.
type MetricsDBConfig struct {
	// Enabled controls whether metrics are persisted to database.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// DSN is the PostgreSQL connection string.
	// Example: postgres://user:password@host:5432/database?sslmode=require
	DSN string `yaml:"dsn" json:"dsn"`

	// MaxConnections is the maximum number of database connections.
	MaxConnections int `yaml:"max-connections" json:"max_connections"`

	// RetentionDays is how many days of metrics to keep.
	RetentionDays int `yaml:"retention-days" json:"retention_days"`

	// FlushIntervalSeconds is how often to flush buffered metrics to the database.
	FlushIntervalSeconds int `yaml:"flush-interval-seconds" json:"flush_interval_seconds"`

	// BatchSize is the number of metrics to batch before flushing.
	BatchSize int `yaml:"batch-size" json:"batch_size"`
}

// ToolsConfig configures tool calling format conversion.
type ToolsConfig struct {
	// Enabled controls whether tool calling features are active.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// FormatConversion enables automatic tool format conversion between providers.
	FormatConversion bool `yaml:"format-conversion" json:"format_conversion"`

	// StreamingAccumulation enables streaming tool call accumulation.
	StreamingAccumulation bool `yaml:"streaming-accumulation" json:"streaming_accumulation"`

	// MaxToolCallsPerIteration limits tool calls per agentic iteration.
	MaxToolCallsPerIteration int `yaml:"max-tool-calls-per-iteration" json:"max_tool_calls_per_iteration"`
}

// ReasoningConfig configures extended thinking/reasoning support.
type ReasoningConfig struct {
	// Enabled controls whether reasoning features are active.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// ExtractThinking controls whether to extract thinking from responses.
	ExtractThinking bool `yaml:"extract-thinking" json:"extract_thinking"`

	// ShowThinkingToClient controls whether thinking is returned to client.
	ShowThinkingToClient bool `yaml:"show-thinking-to-client" json:"show_thinking_to_client"`

	// Claude holds Claude-specific reasoning settings.
	Claude ClaudeReasoningConfig `yaml:"claude,omitempty" json:"claude,omitempty"`

	// Gemini holds Gemini-specific reasoning settings.
	Gemini GeminiReasoningConfig `yaml:"gemini,omitempty" json:"gemini,omitempty"`

	// DeepSeek holds DeepSeek-specific reasoning settings.
	DeepSeek DeepSeekReasoningConfig `yaml:"deepseek,omitempty" json:"deepseek,omitempty"`
}

// ClaudeReasoningConfig holds Claude-specific reasoning settings.
type ClaudeReasoningConfig struct {
	// DefaultEffort is the default effort level (low, medium, high).
	DefaultEffort string `yaml:"default-effort" json:"default_effort"`

	// EnableThinking enables extended thinking mode.
	EnableThinking bool `yaml:"enable-thinking" json:"enable_thinking"`

	// BudgetTokens is the default thinking budget in tokens.
	BudgetTokens int `yaml:"budget-tokens" json:"budget_tokens"`

	// InterleavedTools enables tool use with thinking (requires beta header).
	InterleavedTools bool `yaml:"interleaved-tools" json:"interleaved_tools"`
}

// GeminiReasoningConfig holds Gemini-specific reasoning settings.
type GeminiReasoningConfig struct {
	// DefaultThinkingLevel is the default thinking level (low, high).
	DefaultThinkingLevel string `yaml:"default-thinking-level" json:"default_thinking_level"`

	// IncludeThoughts includes thought summaries in response.
	IncludeThoughts bool `yaml:"include-thoughts" json:"include_thoughts"`

	// PreserveSignatures preserves thought signatures for multi-turn.
	PreserveSignatures bool `yaml:"preserve-signatures" json:"preserve_signatures"`

	// ForceTemperature1 forces temperature to 1.0 for Gemini 3.
	ForceTemperature1 bool `yaml:"force-temperature-1" json:"force_temperature_1"`
}

// DeepSeekReasoningConfig holds DeepSeek-specific reasoning settings.
type DeepSeekReasoningConfig struct {
	// ExtractThinkTags extracts <think>...</think> tags.
	ExtractThinkTags bool `yaml:"extract-think-tags" json:"extract_think_tags"`
}

// AgentConfig configures agentic loop orchestration.
type AgentConfig struct {
	// Enabled controls whether agentic features are active.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// MaxIterations is the maximum agent loop iterations.
	MaxIterations int `yaml:"max-iterations" json:"max_iterations"`

	// ParallelToolCalls enables parallel tool execution.
	ParallelToolCalls bool `yaml:"parallel-tool-calls" json:"parallel_tool_calls"`

	// MaxConcurrency limits concurrent tool executions.
	MaxConcurrency int `yaml:"max-concurrency" json:"max_concurrency"`

	// ToolTimeoutMs is the timeout for tool execution in milliseconds.
	ToolTimeoutMs int `yaml:"tool-timeout-ms" json:"tool_timeout_ms"`

	// AutoExecuteTools executes tools automatically on the server.
	AutoExecuteTools bool `yaml:"auto-execute-tools" json:"auto_execute_tools"`
}

// ContextConfig configures context window management.
type ContextConfig struct {
	// Enabled controls whether context management is active.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Strategy is the truncation strategy (sliding-window, priority, summarize).
	Strategy string `yaml:"strategy" json:"strategy"`

	// ModelLimits maps model names to their context limits.
	ModelLimits map[string]int64 `yaml:"model-limits,omitempty" json:"model_limits,omitempty"`

	// AlwaysKeep defines what should never be truncated.
	AlwaysKeep ContextAlwaysKeep `yaml:"always-keep,omitempty" json:"always_keep,omitempty"`
}

// ContextAlwaysKeep defines what should never be truncated.
type ContextAlwaysKeep struct {
	// SystemPrompt keeps the system prompt.
	SystemPrompt bool `yaml:"system-prompt" json:"system_prompt"`

	// ToolDefinitions keeps tool definitions.
	ToolDefinitions bool `yaml:"tool-definitions" json:"tool_definitions"`

	// RecentMessages keeps the N most recent messages.
	RecentMessages int `yaml:"recent-messages" json:"recent_messages"`
}

// RetryConfig configures retry behavior with exponential backoff.
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts.
	MaxAttempts int `yaml:"max-attempts" json:"max_attempts"`

	// InitialDelayMs is the initial delay between retries in milliseconds.
	InitialDelayMs int `yaml:"initial-delay-ms" json:"initial_delay_ms"`

	// MaxDelayMs is the maximum delay between retries in milliseconds.
	MaxDelayMs int `yaml:"max-delay-ms" json:"max_delay_ms"`

	// Multiplier is the backoff multiplier.
	Multiplier float64 `yaml:"multiplier" json:"multiplier"`

	// Jitter adds randomness to delay (0.0 to 1.0).
	Jitter float64 `yaml:"jitter" json:"jitter"`

	// RetryableStatusCodes lists HTTP status codes to retry.
	RetryableStatusCodes []int `yaml:"retryable-status-codes" json:"retryable_status_codes"`
}
