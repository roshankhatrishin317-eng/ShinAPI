// Package management provides HTTP handlers for the management API.
// This file implements audit log endpoints.
package management

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/audit"
)

// GetAuditLogs returns audit log entries with optional filtering.
func (h *Handler) GetAuditLogs(c *gin.Context) {
	logger := audit.GetAuditLogger()

	filter := audit.AuditFilter{}

	// Parse query parameters
	if level := c.Query("level"); level != "" {
		filter.Level = audit.LogLevel(level)
	}
	if provider := c.Query("provider"); provider != "" {
		filter.Provider = provider
	}
	if model := c.Query("model"); model != "" {
		filter.Model = model
	}
	if authID := c.Query("auth_id"); authID != "" {
		filter.AuthID = authID
	}
	if since := c.Query("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			filter.Since = t
		}
	}
	if until := c.Query("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			filter.Until = t
		}
	}
	if c.Query("errors_only") == "true" {
		filter.ErrorsOnly = true
	}
	if minLatency := c.Query("min_latency_ms"); minLatency != "" {
		if v, err := strconv.ParseInt(minLatency, 10, 64); err == nil {
			filter.MinLatencyMs = v
		}
	}
	if limit := c.Query("limit"); limit != "" {
		if v, err := strconv.Atoi(limit); err == nil {
			filter.Limit = v
		}
	}

	// Default limit
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}

	entries := logger.GetEntries(filter)

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"count":   len(entries),
		"filter":  filter,
	})
}

// GetAuditStats returns aggregate audit statistics.
func (h *Handler) GetAuditStats(c *gin.Context) {
	logger := audit.GetAuditLogger()
	stats := logger.GetStats()

	c.JSON(http.StatusOK, stats)
}

// ClearAuditLogs clears all audit log entries.
func (h *Handler) ClearAuditLogs(c *gin.Context) {
	logger := audit.GetAuditLogger()
	logger.Clear()

	c.JSON(http.StatusOK, gin.H{
		"message": "Audit logs cleared",
	})
}

// ExportAuditLogs exports audit logs as JSON.
func (h *Handler) ExportAuditLogs(c *gin.Context) {
	logger := audit.GetAuditLogger()

	data, err := logger.Export()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to export audit logs",
		})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=audit-logs.json")
	c.Data(http.StatusOK, "application/json", data)
}

// GetAuditConfig returns the current audit configuration.
func (h *Handler) GetAuditConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"enabled":         audit.GetAuditLogger().IsEnabled(),
		"max_entries":     10000,
		"retention_hours": 24,
	})
}
