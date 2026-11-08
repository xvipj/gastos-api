package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/xavie/gastos-api/internal/dto"
)

// RateLimiter implementa un límite fijo por IP usando Redis. Si Redis falla
// permite la petición (fail-open) y deja un warning en el log.
type RateLimiter struct {
	client *redis.Client
	log    *slog.Logger
	limit  int64
	window time.Duration
}

// NewRateLimiter construye el limitador. Un cliente nil desactiva el límite.
func NewRateLimiter(client *redis.Client, log *slog.Logger, limit int64, window time.Duration) *RateLimiter {
	return &RateLimiter{client: client, log: log, limit: limit, window: window}
}

// Limit aplica el límite a un endpoint concreto.
func (r *RateLimiter) Limit(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if r == nil || r.client == nil {
			c.Next()
			return
		}

		key := fmt.Sprintf("ratelimit:%s:%s", scope, c.ClientIP())
		ctx, cancel := context.WithTimeout(c.Request.Context(), 500*time.Millisecond)
		defer cancel()

		count, err := r.client.Incr(ctx, key).Result()
		if err != nil {
			r.log.Warn("rate limiting no disponible; se permite la petición",
				"scope", scope,
				"ip", c.ClientIP(),
				"error", err,
			)
			c.Next()
			return
		}

		if count == 1 {
			if err := r.client.Expire(ctx, key, r.window).Err(); err != nil {
				r.log.Warn("no se pudo establecer el TTL del rate limit",
					"scope", scope,
					"error", err,
				)
			}
		}

		if count > r.limit {
			c.Header("Retry-After", strconv.FormatInt(int64(r.window.Seconds()), 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, dto.NewErrorResponse(
				dto.CodeTooManyRequests,
				fmt.Sprintf("demasiados intentos; vuelve a intentarlo en %d segundos", int64(r.window.Seconds())),
			))
			return
		}

		c.Next()
	}
}
