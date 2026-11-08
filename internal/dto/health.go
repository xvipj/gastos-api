package dto

// HealthResponse describe el estado de la API y sus dependencias.
type HealthResponse struct {
	Status string            `json:"status" example:"ok"`
	Checks map[string]string `json:"checks"`
}

// HealthResponseDoc documenta /health en Swagger.
type HealthResponseDoc struct {
	Data HealthResponse `json:"data"`
}
