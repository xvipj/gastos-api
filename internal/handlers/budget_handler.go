package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/middleware"
	"github.com/xavie/gastos-api/internal/services"
)

// BudgetHandler expone los endpoints de presupuestos.
type BudgetHandler struct {
	budgets *services.BudgetService
}

// NewBudgetHandler construye el handler de presupuestos.
func NewBudgetHandler(budgets *services.BudgetService) *BudgetHandler {
	return &BudgetHandler{budgets: budgets}
}

// List godoc
//
//	@Summary		Listar presupuestos
//	@Description	Devuelve los presupuestos con spent, remaining, percentage_used y status.
//	@Tags			Presupuestos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			month	query		int	false	"Mes (1-12)"
//	@Param			year	query		int	false	"Año (2000-2100)"
//	@Success		200		{object}	dto.BudgetsResponseDoc
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/api/v1/budgets [get]
func (h *BudgetHandler) List(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	var filter dto.BudgetFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		respondBindError(c, err)
		return
	}

	budgets, err := h.budgets.List(c.Request.Context(), userID, filter)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusOK, budgets)
}

// Create godoc
//
//	@Summary		Crear presupuesto
//	@Description	Crea un límite mensual para una categoría de gasto.
//	@Tags			Presupuestos
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.BudgetRequest	true	"Datos del presupuesto"
//	@Success		201		{object}	dto.BudgetResponseDoc
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Failure		422		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/api/v1/budgets [post]
func (h *BudgetHandler) Create(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	var req dto.BudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	budget, err := h.budgets.Create(c.Request.Context(), userID, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusCreated, budget)
}

// Update godoc
//
//	@Summary		Actualizar presupuesto
//	@Description	Modifica un presupuesto existente del usuario.
//	@Tags			Presupuestos
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"ID del presupuesto"
//	@Param			request	body		dto.BudgetRequest	true	"Datos actualizados"
//	@Success		200		{object}	dto.BudgetResponseDoc
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Failure		422		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/api/v1/budgets/{id} [put]
func (h *BudgetHandler) Update(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	budgetID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req dto.BudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	budget, err := h.budgets.Update(c.Request.Context(), userID, budgetID, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusOK, budget)
}

// Delete godoc
//
//	@Summary		Eliminar presupuesto
//	@Description	Elimina un presupuesto del usuario autenticado.
//	@Tags			Presupuestos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"ID del presupuesto"
//	@Success		200	{object}	dto.MessageResponseDoc
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Router			/api/v1/budgets/{id} [delete]
func (h *BudgetHandler) Delete(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	budgetID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.budgets.Delete(c.Request.Context(), userID, budgetID); err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusOK, dto.MessageResponse{Message: "presupuesto eliminado"})
}
