// Package executor provides runtime execution capabilities for various AI service providers.
// This file implements SSE stream fan-out for efficient parallel streaming to multiple clients.
package executor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// StreamFanout manages shared upstream connections for SSE streaming.
// It allows multiple clients to subscribe to the same upstream stream,
// reducing API calls and improving efficiency.
type StreamFanout struct {
	mu      sync.RWMutex
	streams map[string]*SharedStream
	config  StreamFanoutConfig
}

// StreamFanoutConfig configures fan-out behavior.
type StreamFanoutConfig struct {
	Enabled            bool
	BufferSize         int
	DedupWindowSeconds int
}

// DefaultStreamFanoutConfig returns sensible defaults.
func DefaultStreamFanoutConfig() StreamFanoutConfig {
	return StreamFanoutConfig{
		Enabled:            true,
		BufferSize:         50,
		DedupWindowSeconds: 5,
	}
}

// SharedStream represents a single upstream connection shared by multiple subscribers.
type SharedStream struct {
	key         string
	mu          sync.RWMutex
	subscribers map[chan StreamEvent]struct{}
	events      []StreamEvent
	bufferSize  int
	done        chan struct{}
	completed   bool
	createdAt   time.Time
	lastEventAt time.Time
}

// StreamEvent represents a single SSE event in the stream.
type StreamEvent struct {
	Data      []byte
	EventType string
	ID        string
	Timestamp time.Time
}

var (
	globalStreamFanout     *StreamFanout
	globalStreamFanoutOnce sync.Once
)

// GetStreamFanout returns the global stream fanout singleton.
func GetStreamFanout() *StreamFanout {
	globalStreamFanoutOnce.Do(func() {
		globalStreamFanout = NewStreamFanout(DefaultStreamFanoutConfig())
	})
	return globalStreamFanout
}

// NewStreamFanout creates a new stream fanout manager.
func NewStreamFanout(cfg StreamFanoutConfig) *StreamFanout {
	sf := &StreamFanout{
		streams: make(map[string]*SharedStream),
		config:  cfg,
	}
	go sf.cleanupLoop()
	return sf
}

// Configure updates the fanout configuration.
func (sf *StreamFanout) Configure(cfg StreamFanoutConfig) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	sf.config = cfg
}

// IsEnabled returns whether fan-out is enabled.
func (sf *StreamFanout) IsEnabled() bool {
	sf.mu.RLock()
	defer sf.mu.RUnlock()
	return sf.config.Enabled
}

// GenerateStreamKey creates a unique key for a request based on its content.
func GenerateStreamKey(model string, messages []byte, params []byte) string {
	h := sha256.New()
	h.Write([]byte(model))
	h.Write(messages)
	h.Write(params)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// GetOrCreateStream returns an existing stream or creates a new one.
// Returns the stream, a boolean indicating if it's new, and a subscriber channel.
func (sf *StreamFanout) GetOrCreateStream(key string) (*SharedStream, bool, chan StreamEvent) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if !sf.config.Enabled {
		return nil, false, nil
	}

	stream, exists := sf.streams[key]
	if exists && !stream.IsCompleted() {
		// Subscribe to existing stream
		sub := stream.Subscribe()
		log.Debugf("stream fanout: subscribed to existing stream %s, total subscribers: %d", key, stream.SubscriberCount())
		return stream, false, sub
	}

	// Create new stream
	stream = &SharedStream{
		key:         key,
		subscribers: make(map[chan StreamEvent]struct{}),
		events:      make([]StreamEvent, 0, sf.config.BufferSize),
		bufferSize:  sf.config.BufferSize,
		done:        make(chan struct{}),
		createdAt:   time.Now(),
		lastEventAt: time.Now(),
	}
	sf.streams[key] = stream

	sub := stream.Subscribe()
	log.Debugf("stream fanout: created new stream %s", key)
	return stream, true, sub
}

// RemoveStream removes a stream from the manager.
func (sf *StreamFanout) RemoveStream(key string) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	delete(sf.streams, key)
}

// cleanupLoop periodically removes completed or stale streams.
func (sf *StreamFanout) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		sf.cleanup()
	}
}

