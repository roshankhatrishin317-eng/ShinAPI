// Package cmd provides command-line interface functionality for the CLI Proxy API server.
// It includes authentication flows for various AI service providers, service startup,
// and other command-line operations.
package cmd

import (
	"context"
	"errors"
	"os/signal"
	"syscall"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/api"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/cache"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/logging"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/runtime/executor"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy"
	log "github.com/sirupsen/logrus"
)

// initCacheSystem initializes the cache system based on configuration.
func initCacheSystem(cfg *config.Config) *cache.CacheSystem {
	cacheConfig := cache.DefaultCacheSystemConfig()

	// Apply Redis config if enabled
	if cfg.Redis.Enabled {
		cacheConfig.RedisEnabled = true
		cacheConfig.RedisAddress = cfg.Redis.Address
		if cacheConfig.RedisAddress == "" {
			cacheConfig.RedisAddress = "localhost:6379"
		}
		cacheConfig.RedisPassword = cfg.Redis.Password
		cacheConfig.RedisDatabase = cfg.Redis.Database
		cacheConfig.RedisKeyPrefix = cfg.Redis.KeyPrefix
		if cacheConfig.RedisKeyPrefix == "" {
			cacheConfig.RedisKeyPrefix = "shinapi:"
		}
		if cfg.Redis.DefaultTTLSeconds > 0 {
			cacheConfig.RedisTTLSeconds = cfg.Redis.DefaultTTLSeconds
		}
		if cfg.Redis.PoolSize > 0 {
			cacheConfig.RedisPoolSize = cfg.Redis.PoolSize
		}
		if cfg.Redis.DialTimeoutMs > 0 {
			cacheConfig.RedisDialTimeoutMs = cfg.Redis.DialTimeoutMs
		}
		if cfg.Redis.ReadTimeoutMs > 0 {
			cacheConfig.RedisReadTimeoutMs = cfg.Redis.ReadTimeoutMs
		}
		if cfg.Redis.WriteTimeoutMs > 0 {
			cacheConfig.RedisWriteTimeoutMs = cfg.Redis.WriteTimeoutMs
		}
		cacheConfig.RedisEnableTLS = cfg.Redis.EnableTLS
		if cfg.Redis.MaxRetries > 0 {
			cacheConfig.RedisMaxRetries = cfg.Redis.MaxRetries
		}
	}

	// Apply cache config
	if cfg.Cache.Enabled {
		if cfg.Cache.MaxEntries > 0 {
			cacheConfig.LRUCapacity = cfg.Cache.MaxEntries
		}
		if cfg.Cache.DefaultTTLSeconds > 0 {
			cacheConfig.LRUTTLSeconds = cfg.Cache.DefaultTTLSeconds
		}

		// Semantic cache
		if cfg.Cache.SemanticCache.Enabled {
			cacheConfig.SemanticEnabled = true
			if cfg.Cache.SemanticCache.SimilarityThreshold > 0 {
				cacheConfig.SemanticSimilarityThreshold = cfg.Cache.SemanticCache.SimilarityThreshold
			}
		}

		// Streaming cache
		if cfg.Cache.StreamingCache.Enabled {
			cacheConfig.StreamingEnabled = true
			if cfg.Cache.StreamingCache.MaxEntries > 0 {
				cacheConfig.StreamingMaxEntries = cfg.Cache.StreamingCache.MaxEntries
			}
			if cfg.Cache.StreamingCache.MaxEventSizeBytes > 0 {
				cacheConfig.StreamingMaxEventSize = cfg.Cache.StreamingCache.MaxEventSizeBytes
			}
			if cfg.Cache.StreamingCache.MaxTotalSizeBytes > 0 {
				cacheConfig.StreamingMaxTotalSize = cfg.Cache.StreamingCache.MaxTotalSizeBytes
			}
			cacheConfig.StreamingPreserveTimings = cfg.Cache.StreamingCache.PreserveTimings
		}
	}

	return cache.InitCacheSystem(cacheConfig)
}

