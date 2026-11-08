package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/xavie/gastos-api/internal/cache"
	"github.com/xavie/gastos-api/internal/dto"
)

// HealthHandler comprueba que la API, la base de datos y Redis estén operativos.
type HealthHandler struct {
	db    *gorm.DB
	cache *cache.Client
}

// NewHealthHandler construye el handler de salud.
func NewHealthHandler(db *gorm.DB, cacheClient *cache.Client) *HealthHandler {
	return &HealthHandler{db: db, cache: cacheClient}
}

// Check godoc
//
//	@Summary		Estado de la API
//	@Description	Comprueba SQLite y, si está configurado, Redis. No requiere autenticación.
//	@Tags			Sistema
//	@Produce		json
//	@Success		200	{object}	dto.HealthResponseDoc
//	@Failure		503	{object}	dto.HealthResponseDoc
//	@Router			/health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]string{"database": "up", "redis": "disabled"}
	status := "ok"
	httpStatus := http.StatusOK

	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		checks["database"] = "down"
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	if h.cache.Enabled() {
		if err := h.cache.Ping(ctx); err != nil {
			checks["redis"] = "down"
			// Redis es opcional: la API sigue siendo funcional, pero se reporta degradada.
			if status == "ok" {
				status = "degraded"
			}
		} else {
			checks["redis"] = "up"
		}
	}

	respondData(c, httpStatus, dto.HealthResponse{Status: status, Checks: checks})
}
