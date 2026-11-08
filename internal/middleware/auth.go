package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xavie/gastos-api/internal/dto"
)

const userIDKey = "user_id"

// TokenValidator valida un access token y devuelve el ID del usuario.
type TokenValidator interface {
	ValidateAccessToken(rawToken string) (uint, error)
}

// Auth exige un access token válido en la cabecera Authorization: Bearer <token>.
func Auth(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			abortUnauthorized(c, "falta la cabecera Authorization")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			abortUnauthorized(c, "formato de Authorization inválido; usa Bearer <token>")
			return
		}

		userID, err := validator.ValidateAccessToken(strings.TrimSpace(parts[1]))
		if err != nil {
			abortUnauthorized(c, "access token inválido o expirado")
			return
		}

		c.Set(userIDKey, userID)
		c.Next()
	}
}

// UserID recupera el usuario autenticado guardado por el middleware Auth.
func UserID(c *gin.Context) (uint, bool) {
	value, exists := c.Get(userIDKey)
	if !exists {
		return 0, false
	}
	userID, ok := value.(uint)
	return userID, ok
}

func abortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, dto.NewErrorResponse(dto.CodeUnauthorized, message))
}
