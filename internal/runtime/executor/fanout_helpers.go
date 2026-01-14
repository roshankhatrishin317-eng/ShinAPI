// Package executor provides runtime execution capabilities for various AI service providers.
// This file provides integration helpers for stream fanout in executors.
package executor

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"

	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
	log "github.com/sirupsen/logrus"
)

// StreamFanoutResult contains the result of a fanout check.
type StreamFanoutResult struct {
	// IsNew indicates whether a new upstream connection should be created
	IsNew bool
	// Stream is the shared stream (nil if fanout is disabled)
	Stream *SharedStream
	// Subscriber is the channel to receive events (nil if fanout is disabled)
	Subscriber chan StreamEvent
	// Key is the stream key for later cleanup
	Key string
}

// CheckStreamFanout checks if a stream fanout is available for the given request.
// Returns a result indicating whether to create a new upstream or subscribe to existing.
func CheckStreamFanout(model string, payload []byte) StreamFanoutResult {
	fanout := GetStreamFanout()
	if !fanout.IsEnabled() {
		return StreamFanoutResult{IsNew: true}
	}

	key := generateStreamKey(model, payload)
	stream, isNew, sub := fanout.GetOrCreateStream(key)

	return StreamFanoutResult{
		IsNew:      isNew,
		Stream:     stream,
		Subscriber: sub,
		Key:        key,
	}
}

// generateStreamKey creates a unique key for request deduplication.
func generateStreamKey(model string, payload []byte) string {
	h := sha256.New()
	h.Write([]byte(model))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// PublishToFanout publishes stream chunks to all subscribers.
// Call this for each chunk received from the upstream.
func PublishToFanout(stream *SharedStream, data []byte) {
	if stream == nil {
		return
	}
	stream.PublishBytes(data)
}

// CompleteFanout marks the stream as complete.
// Call this when the upstream stream finishes.
func CompleteFanout(stream *SharedStream) {
	if stream == nil {
		return
	}
	stream.Complete()
}

// SubscribeToFanout converts a subscriber channel to a StreamChunk channel.
// This bridges the fanout system with the executor stream interface.
func SubscribeToFanout(ctx context.Context, sub chan StreamEvent, model string, transform func([]byte) []cliproxyexecutor.StreamChunk) <-chan cliproxyexecutor.StreamChunk {
	out := make(chan cliproxyexecutor.StreamChunk)

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-sub:
				if !ok {
					// Stream completed
					return
				}
				if transform != nil {
					chunks := transform(event.Data)
					for _, chunk := range chunks {
						select {
						case out <- chunk:
						case <-ctx.Done():
							return
						}
					}
				} else {
					select {
					case out <- cliproxyexecutor.StreamChunk{Payload: event.Data}:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return out
}

// FanoutStats returns current fanout statistics for monitoring.
func GetFanoutStats() FanoutStats {
	return GetStreamFanout().GetStats()
}

// StreamFanoutMiddleware provides a helper for integrating fanout into ExecuteStream methods.
// Usage:
//
//	result := executor.CheckStreamFanout(model, payload)
//	if !result.IsNew {
//	    // Subscribe to existing stream
//	    return executor.SubscribeToFanout(ctx, result.Subscriber, model, nil), nil
//	}
//	// Create new upstream and publish to fanout
//	defer executor.CompleteFanout(result.Stream)
//	... fetch from upstream and call PublishToFanout for each chunk ...
type StreamFanoutMiddleware struct {
	Result StreamFanoutResult
}

// NewStreamFanoutMiddleware creates a new middleware for the given request.
func NewStreamFanoutMiddleware(model string, payload []byte) *StreamFanoutMiddleware {
	return &StreamFanoutMiddleware{
		Result: CheckStreamFanout(model, payload),
	}
}

// ShouldCreateUpstream returns true if a new upstream connection should be created.
func (m *StreamFanoutMiddleware) ShouldCreateUpstream() bool {
	return m.Result.IsNew
}

// GetSubscriberStream returns a stream for subscribing to an existing fanout.
func (m *StreamFanoutMiddleware) GetSubscriberStream(ctx context.Context, transform func([]byte) []cliproxyexecutor.StreamChunk) <-chan cliproxyexecutor.StreamChunk {
	if m.Result.Subscriber == nil {
		return nil
	}
	return SubscribeToFanout(ctx, m.Result.Subscriber, "", transform)
}

// PublishChunk publishes a chunk to all fanout subscribers.
func (m *StreamFanoutMiddleware) PublishChunk(data []byte) {
	PublishToFanout(m.Result.Stream, data)
}

// Complete marks the fanout stream as complete.
func (m *StreamFanoutMiddleware) Complete() {
	CompleteFanout(m.Result.Stream)
}

// LogFanoutStats logs the current fanout statistics.
func LogFanoutStats() {
	stats := GetFanoutStats()
	if stats.ActiveStreams > 0 {
		log.Debugf("stream fanout stats: active_streams=%d, total_subscribers=%d",
			stats.ActiveStreams, stats.TotalSubscribers)
	}
}

// WrapWithFanout wraps an executor's streaming logic with fanout support.
// This is a high-level helper that handles the common fanout pattern.
func WrapWithFanout(
	ctx context.Context,
	model string,
	payload []byte,
	createUpstream func() (<-chan cliproxyexecutor.StreamChunk, error),
) (<-chan cliproxyexecutor.StreamChunk, error) {
	middleware := NewStreamFanoutMiddleware(model, payload)

	// If not new, subscribe to existing stream
	if !middleware.ShouldCreateUpstream() {
		log.Debugf("fanout: subscribing to existing stream for model %s", model)
		return middleware.GetSubscriberStream(ctx, func(data []byte) []cliproxyexecutor.StreamChunk {
			return []cliproxyexecutor.StreamChunk{{Payload: data}}
		}), nil
	}

	// Create new upstream
	upstreamChan, err := createUpstream()
	if err != nil {
		middleware.Complete()
		return nil, err
	}

	// Create output channel that also publishes to fanout
	out := make(chan cliproxyexecutor.StreamChunk)

	go func() {
		defer close(out)
		defer middleware.Complete()

		for chunk := range upstreamChan {
			// Publish to fanout subscribers
			if len(chunk.Payload) > 0 {
				middleware.PublishChunk(chunk.Payload)
			}

			// Forward to caller
			select {
			case out <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

// RequestHash generates a hash for request deduplication.
func RequestHash(model string, messages, params []byte) string {
	h := sha256.New()
	h.Write([]byte(model))
	if messages != nil {
		h.Write(messages)
	}
	if params != nil {
		h.Write(params)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// NormalizePayloadForDedup removes fields that shouldn't affect deduplication.
func NormalizePayloadForDedup(payload []byte) []byte {
	// Remove fields that shouldn't affect deduplication
	// like request IDs, timestamps, etc.
	// For now, we use the payload as-is since most fields are meaningful
	return bytes.TrimSpace(payload)
}
