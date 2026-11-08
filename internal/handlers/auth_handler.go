package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/services"
)

// AuthHandler expone los endpoints de autenticación.
type AuthHandler struct {
	auth *services.AuthService
}

// NewAuthHandler construye el handler de autenticación.
func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register godoc
//
//	@Summary		Registrar usuario
//	@Description	Crea una cuenta y devuelve un par de tokens (access + refresh).
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RegisterRequest	true	"Datos de registro"
//	@Success		201		{object}	dto.AuthResponseDoc
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		409		{object}	dto.ErrorResponse
//	@Failure		429		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	response, err := h.auth.Register(c.Request.Context(), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	respondData(c, http.StatusCreated, response)
}

// Login godoc
//
//	@Summary		Iniciar sesión
//	@Description	Valida email y contraseña y devuelve un par de tokens.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.LoginRequest	true	"Credenciales"
//	@Success		200		{object}	dto.AuthResponseDoc
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		429		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	response, err := h.auth.Login(c.Request.Context(), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	respondData(c, http.StatusOK, response)
}

// Refresh godoc
//
//	@Summary		Renovar tokens
//	@Description	Rota el refresh token: lo revoca y devuelve un par nuevo. Reutilizar un token revocado responde 401.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RefreshRequest	true	"Refresh token vigente"
//	@Success		200		{object}	dto.AuthResponseDoc
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	response, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	respondData(c, http.StatusOK, response)
}

// Logout godoc
//
//	@Summary		Cerrar sesión
//	@Description	Revoca el refresh token indicado. La operación es idempotente.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.LogoutRequest	true	"Refresh token a revocar"
//	@Success		200		{object}	dto.MessageResponseDoc
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		500		{object}	dto.ErrorResponse
//	@Router			/api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}

	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		respondServiceError(c, err)
		return
	}

	respondData(c, http.StatusOK, dto.MessageResponse{Message: "sesión cerrada"})
}