func (sf *StreamFanout) cleanup() {
	// Copy stream references to avoid nested locking
	sf.mu.Lock()
	toCheck := make(map[string]*SharedStream, len(sf.streams))
	for k, v := range sf.streams {
		toCheck[k] = v
	}
	sf.mu.Unlock()

	now := time.Now()
	dedupWindow := time.Duration(sf.config.DedupWindowSeconds) * time.Second
	if dedupWindow <= 0 {
		dedupWindow = 5 * time.Second
	}

	var toDelete []string
	for key, stream := range toCheck {
		stream.mu.RLock()
		isStale := stream.completed && now.Sub(stream.lastEventAt) > dedupWindow
		noSubscribers := len(stream.subscribers) == 0 && now.Sub(stream.createdAt) > dedupWindow
		stream.mu.RUnlock()

		if isStale || noSubscribers {
			toDelete = append(toDelete, key)
		}
	}

	if len(toDelete) > 0 {
		sf.mu.Lock()
		for _, key := range toDelete {
			delete(sf.streams, key)
			log.Debugf("stream fanout: cleaned up stream %s", key)
		}
		sf.mu.Unlock()
	}
}

// Stats returns current fanout statistics.
type FanoutStats struct {
	ActiveStreams   int
	TotalSubscribers int
}

// GetStats returns current fanout statistics.
func (sf *StreamFanout) GetStats() FanoutStats {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	stats := FanoutStats{
		ActiveStreams: len(sf.streams),
	}

	for _, stream := range sf.streams {
		stats.TotalSubscribers += stream.SubscriberCount()
	}

	return stats
}

// Subscribe adds a new subscriber to the stream and returns a channel for events.
func (s *SharedStream) Subscribe() chan StreamEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan StreamEvent, 100)
	s.subscribers[ch] = struct{}{}

	// Replay buffered events to late joiner
	for _, event := range s.events {
		select {
		case ch <- event:
		default:
			// Channel full, skip old events
		}
	}

	return ch
}

// Unsubscribe removes a subscriber from the stream.
func (s *SharedStream) Unsubscribe(ch chan StreamEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.subscribers[ch]; ok {
		delete(s.subscribers, ch)
		close(ch)
	}
}

// Publish sends an event to all subscribers and buffers it for late joiners.
func (s *SharedStream) Publish(event StreamEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.completed {
		return
	}

	event.Timestamp = time.Now()
	s.lastEventAt = event.Timestamp

	// Buffer the event for late joiners
	if len(s.events) >= s.bufferSize {
		// Remove oldest event
		s.events = s.events[1:]
	}
	s.events = append(s.events, event)

	// Broadcast to all subscribers
	for ch := range s.subscribers {
		select {
		case ch <- event:
		default:
			// Subscriber channel full, they're too slow
			log.Debugf("stream fanout: dropping event for slow subscriber on stream %s", s.key)
		}
	}
}

// PublishBytes is a convenience method to publish raw bytes as a data event.
func (s *SharedStream) PublishBytes(data []byte) {
	s.Publish(StreamEvent{
		Data:      data,
		EventType: "message",
	})
}

// Complete marks the stream as completed and notifies all subscribers.
func (s *SharedStream) Complete() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.completed {
		return
	}

	s.completed = true
	close(s.done)

	// Close all subscriber channels
	for ch := range s.subscribers {
		close(ch)
	}
	s.subscribers = make(map[chan StreamEvent]struct{})

	log.Debugf("stream fanout: completed stream %s", s.key)
}

// IsCompleted returns whether the stream has finished.
func (s *SharedStream) IsCompleted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.completed
}

// Done returns a channel that's closed when the stream completes.
func (s *SharedStream) Done() <-chan struct{} {
	return s.done
}

// SubscriberCount returns the number of active subscribers.
func (s *SharedStream) SubscriberCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subscribers)
}

// WaitForEvents waits for events from the subscriber channel with context support.
func WaitForEvents(ctx context.Context, sub chan StreamEvent) ([]StreamEvent, error) {
	var events []StreamEvent

	for {
		select {
		case <-ctx.Done():
			return events, ctx.Err()
		case event, ok := <-sub:
			if !ok {
				// Channel closed, stream completed
				return events, nil
			}
			events = append(events, event)
		}
	}
}
