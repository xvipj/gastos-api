package dto

import "time"

// TransactionRequest es el cuerpo para crear o actualizar una transacción.
// Amount se expresa en la unidad mínima de la moneda (centavos).
type TransactionRequest struct {
	CategoryID  uint   `json:"category_id" binding:"required" example:"3"`
	Type        string `json:"type" binding:"required,oneof=income expense" example:"expense"`
	Amount      int64  `json:"amount" binding:"required,gt=0" example:"125050"`
	Description string `json:"description" binding:"max=255" example:"Compra semanal"`
	Date        string `json:"date" binding:"required,datetime=2006-01-02" example:"2026-11-08"`
}

// TransactionFilter agrupa los filtros y la paginación de transacciones.
type TransactionFilter struct {
	From       string `form:"from" binding:"omitempty,datetime=2006-01-02"`
	To         string `form:"to" binding:"omitempty,datetime=2006-01-02"`
	CategoryID uint   `form:"category_id"`
	Type       string `form:"type" binding:"omitempty,oneof=income expense"`
	MinAmount  *int64 `form:"min_amount" binding:"omitempty,gte=0"`
	MaxAmount  *int64 `form:"max_amount" binding:"omitempty,gte=0"`
	Page       int    `form:"page" binding:"omitempty,min=1"`
	Limit      int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

// Normalize aplica los valores por defecto de paginación.
func (f *TransactionFilter) Normalize() {
	if f.Page == 0 {
		f.Page = 1
	}
	if f.Limit == 0 {
		f.Limit = 20
	}
}

// TransactionResponse expone una transacción con el nombre de su categoría.
type TransactionResponse struct {
	ID           uint      `json:"id" example:"10"`
	CategoryID   uint      `json:"category_id" example:"3"`
	CategoryName string    `json:"category_name" example:"Alimentación"`
	Type         string    `json:"type" example:"expense"`
	Amount       int64     `json:"amount" example:"125050"`
	Description  string    `json:"description" example:"Compra semanal"`
	Date         string    `json:"date" example:"2026-11-08"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TransactionResponseDoc documenta una transacción en Swagger.
type TransactionResponseDoc struct {
	Data TransactionResponse `json:"data"`
}

// TransactionsResponseDoc documenta un listado paginado en Swagger.
type TransactionsResponseDoc struct {
	Data []TransactionResponse `json:"data"`
	Meta Meta                  `json:"meta"`
}