// initPerformanceSystem initializes HTTP connection pooling and stream fanout.
func initPerformanceSystem(cfg *config.Config) {
	// Configure HTTP connection pool
	httpPoolCfg := executor.DefaultHTTPPoolConfig()
	if cfg.Performance.HTTPPool.MaxIdleConns > 0 {
		httpPoolCfg.MaxIdleConns = cfg.Performance.HTTPPool.MaxIdleConns
	}
	if cfg.Performance.HTTPPool.MaxIdleConnsPerHost > 0 {
		httpPoolCfg.MaxIdleConnsPerHost = cfg.Performance.HTTPPool.MaxIdleConnsPerHost
	}
	if cfg.Performance.HTTPPool.MaxConnsPerHost > 0 {
		httpPoolCfg.MaxConnsPerHost = cfg.Performance.HTTPPool.MaxConnsPerHost
	}
	if cfg.Performance.HTTPPool.IdleConnTimeoutSeconds > 0 {
		httpPoolCfg.IdleConnTimeout = time.Duration(cfg.Performance.HTTPPool.IdleConnTimeoutSeconds) * time.Second
	}
	httpPoolCfg.ForceHTTP2 = cfg.Performance.HTTPPool.ForceHTTP2

	executor.GetHTTPPool().Configure(httpPoolCfg)
	log.Infof("HTTP/2 connection pool initialized: max_idle=%d, max_per_host=%d, idle_timeout=%v",
		httpPoolCfg.MaxIdleConns, httpPoolCfg.MaxConnsPerHost, httpPoolCfg.IdleConnTimeout)

	// Configure stream fanout
	fanoutCfg := executor.DefaultStreamFanoutConfig()
	fanoutCfg.Enabled = cfg.Performance.StreamFanout.Enabled
	if cfg.Performance.StreamFanout.BufferSize > 0 {
		fanoutCfg.BufferSize = cfg.Performance.StreamFanout.BufferSize
	}
	if cfg.Performance.StreamFanout.DedupWindowSeconds > 0 {
		fanoutCfg.DedupWindowSeconds = cfg.Performance.StreamFanout.DedupWindowSeconds
	}

	executor.GetStreamFanout().Configure(fanoutCfg)
	if fanoutCfg.Enabled {
		log.Infof("Stream fanout enabled: buffer_size=%d, dedup_window=%ds",
			fanoutCfg.BufferSize, fanoutCfg.DedupWindowSeconds)
	}
}

// StartService builds and runs the proxy service using the exported SDK.
// It creates a new proxy service instance, sets up signal handling for graceful shutdown,
// and starts the service with the provided configuration.
//
// Parameters:
//   - cfg: The application configuration
//   - configPath: The path to the configuration file
//   - localPassword: Optional password accepted for local management requests
func StartService(cfg *config.Config, configPath string, localPassword string) {
	// Initialize optional Zap logger if configured
	if cfg.UseZapLogger {
		if err := logging.InitZapLoggerSimple(cfg.Debug); err != nil {
			log.Warnf("failed to initialize zap logger: %v", err)
		} else {
			log.Info("Zap structured logger initialized (high-performance mode)")
			defer logging.ZapSync()
		}
	}

	// Initialize cache system (including Redis if configured)
	cacheSystem := initCacheSystem(cfg)
	defer func() {
		if err := cacheSystem.Close(); err != nil {
			log.Warnf("failed to close cache system: %v", err)
		}
	}()

	// Initialize metrics database if configured
	if cfg.MetricsDB.Enabled {
		if err := usage.InitMetricsDB(cfg.MetricsDB); err != nil {
			log.Warnf("failed to initialize metrics database: %v", err)
		} else {
			defer func() {
				if db := usage.GetMetricsDB(); db != nil {
					db.Close()
				}
			}()
		}
	}

	// Initialize performance optimizations (HTTP/2 pooling, stream fanout)
	initPerformanceSystem(cfg)
	defer executor.GetHTTPPool().CloseIdleConnections()

	builder := cliproxy.NewBuilder().
		WithConfig(cfg).
		WithConfigPath(configPath).
		WithLocalManagementPassword(localPassword)

	ctxSignal, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	runCtx := ctxSignal
	if localPassword != "" {
		var keepAliveCancel context.CancelFunc
		runCtx, keepAliveCancel = context.WithCancel(ctxSignal)
		builder = builder.WithServerOptions(api.WithKeepAliveEndpoint(10*time.Second, func() {
			log.Warn("keep-alive endpoint idle for 10s, shutting down")
			keepAliveCancel()
		}))
	}

	service, err := builder.Build()
	if err != nil {
		log.Errorf("failed to build proxy service: %v", err)
		return
	}

	err = service.Run(runCtx)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Errorf("proxy service exited with error: %v", err)
	}
}

// WaitForCloudDeploy waits indefinitely for shutdown signals in cloud deploy mode
// when no configuration file is available.
func WaitForCloudDeploy() {
	// Clarify that we are intentionally idle for configuration and not running the API server.
	log.Info("Cloud deploy mode: No config found; standing by for configuration. API server is not started. Press Ctrl+C to exit.")

	ctxSignal, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Block until shutdown signal is received
	<-ctxSignal.Done()
	log.Info("Cloud deploy mode: Shutdown signal received; exiting")
}
