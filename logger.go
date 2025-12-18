package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

type HTTPLogEntry struct {
	Timestamp  string  `json:"ts"`
	Method     string  `json:"method"`
	Path       string  `json:"path"`
	Status     int     `json:"status"`
	LatencyMs  int64   `json:"latency_ms"`
	ClientIP   string  `json:"client_ip"`
	UserAgent  string  `json:"user_agent"`
	UserID     *int    `json:"user_id,omitempty"`
	TraceID    string  `json:"trace_id,omitempty"`
	SpanID     string  `json:"span_id,omitempty"`
	RequestID  string  `json:"request_id,omitempty"`
}

// FilteredLogger logs all requests in JSON EXCEPT /healthz and /readyz
func FilteredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip health probes
		if path == "/healthz" || path == "/readyz" {
			c.Next()
			return
		}

		start := time.Now()

		// Let the handler run
		c.Next()

		latency := time.Since(start).Milliseconds()
		status := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()
		ua := c.Request.UserAgent()

		// Optional: userID from context (set by AuthMiddleware)
		var userIDPtr *int
		if val, ok := c.Get("userID"); ok {
			if id, ok := val.(int); ok {
				userIDPtr = &id
			}
		}

		// Optional: future OTEL / trace headers
		traceID := c.Request.Header.Get("X-Trace-Id")
		spanID := c.Request.Header.Get("X-Span-Id")
		requestID := c.Request.Header.Get("X-Request-Id")

		entry := HTTPLogEntry{
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Method:    method,
			Path:      path,
			Status:    status,
			LatencyMs: latency,
			ClientIP:  clientIP,
			UserAgent: ua,
			UserID:    userIDPtr,
			TraceID:   traceID,
			SpanID:    spanID,
			RequestID: requestID,
		}

		b, err := json.Marshal(entry)
		if err != nil {
			log.Printf("failed to marshal http log: %v", err)
			return
		}

		log.Println(string(b))
	}
}
