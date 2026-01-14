package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dashboard/*
var dashboardFS embed.FS

// serveDashboard serves the Next.js dashboard static files
func (s *Server) serveDashboard(c *gin.Context) {
	// Get the path after /dashboard
	path := c.Param("filepath")
	if path == "" || path == "/" {
		path = "/index.html"
	}

	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// Get the embedded filesystem rooted at dashboard
	subFS, err := fs.Sub(dashboardFS, "dashboard")
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Try to read the file
	content, err := fs.ReadFile(subFS, path)
	if err != nil {
		// Try with trailing index.html for directories
		if !strings.HasSuffix(path, ".html") {
			indexPath := strings.TrimSuffix(path, "/") + "/index.html"
			content, err = fs.ReadFile(subFS, indexPath)
			if err != nil {
				// Fallback to index.html for SPA routing
				content, err = fs.ReadFile(subFS, "index.html")
				if err != nil {
					c.AbortWithStatus(http.StatusNotFound)
					return
				}
			}
			path = indexPath
		} else {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
	}

	// Set content type based on file extension
	contentType := "text/html"
	if strings.HasSuffix(path, ".js") {
		contentType = "application/javascript"
	} else if strings.HasSuffix(path, ".css") {
		contentType = "text/css"
	} else if strings.HasSuffix(path, ".json") {
		contentType = "application/json"
	} else if strings.HasSuffix(path, ".svg") {
		contentType = "image/svg+xml"
	} else if strings.HasSuffix(path, ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(path, ".ico") {
		contentType = "image/x-icon"
	} else if strings.HasSuffix(path, ".woff") {
		contentType = "font/woff"
	} else if strings.HasSuffix(path, ".woff2") {
		contentType = "font/woff2"
	} else if strings.HasSuffix(path, ".txt") {
		contentType = "text/plain"
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Data(http.StatusOK, contentType, content)
}
