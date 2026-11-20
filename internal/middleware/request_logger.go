package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const requestIDContextKey = "navbox_request_id"

func RequestLogger(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Set(requestIDContextKey, reqID)
		c.Header("X-Request-ID", reqID)

		start := time.Now()
		c.Next()

		logger.Info().
			Str("request_id", reqID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start)).
			Msg("HTTP request completed")
	}
}

func GetRequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v, ok := c.Get(requestIDContextKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
