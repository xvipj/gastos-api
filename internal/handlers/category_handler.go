package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/middleware"
	"github.com/xavie/gastos-api/internal/services"
)

// CategoryHandler expone los endpoints de categorías.
type CategoryHandler struct {
	categories *services.CategoryService
}

// NewCategoryHandler construye el handler de categorías.
func NewCategoryHandler(categories *services.CategoryService) *CategoryHandler {
	return &CategoryHandler{categories: categories}
}

// List godoc
//
//	@Summary		Listar categorías
//	@Description	Devuelve las categorías del usuario autenticado.
//	@Tags			Categorías
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	dto.CategoriesResponseDoc
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Router			/api/v1/categories [get]
func (h *CategoryHandler) List(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	categories, err := h.categories.List(c.Request.Context(), userID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusOK, categories)
}

// Create godoc
//
//	@Summary		Crear categoría
//	@Description	Crea una categoría de tipo income o expense para el usuario autenticado.
//	@Tags			Categorías
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.CategoryRequest	true	"Datos de la categoría"
//	@Success		201		{object}	dto.CategoryResponseDoc
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/api/v1/categories [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	var req dto.CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	category, err := h.categories.Create(c.Request.Context(), userID, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusCreated, category)
}

// Update godoc
//
//	@Summary		Actualizar categoría
//	@Description	Modifica el nombre o el tipo de una categoría del usuario.
//	@Tags			Categorías
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"ID de la categoría"
//	@Param			request	body		dto.CategoryRequest	true	"Datos actualizados"
//	@Success		200		{object}	dto.CategoryResponseDoc
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/api/v1/categories/{id} [put]
func (h *CategoryHandler) Update(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	categoryID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req dto.CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	category, err := h.categories.Update(c.Request.Context(), userID, categoryID, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusOK, category)
}

// Delete godoc
//
//	@Summary		Eliminar categoría
//	@Description	Elimina una categoría. Responde 409 si tiene transacciones asociadas.
//	@Tags			Categorías
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"ID de la categoría"
//	@Success		200	{object}	dto.MessageResponseDoc
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Failure		409	{object}	dto.ErrorResponse
//	@Failure		500	{object}	dto.ErrorResponse
//	@Router			/api/v1/categories/{id} [delete]
func (h *CategoryHandler) Delete(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, "usuario no autenticado")
		return
	}

	categoryID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.categories.Delete(c.Request.Context(), userID, categoryID); err != nil {
		respondServiceError(c, err)
		return
	}
	respondData(c, http.StatusOK, dto.MessageResponse{Message: "categoría eliminada"})
}
