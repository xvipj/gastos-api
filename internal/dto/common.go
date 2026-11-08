package dto

// Códigos de error estables que expone la API.
const (
	CodeValidation      = "VALIDATION_ERROR"
	CodeUnauthorized    = "UNAUTHORIZED"
	CodeNotFound        = "NOT_FOUND"
	CodeConflict        = "CONFLICT"
	CodeUnprocessable   = "UNPROCESSABLE_ENTITY"
	CodeTooManyRequests = "TOO_MANY_REQUESTS"
	CodeInternal        = "INTERNAL_ERROR"
)

// ErrorDetail describe un error de la API.
type ErrorDetail struct {
	Code    string `json:"code" example:"VALIDATION_ERROR"`
	Message string `json:"message" example:"el campo email es obligatorio"`
}

// ErrorResponse es el formato uniforme de todos los errores.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// NewErrorResponse construye una respuesta de error.
func NewErrorResponse(code, message string) ErrorResponse {
	return ErrorResponse{Error: ErrorDetail{Code: code, Message: message}}
}

// Meta contiene los metadatos de una respuesta paginada.
type Meta struct {
	Page  int   `json:"page" example:"1"`
	Limit int   `json:"limit" example:"20"`
	Total int64 `json:"total" example:"42"`
}

// PaginatedResponse es el sobre de una respuesta con paginación.
type PaginatedResponse struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
}

// MessageResponse es una respuesta sencilla con un mensaje.
type MessageResponse struct {
	Message string `json:"message" example:"operación realizada correctamente"`
}

// MessageResponseDoc documenta un mensaje en Swagger.
type MessageResponseDoc struct {
	Data MessageResponse `json:"data"`
}
