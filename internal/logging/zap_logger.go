// Package logging provides logging utilities for the CLI Proxy API server.
// This file provides an optional high-performance Zap logger that can coexist
// with the existing logrus logger.
package logging

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	zapLogger  *zap.Logger
	zapSugar   *zap.SugaredLogger
	zapEnabled bool
	zapOnce    sync.Once
	zapMu      sync.RWMutex
)

// ZapConfig configures the Zap logger.
type ZapConfig struct {
	// Development enables development mode (more verbose, human-readable output).
	Development bool
	// Level sets the minimum log level.
	Level zapcore.Level
	// OutputPaths are the paths to write logs to (e.g., "stdout", "/var/log/app.log").
	OutputPaths []string
	// ErrorOutputPaths are the paths to write error logs to.
	ErrorOutputPaths []string
	// EnableCaller adds caller information to log entries.
	EnableCaller bool
	// EnableStacktrace adds stack trace on error logs.
	EnableStacktrace bool
}

// DefaultZapConfig returns sensible defaults for Zap logging.
func DefaultZapConfig(debug bool) ZapConfig {
	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}
	return ZapConfig{
		Development:      debug,
		Level:            level,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EnableCaller:     true,
		EnableStacktrace: !debug,
	}
}

// InitZapLogger initializes the Zap logger with the given configuration.
// This can be called multiple times safely; initialization happens only once.
// Returns nil if initialization succeeds, otherwise returns the error.
func InitZapLogger(cfg ZapConfig) error {
	var initErr error
	zapOnce.Do(func() {
		var zapCfg zap.Config

		if cfg.Development {
			zapCfg = zap.NewDevelopmentConfig()
			zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			zapCfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
		} else {
			zapCfg = zap.NewProductionConfig()
			zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		}

		zapCfg.Level = zap.NewAtomicLevelAt(cfg.Level)

		if len(cfg.OutputPaths) > 0 {
			zapCfg.OutputPaths = cfg.OutputPaths
		}
		if len(cfg.ErrorOutputPaths) > 0 {
			zapCfg.ErrorOutputPaths = cfg.ErrorOutputPaths
		}

		zapCfg.DisableCaller = !cfg.EnableCaller
		zapCfg.DisableStacktrace = !cfg.EnableStacktrace

		var err error
		zapLogger, err = zapCfg.Build()
		if err != nil {
			initErr = err
			return
		}

		zapSugar = zapLogger.Sugar()
		zapEnabled = true
	})
	return initErr
}

// InitZapLoggerSimple initializes Zap with simple debug flag.
func InitZapLoggerSimple(debug bool) error {
	return InitZapLogger(DefaultZapConfig(debug))
}

// ZapEnabled returns true if Zap logger has been initialized.
func ZapEnabled() bool {
	zapMu.RLock()
	defer zapMu.RUnlock()
	return zapEnabled
}

// Zap returns the Zap logger instance.
// Returns nil if Zap has not been initialized.
func Zap() *zap.Logger {
	zapMu.RLock()
	defer zapMu.RUnlock()
	if !zapEnabled {
		return nil
	}
	return zapLogger
}

// Sugar returns the Zap sugared logger instance.
// Returns nil if Zap has not been initialized.
func Sugar() *zap.SugaredLogger {
	zapMu.RLock()
	defer zapMu.RUnlock()
	if !zapEnabled {
		return nil
	}
	return zapSugar
}

// ZapSync flushes any buffered log entries.
// Should be called before program exit.
func ZapSync() error {
	zapMu.RLock()
	defer zapMu.RUnlock()
	if !zapEnabled || zapLogger == nil {
		return nil
	}
	return zapLogger.Sync()
}

// ZapField creates a zap.Field for structured logging.
// These are convenience wrappers around zap's field constructors.
func ZapString(key, val string) zap.Field {
	return zap.String(key, val)
}

func ZapInt(key string, val int) zap.Field {
	return zap.Int(key, val)
}

func ZapInt64(key string, val int64) zap.Field {
	return zap.Int64(key, val)
}

func ZapFloat64(key string, val float64) zap.Field {
	return zap.Float64(key, val)
}

func ZapBool(key string, val bool) zap.Field {
	return zap.Bool(key, val)
}

func ZapError(err error) zap.Field {
	return zap.Error(err)
}

func ZapAny(key string, val any) zap.Field {
	return zap.Any(key, val)
}

// ZapRequestID creates a request_id field for structured logging.
func ZapRequestID(id string) zap.Field {
	return zap.String("request_id", id)
}

// ZapModel creates a model field for structured logging.
func ZapModel(model string) zap.Field {
	return zap.String("model", model)
}

// ZapProvider creates a provider field for structured logging.
func ZapProvider(provider string) zap.Field {
	return zap.String("provider", provider)
}

// ZapDuration creates a duration_ms field for structured logging.
func ZapDurationMs(durationMs float64) zap.Field {
	return zap.Float64("duration_ms", durationMs)
}

// ZapTokens creates a tokens field for structured logging.
func ZapTokens(tokens int64) zap.Field {
	return zap.Int64("tokens", tokens)
}

// Example usage with existing logrus:
//
//   if logging.ZapEnabled() {
//       logging.Sugar().Infow("request completed",
//           "request_id", reqID,
//           "model", model,
//           "duration_ms", duration,
//       )
//   } else {
//       log.WithFields(log.Fields{
//           "request_id": reqID,
//           "model": model,
//       }).Info("request completed")
//   }

// ZapLogToFile configures Zap to also log to a file.
func ZapLogToFile(filePath string, debug bool) error {
	cfg := DefaultZapConfig(debug)
	cfg.OutputPaths = append(cfg.OutputPaths, filePath)
	return InitZapLogger(cfg)
}

// ZapWithRotation sets up Zap with file rotation using lumberjack.
// Note: For file rotation, use the existing logrus + lumberjack setup,
// or integrate go.uber.org/zap/zapcore with lumberjack.v2.
func ZapWithRotation(filePath string, maxSizeMB, maxBackups, maxAgeDays int, debug bool) error {
	// For now, just use the simple file path.
	// Full rotation support would require additional integration.
	return ZapLogToFile(filePath, debug)
}

func init() {
	// Check if ZAP_ENABLED environment variable is set
	if os.Getenv("ZAP_ENABLED") == "true" {
		debug := os.Getenv("DEBUG") == "true"
		_ = InitZapLoggerSimple(debug)
	}
}
