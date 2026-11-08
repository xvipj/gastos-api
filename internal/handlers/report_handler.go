package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/middleware"
	"github.com/xavie/gastos-api/internal/services"
)

type summaryQuery struct {
	Month int `form:"month" binding:"omitempty,min=1,max=12"`
	Year  int `form:"year" binding:"omitempty,min=2000,max=2100"`
}

type trendQuery struct {
	Months int `form:"months" binding:"omitempty,min=1,max=24"`
}

// ReportHandler expone los reportes financieros.
type ReportHandler struct {
	reports *services.ReportService
}

// NewReportHandler construye el handler de reportes.
func NewReportHandler(reports *services.ReportService) *ReportHandler {
	return &ReportHandler{reports: reports}
}

// Summary godoc
//
//	@Summary		Resumen mensual
//	@Description	Totales de ingresos, gastos, balance y gasto agrupado por categoría. Si no se envían month/year se usa el mes actual.
//	@Tags			Reportes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			month	query		int	false	"Mes (1-12)"
//	@Param			year	query		int	false	"Año (2000-2100)"
//	@Success		200		{object}	dto.SummaryResponseDoc
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/api/v1/reports/summary [get]
func (h *ReportHandler) Summary(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	var query summaryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondBindError(c, err)
		return
	}

	if query.Month == 0 && query.Year == 0 {
		now := time.Now().UTC()
		query.Month = int(now.Month())
		query.Year = now.Year()
	}

	summary, err := h.reports.Summary(c.Request.Context(), userID, query.Month, query.Year)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusOK, summary)
}

// Trend godoc
//
//	@Summary		Tendencia mensual
//	@Description	Ingresos, gastos y balance de cada uno de los últimos N meses (6 por defecto).
//	@Tags			Reportes
//	@Produce		json
//	@Security		BearerAuth
//	@Param			months	query		int	false	"Cantidad de meses (1-24, por defecto 6)"
//	@Success		200		{object}	dto.TrendResponseDoc
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/api/v1/reports/trend [get]
func (h *ReportHandler) Trend(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	var query trendQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondBindError(c, err)
		return
	}

	trend, err := h.reports.Trend(c.Request.Context(), userID, query.Months)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusOK, trend)
}
