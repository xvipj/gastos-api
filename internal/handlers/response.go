package handlers

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/services"
)

func respondData(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

func respondPaginated(c *gin.Context, data any, page, limit int, total int64) {
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"meta": dto.Meta{Page: page, Limit: limit, Total: total},
	})
}

func respondError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, dto.NewErrorResponse(code, message))
}

// respondBindError traduce errores de binding/validación de Gin a mensajes claros.
func respondBindError(c *gin.Context, err error) {
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		messages := make([]string, 0, len(validationErrors))
		for _, fieldError := range validationErrors {
			messages = append(messages, fieldErrorMessage(fieldError))
		}
		respondError(c, http.StatusBadRequest, dto.CodeValidation, strings.Join(messages, "; "))
		return
	}

	if errors.Is(err, io.EOF) {
		respondError(c, http.StatusBadRequest, dto.CodeValidation, "el cuerpo de la petición es obligatorio")
		return
	}

	respondError(c, http.StatusBadRequest, dto.CodeValidation, "cuerpo o parámetros de la petición inválidos")
}

// respondServiceError traduce los errores de negocio a códigos HTTP.
func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrValidation):
		respondError(c, http.StatusBadRequest, dto.CodeValidation, err.Error())
	case errors.Is(err, services.ErrUnprocessable):
		respondError(c, http.StatusUnprocessableEntity, dto.CodeUnprocessable, err.Error())
	case errors.Is(err, services.ErrUnauthorized):
		respondError(c, http.StatusUnauthorized, dto.CodeUnauthorized, err.Error())
	case errors.Is(err, services.ErrNotFound):
		respondError(c, http.StatusNotFound, dto.CodeNotFound, err.Error())
	case errors.Is(err, services.ErrConflict):
		respondError(c, http.StatusConflict, dto.CodeConflict, err.Error())
	default:
		slog.Error("error no controlado",
			"error", err,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
		)
		respondError(c, http.StatusInternalServerError, dto.CodeInternal, "error interno del servidor")
	}
}

func fieldErrorMessage(fieldError validator.FieldError) string {
	field := fieldError.Field()
	tag := fieldError.Tag()

	switch tag {
	case "required":
		return "el campo " + field + " es obligatorio"
	case "email":
		return "el campo " + field + " debe ser un email válido"
	case "min":
		if fieldError.Kind() == reflect.String {
			return "el campo " + field + " debe tener al menos " + fieldError.Param() + " caracteres"
		}
		return "el campo " + field + " debe ser mayor o igual a " + fieldError.Param()
	case "max":
		if fieldError.Kind() == reflect.String {
			return "el campo " + field + " debe tener como máximo " + fieldError.Param() + " caracteres"
		}
		return "el campo " + field + " debe ser menor o igual a " + fieldError.Param()
	case "gt":
		return "el campo " + field + " debe ser mayor que " + fieldError.Param()
	case "gte":
		return "el campo " + field + " debe ser mayor o igual a " + fieldError.Param()
	case "lt":
		return "el campo " + field + " debe ser menor que " + fieldError.Param()
	case "lte":
		return "el campo " + field + " debe ser menor o igual a " + fieldError.Param()
	case "oneof":
		return "el campo " + field + " debe ser uno de: " + fieldError.Param()
	case "datetime":
		return "el campo " + field + " debe tener el formato " + fieldError.Param()
	default:
		return "el campo " + field + " no es válido"
	}
}
