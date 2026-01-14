// Package observability provides metrics collection and tracing for the API proxy.
// This file implements OpenTelemetry tracing integration.
package observability

import (
	"context"
	"sync"
	"time"
)

// Span represents a trace span.
type Span interface {
	End()
	SetStatus(code SpanStatusCode, description string)
	SetAttribute(key string, value interface{})
	AddEvent(name string, attrs map[string]interface{})
	RecordError(err error)
	SpanContext() SpanContext
}

// SpanStatusCode represents the status of a span.
type SpanStatusCode int

const (
	SpanStatusUnset SpanStatusCode = iota
	SpanStatusOK
	SpanStatusError
)

// SpanContext contains identifying trace information.
type SpanContext struct {
	TraceID    string
	SpanID     string
	TraceFlags byte
	Remote     bool
}

// SpanKind is the role a span plays in a trace.
type SpanKind int

const (
	SpanKindInternal SpanKind = iota
	SpanKindServer
	SpanKindClient
	SpanKindProducer
	SpanKindConsumer
)

// Tracer provides methods to create spans.
type Tracer interface {
	Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span)
}

// SpanOption configures a span.
type SpanOption func(*spanConfig)

type spanConfig struct {
	kind       SpanKind
	attributes map[string]interface{}
	links      []SpanContext
}

// WithSpanKind sets the span kind.
func WithSpanKind(kind SpanKind) SpanOption {
	return func(cfg *spanConfig) {
		cfg.kind = kind
	}
}

// WithAttributes sets initial span attributes.
func WithAttributes(attrs map[string]interface{}) SpanOption {
	return func(cfg *spanConfig) {
		cfg.attributes = attrs
	}
}

// TracerConfig configures the tracer.
type TracerConfig struct {
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
	// BatchTimeout is the maximum delay before exporting spans.
	BatchTimeoutMs int `yaml:"batch-timeout-ms" json:"batch_timeout_ms"`
	// MaxExportBatchSize is the maximum number of spans per batch.
	MaxExportBatchSize int `yaml:"max-export-batch-size" json:"max_export_batch_size"`
}

// DefaultTracerConfig returns sensible defaults.
func DefaultTracerConfig() TracerConfig {
	return TracerConfig{
		Enabled:            false,
		ServiceName:        "shinapi",
		ServiceVersion:     "1.0.0",
		SamplingRate:       0.1, // Sample 10% of traces
		ExporterType:       "none",
		ExporterEndpoint:   "localhost:4317",
		Insecure:           true,
		BatchTimeoutMs:     5000,
		MaxExportBatchSize: 512,
	}
}

// NoopSpan is a span that does nothing.
type NoopSpan struct {
	ctx SpanContext
}

func (s *NoopSpan) End()                                               {}
func (s *NoopSpan) SetStatus(code SpanStatusCode, description string)  {}
func (s *NoopSpan) SetAttribute(key string, value interface{})         {}
func (s *NoopSpan) AddEvent(name string, attrs map[string]interface{}) {}
func (s *NoopSpan) RecordError(err error)                              {}
func (s *NoopSpan) SpanContext() SpanContext                           { return s.ctx }

// NoopTracer is a tracer that creates no-op spans.
type NoopTracer struct{}

func (t *NoopTracer) Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	return ctx, &NoopSpan{}
}

// InMemorySpan records span data in memory for testing/debugging.
type InMemorySpan struct {
	mu          sync.Mutex
	name        string
	kind        SpanKind
	startTime   time.Time
	endTime     time.Time
	status      SpanStatusCode
	statusDesc  string
	attributes  map[string]interface{}
	events      []spanEvent
	errors      []error
	ctx         SpanContext
	ended       bool
}

type spanEvent struct {
	name       string
	timestamp  time.Time
	attributes map[string]interface{}
}

func (s *InMemorySpan) End() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ended {
		s.endTime = time.Now()
		s.ended = true
	}
}

func (s *InMemorySpan) SetStatus(code SpanStatusCode, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = code
	s.statusDesc = description
}

func (s *InMemorySpan) SetAttribute(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.attributes == nil {
		s.attributes = make(map[string]interface{})
	}
	s.attributes[key] = value
}

func (s *InMemorySpan) AddEvent(name string, attrs map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, spanEvent{
		name:       name,
		timestamp:  time.Now(),
		attributes: attrs,
	})
}

func (s *InMemorySpan) RecordError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.errors = append(s.errors, err)
		s.status = SpanStatusError
	}
}

func (s *InMemorySpan) SpanContext() SpanContext {
	return s.ctx
}

// Duration returns the span duration.
func (s *InMemorySpan) Duration() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ended {
		return s.endTime.Sub(s.startTime)
	}
	return time.Since(s.startTime)
}

