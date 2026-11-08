package dto

import "time"

// RegisterRequest es el cuerpo de POST /auth/register.
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100" example:"Ana García"`
	Email    string `json:"email" binding:"required,email,max=255" example:"ana@example.com"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"super-secret-123"`
}

// LoginRequest es el cuerpo de POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"ana@example.com"`
	Password string `json:"password" binding:"required" example:"super-secret-123"`
}

// RefreshRequest es el cuerpo de POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"a1b2c3..." `
}

// LogoutRequest es el cuerpo de POST /auth/logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"a1b2c3..."`
}

// UserResponse expone los datos públicos de un usuario.
type UserResponse struct {
	ID        uint      `json:"id" example:"1"`
	Name      string    `json:"name" example:"Ana García"`
	Email     string    `json:"email" example:"ana@example.com"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthResponse contiene el par de tokens y los datos del usuario autenticado.
type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type" example:"Bearer"`
	ExpiresIn    int64        `json:"expires_in" example:"900"`
	User         UserResponse `json:"user"`
}

// AuthResponseDoc documenta el sobre de respuesta de autenticación en Swagger.
type AuthResponseDoc struct {
	Data AuthResponse `json:"data"`
}
