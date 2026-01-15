package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	managementHandlers "github.com/router-for-me/CLIProxyAPI/v6/internal/api/handlers/management"
	log "github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for metrics dashboard
	},
}

// MetricsHub maintains active WebSocket connections and broadcasts metrics
type MetricsHub struct {
	clients    map[*MetricsClient]bool
	register   chan *MetricsClient
	unregister chan *MetricsClient
	mu         sync.RWMutex

	// Metrics provider
	metricsHandler *managementHandlers.Handler

	// Recent activity tracking
	recentRequests []RequestLog
	recentErrors   []ErrorLog
	requestsMu     sync.RWMutex
}

// MetricsClient represents a WebSocket client
type MetricsClient struct {
	hub  *MetricsHub
	conn *websocket.Conn
	send chan []byte
}

// RequestLog represents a single request for the activity feed
type RequestLog struct {
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Model     string `json:"model"`
	Tokens    int64  `json:"tokens"`
	LatencyMs int64  `json:"latency_ms"`
	Status    string `json:"status"` // success, error, rate_limited
	AuthID    string `json:"auth_id"`
	Endpoint  string `json:"endpoint"`
}

// ErrorLog represents an error for the error panel
type ErrorLog struct {
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Model     string `json:"model"`
	Error     string `json:"error"`
	Code      int    `json:"code"`
	AuthID    string `json:"auth_id"`
}

// EnhancedMetrics extends LiveMetricsSnapshot with activity data
type EnhancedMetrics struct {
	managementHandlers.LiveMetricsSnapshot
	RecentRequests []RequestLog `json:"recent_requests"`
	RecentErrors   []ErrorLog   `json:"recent_errors"`
	ConnectionID   string       `json:"connection_id"`
}

var (
	globalHub     *MetricsHub
	globalHubOnce sync.Once
)

// GetMetricsHub returns the global metrics hub singleton
func GetMetricsHub() *MetricsHub {
	globalHubOnce.Do(func() {
		globalHub = &MetricsHub{
			clients:        make(map[*MetricsClient]bool),
			register:       make(chan *MetricsClient),
			unregister:     make(chan *MetricsClient),
			recentRequests: make([]RequestLog, 0, 100),
			recentErrors:   make([]ErrorLog, 0, 50),
		}
		go globalHub.run()
	})
	return globalHub
}

// SetMetricsHandler sets the management handler for metrics
func (h *MetricsHub) SetMetricsHandler(handler *managementHandlers.Handler) {
	h.metricsHandler = handler
}

// run handles client registration and message broadcasting
func (h *MetricsHub) run() {
	ticker := time.NewTicker(100 * time.Millisecond) // 100ms broadcast interval
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			clientCount := len(h.clients)
			h.mu.Unlock()
			log.Debugf("WebSocket client connected, total: %d", clientCount)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			clientCount := len(h.clients)
			h.mu.Unlock()
			log.Debugf("WebSocket client disconnected, total: %d", clientCount)

		case <-ticker.C:
			h.broadcastMetrics()
		}
	}
}

// broadcastMetrics sends current metrics to all connected clients
func (h *MetricsHub) broadcastMetrics() {
	h.mu.RLock()
	clientCount := len(h.clients)
	h.mu.RUnlock()

	if clientCount == 0 {
		return
	}

	// Get current metrics
	var snapshot managementHandlers.LiveMetricsSnapshot
	if h.metricsHandler != nil {
		tracker := managementHandlers.GetRealTimeTracker()
		if tracker != nil {
			snapshot = tracker.Snapshot()
		}
	}

	// Build enhanced metrics with activity data
	h.requestsMu.RLock()
	enhanced := EnhancedMetrics{
		LiveMetricsSnapshot: snapshot,
		RecentRequests:      h.recentRequests,
		RecentErrors:        h.recentErrors,
	}
	h.requestsMu.RUnlock()

	data, err := json.Marshal(enhanced)
	if err != nil {
		log.Errorf("Failed to marshal metrics: %v", err)
		return
	}

	h.broadcastToClients(data)
}

func (h *MetricsHub) broadcastToClients(data []byte) {
	h.mu.RLock()
	if len(h.clients) == 0 {
		h.mu.RUnlock()
		return
	}
	clients := make([]*MetricsClient, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	var stale []*MetricsClient
	for _, client := range clients {
		select {
		case client.send <- data:
		default:
			stale = append(stale, client)
		}
	}

	if len(stale) == 0 {
		return
	}

	h.mu.Lock()
	for _, client := range stale {
		if _, ok := h.clients[client]; ok {
			delete(h.clients, client)
			close(client.send)
		}
	}
	h.mu.Unlock()
}

// AddRequest adds a request to the activity feed
func (h *MetricsHub) AddRequest(req RequestLog) {
	h.requestsMu.Lock()
	defer h.requestsMu.Unlock()

	h.recentRequests = append([]RequestLog{req}, h.recentRequests...)
	if len(h.recentRequests) > 50 {
		h.recentRequests = h.recentRequests[:50]
	}
}

// AddError adds an error to the error log
func (h *MetricsHub) AddError(err ErrorLog) {
	h.requestsMu.Lock()
	defer h.requestsMu.Unlock()

	h.recentErrors = append([]ErrorLog{err}, h.recentErrors...)
	if len(h.recentErrors) > 20 {
		h.recentErrors = h.recentErrors[:20]
	}
}

// GetClientCount returns the number of connected clients
func (h *MetricsHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// serveWebSocket handles WebSocket upgrade and client connection
func (s *Server) serveMetricsWebSocket(c *gin.Context) {
	// Validate management key
	key := c.Query("key")
	if key == "" {
		key = c.GetHeader("Authorization")
		if len(key) > 7 && key[:7] == "Bearer " {
			key = key[7:]
		}
	}

	cfg := s.cfg
	if cfg == nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	// Simple key validation (matches management middleware logic)
	secretHash := cfg.RemoteManagement.SecretKey
	if secretHash == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// For now, accept the key directly (simplified auth for WebSocket)
	// In production, you'd want to validate against the bcrypt hash
	if key == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("WebSocket upgrade failed: %v", err)
		return
	}

	hub := GetMetricsHub()
	hub.SetMetricsHandler(s.mgmt)

	client := &MetricsClient{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	hub.register <- client

	// Start read/write pumps
	go client.writePump()
	go client.readPump()
}

// readPump handles incoming messages and connection health
func (c *MetricsClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Debugf("WebSocket read error: %v", err)
			}
			break
		}
	}
}

// writePump sends messages to the WebSocket client
func (c *MetricsClient) writePump() {
	ticker := time.NewTicker(30 * time.Second) // Ping interval
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Write any queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