// InMemoryTracer creates spans stored in memory.
type InMemoryTracer struct {
	mu        sync.Mutex
	spans     []*InMemorySpan
	maxSpans  int
	idCounter uint64
}

// NewInMemoryTracer creates a new in-memory tracer.
func NewInMemoryTracer(maxSpans int) *InMemoryTracer {
	if maxSpans <= 0 {
		maxSpans = 1000
	}
	return &InMemoryTracer{
		spans:    make([]*InMemorySpan, 0, maxSpans),
		maxSpans: maxSpans,
	}
}

func (t *InMemoryTracer) Start(ctx context.Context, name string, opts ...SpanOption) (context.Context, Span) {
	cfg := &spanConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	t.mu.Lock()
	t.idCounter++
	spanID := t.idCounter
	
	span := &InMemorySpan{
		name:       name,
		kind:       cfg.kind,
		startTime:  time.Now(),
		attributes: cfg.attributes,
		ctx: SpanContext{
			TraceID: generateTraceID(),
			SpanID:  generateSpanID(spanID),
		},
	}
	
	// Evict old spans if at capacity
	if len(t.spans) >= t.maxSpans {
		t.spans = t.spans[1:]
	}
	t.spans = append(t.spans, span)
	t.mu.Unlock()

	return ctx, span
}

// Spans returns all recorded spans.
func (t *InMemoryTracer) Spans() []*InMemorySpan {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make([]*InMemorySpan, len(t.spans))
	copy(result, t.spans)
	return result
}

// Clear removes all recorded spans.
func (t *InMemoryTracer) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spans = t.spans[:0]
}

func generateTraceID() string {
	return time.Now().Format("20060102150405.000000")
}

func generateSpanID(id uint64) string {
	return time.Now().Format("150405") + "-" + string(rune('A'+id%26))
}

// TracingMiddleware provides request tracing.
type TracingMiddleware struct {
	tracer Tracer
	config TracerConfig
}

// NewTracingMiddleware creates new tracing middleware.
func NewTracingMiddleware(tracer Tracer, cfg TracerConfig) *TracingMiddleware {
	return &TracingMiddleware{
		tracer: tracer,
		config: cfg,
	}
}

// StartRequestSpan starts a span for an HTTP request.
func (m *TracingMiddleware) StartRequestSpan(ctx context.Context, method, path, model string) (context.Context, Span) {
	if m.tracer == nil {
		return ctx, &NoopSpan{}
	}

	return m.tracer.Start(ctx, "http.request",
		WithSpanKind(SpanKindServer),
		WithAttributes(map[string]interface{}{
			"http.method": method,
			"http.path":   path,
			"model":       model,
		}),
	)
}

// StartProviderSpan starts a span for a provider call.
func (m *TracingMiddleware) StartProviderSpan(ctx context.Context, provider, model string) (context.Context, Span) {
	if m.tracer == nil {
		return ctx, &NoopSpan{}
	}

	return m.tracer.Start(ctx, "provider.request",
		WithSpanKind(SpanKindClient),
		WithAttributes(map[string]interface{}{
			"provider": provider,
			"model":    model,
		}),
	)
}

// StartCacheSpan starts a span for a cache operation.
func (m *TracingMiddleware) StartCacheSpan(ctx context.Context, operation string) (context.Context, Span) {
	if m.tracer == nil {
		return ctx, &NoopSpan{}
	}

	return m.tracer.Start(ctx, "cache."+operation,
		WithSpanKind(SpanKindInternal),
	)
}

// Global tracer
var (
	globalTracer     Tracer
	globalTracerOnce sync.Once
	globalTracerMu   sync.RWMutex
)

// GetTracer returns the global tracer.
func GetTracer() Tracer {
	globalTracerMu.RLock()
	defer globalTracerMu.RUnlock()
	if globalTracer == nil {
		return &NoopTracer{}
	}
	return globalTracer
}

// SetTracer sets the global tracer.
func SetTracer(t Tracer) {
	globalTracerMu.Lock()
	defer globalTracerMu.Unlock()
	globalTracer = t
}

// InitTracer initializes the global tracer based on config.
func InitTracer(cfg TracerConfig) Tracer {
	globalTracerOnce.Do(func() {
		if !cfg.Enabled {
			globalTracer = &NoopTracer{}
			return
		}

		switch cfg.ExporterType {
		case "stdout", "memory":
			globalTracer = NewInMemoryTracer(1000)
		default:
			// For production, you would integrate with real OTEL SDK here
			globalTracer = &NoopTracer{}
		}
	})
	return globalTracer
}
