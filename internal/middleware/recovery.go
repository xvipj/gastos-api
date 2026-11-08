package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xavie/gastos-api/internal/dto"
)

// Recovery captura los panics y responde con el formato de error uniforme.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.Error("panic recuperado",
			"error", recovered,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError, dto.NewErrorResponse(
			dto.CodeInternal,
			"error interno del servidor",
		))
	})
}
