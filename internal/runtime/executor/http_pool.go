// Package executor provides runtime execution capabilities for various AI service providers.
// This file implements HTTP/2 connection pooling for high-performance API requests.
package executor

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
)

// HTTPPoolConfig holds configuration for HTTP connection pooling.
type HTTPPoolConfig struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
	IdleConnTimeout     time.Duration
	TLSHandshakeTimeout time.Duration
	ForceHTTP2          bool
	DisableCompression  bool
}

// DefaultHTTPPoolConfig returns optimized defaults for AI API providers.
func DefaultHTTPPoolConfig() HTTPPoolConfig {
	return HTTPPoolConfig{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ForceHTTP2:          true,
		DisableCompression:  false,
	}
}

// HTTPPool manages a pool of reusable HTTP transports for different providers.
type HTTPPool struct {
	mu         sync.RWMutex
	transports map[string]*http.Transport
	config     HTTPPoolConfig
}

var (
	globalHTTPPool     *HTTPPool
	globalHTTPPoolOnce sync.Once
)

// GetHTTPPool returns the global HTTP connection pool singleton.
func GetHTTPPool() *HTTPPool {
	globalHTTPPoolOnce.Do(func() {
		globalHTTPPool = NewHTTPPool(DefaultHTTPPoolConfig())
	})
	return globalHTTPPool
}

// NewHTTPPool creates a new HTTP connection pool with the given configuration.
func NewHTTPPool(cfg HTTPPoolConfig) *HTTPPool {
	return &HTTPPool{
		transports: make(map[string]*http.Transport),
		config:     cfg,
	}
}

// Configure updates the pool configuration. Should be called at startup.
func (p *HTTPPool) Configure(cfg HTTPPoolConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = cfg
	// Clear existing transports to apply new config
	for _, t := range p.transports {
		t.CloseIdleConnections()
	}
	p.transports = make(map[string]*http.Transport)
}

// GetTransport returns a shared transport for the given provider key.
// The key should identify the provider (e.g., "openai", "claude", "gemini").
func (p *HTTPPool) GetTransport(providerKey string) *http.Transport {
	p.mu.RLock()
	if t, ok := p.transports[providerKey]; ok {
		p.mu.RUnlock()
		return t
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if t, ok := p.transports[providerKey]; ok {
		return t
	}

	t := p.createTransport(nil)
	p.transports[providerKey] = t
	log.Debugf("created new HTTP/2 transport pool for provider: %s", providerKey)
	return t
}

// GetProxyTransport returns a transport configured with proxy settings.
func (p *HTTPPool) GetProxyTransport(providerKey, proxyURL string) *http.Transport {
	cacheKey := providerKey + "|" + proxyURL
	
	p.mu.RLock()
	if t, ok := p.transports[cacheKey]; ok {
		p.mu.RUnlock()
		return t
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if t, ok := p.transports[cacheKey]; ok {
		return t
	}

	t := p.createProxyTransport(proxyURL)
	if t != nil {
		p.transports[cacheKey] = t
		log.Debugf("created new HTTP/2 proxy transport pool for provider: %s", providerKey)
	}
	return t
}

// createTransport creates a new optimized HTTP transport.
func (p *HTTPPool) createTransport(proxyFunc func(*http.Request) (*url.URL, error)) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	t := &http.Transport{
		Proxy:                 proxyFunc,
		DialContext:           dialer.DialContext,
		MaxIdleConns:          p.config.MaxIdleConns,
		MaxIdleConnsPerHost:   p.config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       p.config.MaxConnsPerHost,
		IdleConnTimeout:       p.config.IdleConnTimeout,
		TLSHandshakeTimeout:   p.config.TLSHandshakeTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     p.config.ForceHTTP2,
		DisableCompression:    p.config.DisableCompression,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	return t
}

// createProxyTransport creates a transport with proxy configuration.
func (p *HTTPPool) createProxyTransport(proxyURL string) *http.Transport {
	if proxyURL == "" {
		return p.createTransport(nil)
	}

	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		log.Errorf("failed to parse proxy URL: %v", err)
		return nil
	}

	if parsedURL.Scheme == "socks5" {
		return p.createSOCKS5Transport(parsedURL)
	}

	if parsedURL.Scheme == "http" || parsedURL.Scheme == "https" {
		return p.createTransport(http.ProxyURL(parsedURL))
	}

	log.Errorf("unsupported proxy scheme: %s", parsedURL.Scheme)
	return nil
}

// createSOCKS5Transport creates a transport for SOCKS5 proxy.
func (p *HTTPPool) createSOCKS5Transport(parsedURL *url.URL) *http.Transport {
	var proxyAuth *proxy.Auth
	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		password, _ := parsedURL.User.Password()
		proxyAuth = &proxy.Auth{User: username, Password: password}
	}

	dialer, err := proxy.SOCKS5("tcp", parsedURL.Host, proxyAuth, proxy.Direct)
	if err != nil {
		log.Errorf("failed to create SOCKS5 dialer: %v", err)
		return nil
	}

	t := &http.Transport{
		DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
		MaxIdleConns:          p.config.MaxIdleConns,
		MaxIdleConnsPerHost:   p.config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       p.config.MaxConnsPerHost,
		IdleConnTimeout:       p.config.IdleConnTimeout,
		TLSHandshakeTimeout:   p.config.TLSHandshakeTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     p.config.ForceHTTP2,
		DisableCompression:    p.config.DisableCompression,
	}

	return t
}

// GetClient returns an HTTP client using the pooled transport for the given provider.
func (p *HTTPPool) GetClient(providerKey string, timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: p.GetTransport(providerKey),
		Timeout:   timeout,
	}
}

// GetProxyClient returns an HTTP client with proxy using the pooled transport.
func (p *HTTPPool) GetProxyClient(providerKey, proxyURL string, timeout time.Duration) *http.Client {
	t := p.GetProxyTransport(providerKey, proxyURL)
	if t == nil {
		return &http.Client{Timeout: timeout}
	}
	return &http.Client{
		Transport: t,
		Timeout:   timeout,
	}
}

// CloseIdleConnections closes all idle connections in the pool.
func (p *HTTPPool) CloseIdleConnections() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, t := range p.transports {
		t.CloseIdleConnections()
	}
}

// Stats returns connection pool statistics.
type PoolStats struct {
	ProviderCount int
	Providers     []string
}

// GetStats returns current pool statistics.
func (p *HTTPPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	providers := make([]string, 0, len(p.transports))
	for k := range p.transports {
		providers = append(providers, k)
	}

	return PoolStats{
		ProviderCount: len(p.transports),
		Providers:     providers,
	}
}


