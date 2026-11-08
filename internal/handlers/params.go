package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/xavie/gastos-api/internal/dto"
)

// parseIDParam lee un :id positivo de la ruta y responde 400 si no es válido.
func parseIDParam(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		respondError(c, http.StatusBadRequest, dto.CodeValidation, "el parámetro "+name+" debe ser un entero positivo")
		return 0, false
	}
	return uint(value), true
}
