package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/middleware"
	"github.com/xavie/gastos-api/internal/services"
)

// TransactionHandler expone los endpoints de transacciones.
type TransactionHandler struct {
	transactions *services.TransactionService
}

// NewTransactionHandler construye el handler de transacciones.
func NewTransactionHandler(transactions *services.TransactionService) *TransactionHandler {
	return &TransactionHandler{transactions: transactions}
}

// List godoc
//
//	@Summary		Listar transacciones
//	@Description	Devuelve transacciones paginadas del usuario con filtros opcionales.
//	@Tags			Transacciones
//	@Produce		json
//	@Security		BearerAuth
//	@Param			from		query		string	false	"Fecha inicial (YYYY-MM-DD)"
//	@Param			to			query		string	false	"Fecha final (YYYY-MM-DD)"
//	@Param			category_id	query		int		false	"ID de categoría"
//	@Param			type		query		string	false	"Tipo de movimiento"	Enums(income, expense)
//	@Param			min_amount	query		int		false	"Monto mínimo en centavos"
//	@Param			max_amount	query		int		false	"Monto máximo en centavos"
//	@Param			page		query		int		false	"Página (por defecto 1)"
//	@Param			limit		query		int		false	"Tamaño de página (por defecto 20, máximo 100)"
//	@Success		200			{object}	dto.TransactionsResponseDoc
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/api/v1/transactions [get]
func (h *TransactionHandler) List(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	var filter dto.TransactionFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		respondBindError(c, err)
		return
	}

	transactions, total, normalized, err := h.transactions.List(c.Request.Context(), userID, filter)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	respondPaginated(c, transactions, normalized.Page, normalized.Limit, total)
}

// Get godoc
//
//	@Summary		Obtener transacción
//	@Description	Devuelve una transacción del usuario autenticado.
//	@Tags			Transacciones
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"ID de la transacción"
//	@Success		200	{object}	dto.TransactionResponseDoc
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Router			/api/v1/transactions/{id} [get]
func (h *TransactionHandler) Get(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	transactionID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	transaction, err := h.transactions.Get(c.Request.Context(), userID, transactionID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusOK, transaction)
}

// Create godoc
//
//	@Summary		Crear transacción
//	@Description	Registra un ingreso o gasto. El tipo debe coincidir con el de la categoría.
//	@Tags			Transacciones
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.TransactionRequest	true	"Datos de la transacción"
//	@Success		201		{object}	dto.TransactionResponseDoc
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		422		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/api/v1/transactions [post]
func (h *TransactionHandler) Create(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	var req dto.TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	transaction, err := h.transactions.Create(c.Request.Context(), userID, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusCreated, transaction)
}

// Update godoc
//
//	@Summary		Actualizar transacción
//	@Description	Modifica una transacción existente del usuario autenticado.
//	@Tags			Transacciones
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int						true	"ID de la transacción"
//	@Param			request	body		dto.TransactionRequest	true	"Datos actualizados"
//	@Success		200		{object}	dto.TransactionResponseDoc
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Failure		422		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/api/v1/transactions/{id} [put]
func (h *TransactionHandler) Update(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	transactionID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req dto.TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	transaction, err := h.transactions.Update(c.Request.Context(), userID, transactionID, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusOK, transaction)
}

// Delete godoc
//
//	@Summary		Eliminar transacción
//	@Description	Elimina una transacción del usuario autenticado.
//	@Tags			Transacciones
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"ID de la transacción"
//	@Success		200	{object}	dto.MessageResponseDoc
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Router			/api/v1/transactions/{id} [delete]
func (h *TransactionHandler) Delete(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	transactionID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.transactions.Delete(c.Request.Context(), userID, transactionID); err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusOK, dto.MessageResponse{Message: "transacción eliminada"})
}

// Export godoc
//
//	@Summary		Exportar transacciones
//	@Description	Descarga un CSV con las transacciones que cumplen los mismos filtros del listado.
//	@Tags			Transacciones
//	@Produce		text/csv
//	@Security		BearerAuth
//	@Param			from		query		string	false	"Fecha inicial (YYYY-MM-DD)"
//	@Param			to			query		string	false	"Fecha final (YYYY-MM-DD)"
//	@Param			category_id	query		int		false	"ID de categoría"
//	@Param			type		query		string	false	"Tipo de movimiento"	Enums(income, expense)
//	@Param			min_amount	query		int		false	"Monto mínimo en centavos"
//	@Param			max_amount	query		int		false	"Monto máximo en centavos"
//	@Success		200			{file}		file	"Archivo CSV"
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		401			{object}	dto.ErrorResponse
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/api/v1/transactions/export [get]
func (h *TransactionHandler) Export(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	var filter dto.TransactionFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		respondBindError(c, err)
		return
	}

	transactions, err := h.transactions.Export(c.Request.Context(), userID, filter)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{"id", "date", "type", "category", "amount", "description"})
	for _, transaction := range transactions {
		_ = writer.Write([]string{
			strconv.FormatUint(uint64(transaction.ID), 10),
			transaction.Date,
			transaction.Type,
			transaction.CategoryName,
			formatCents(transaction.Amount),
			transaction.Description,
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		slog.Error("generando CSV de transacciones", "error", err)
		respondError(c, http.StatusInternalServerError, dto.CodeInternal, "no se pudo generar el CSV")
		return
	}

	filename := fmt.Sprintf("transactions-%s.csv", time.Now().UTC().Format("20060102"))
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
}

// formatCents convierte centavos a una representación decimal sin usar floats.
func formatCents(amount int64) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}
	return fmt.Sprintf("%s%d.%02d", sign, amount/100, amount%100)
}
