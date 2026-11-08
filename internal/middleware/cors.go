package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS habilita el consumo de la API desde cualquier origen. Es una política
// deliberadamente abierta porque la API se autentica con Bearer tokens y no
// con cookies.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		c.Header("Access-Control-Expose-Headers", "Content-Disposition")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
