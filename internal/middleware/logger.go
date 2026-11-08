package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger registra cada petición HTTP con el logger estructurado de la aplicación.
func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
			"size", c.Writer.Size(),
		}
		if query != "" {
			attrs = append(attrs, "query", query)
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		if c.Writer.Status() >= 500 {
			log.Error("http request", attrs...)
			return
		}
		log.Info("http request", attrs...)
	}
}
