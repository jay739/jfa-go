package main

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// securityHeaders adds standard security response headers to every response.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("X-XSS-Protection", "0") // modern browsers use CSP; legacy header disabled intentionally
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		// Only set HSTS when the connection is already TLS (behind a reverse proxy that terminates TLS).
		if c.Request.Header.Get("X-Forwarded-Proto") == "https" {
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		c.Next()
	}
}

// ipBucket tracks request counts per IP within a sliding window.
type ipBucket struct {
	mu       sync.Mutex
	requests []time.Time
}

func (b *ipBucket) allow(limit int, windowSecs int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	cutoff := time.Now().Add(-time.Duration(windowSecs) * time.Second)
	filtered := b.requests[:0]
	for _, t := range b.requests {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	b.requests = filtered
	if len(b.requests) >= limit {
		return false
	}
	b.requests = append(b.requests, time.Now())
	return true
}

var rlStore sync.Map // map[string]*ipBucket

func getBucket(key string) *ipBucket {
	v, _ := rlStore.LoadOrStore(key, &ipBucket{})
	return v.(*ipBucket)
}

// clientIP returns the best-effort real IP from the request.
func clientIP(c *gin.Context) string {
	if fwd := c.Request.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.SplitN(fwd, ",", 2)[0])
	}
	return c.ClientIP()
}

// rateLimitMiddleware limits requests to `limit` per `windowSecs` seconds per IP + path.
func rateLimitMiddleware(limit, windowSecs int) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := clientIP(c) + "|" + c.FullPath()
		if !getBucket(key).allow(limit, windowSecs) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded — try again later",
			})
			return
		}
		c.Next()
	}
}
