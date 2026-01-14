// Package executor provides runtime execution capabilities for various AI service providers.
// This file provides helper functions for proper HTTP connection management.
package executor

import (
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// DrainAndClose properly drains and closes an HTTP response body.
// This is critical for connection reuse - connections are only returned to the pool
// when the response body is fully read and closed.
func DrainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}

	// Drain the body to allow connection reuse
	_, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		log.Debugf("failed to drain response body: %v", err)
	}

	if err := resp.Body.Close(); err != nil {
		log.Debugf("failed to close response body: %v", err)
	}
}

// DrainBody drains the response body without closing it.
// Use this when you need to drain but close separately.
func DrainBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}

	_, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		log.Debugf("failed to drain response body: %v", err)
	}
}

// CloseBody closes the response body with error logging.
func CloseBody(resp *http.Response, context string) {
	if resp == nil || resp.Body == nil {
		return
	}

	if err := resp.Body.Close(); err != nil {
		log.Errorf("%s: close response body error: %v", context, err)
	}
}
