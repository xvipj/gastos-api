package dto

import "time"

// BudgetRequest es el cuerpo para crear o actualizar un presupuesto.
// LimitAmount se expresa en centavos.
type BudgetRequest struct {
	CategoryID  uint  `json:"category_id" binding:"required" example:"3"`
	Month       int   `json:"month" binding:"required,min=1,max=12" example:"11"`
	Year        int   `json:"year" binding:"required,min=2000,max=2100" example:"2026"`
	LimitAmount int64 `json:"limit_amount" binding:"required,gt=0" example:"500000"`
}

// BudgetFilter agrupa los filtros opcionales de listado de presupuestos.
type BudgetFilter struct {
	Month int `form:"month" binding:"omitempty,min=1,max=12"`
	Year  int `form:"year" binding:"omitempty,min=2000,max=2100"`
}

// BudgetResponse incluye el cálculo de gasto y el estado del presupuesto.
type BudgetResponse struct {
	ID             uint      `json:"id" example:"1"`
	CategoryID     uint      `json:"category_id" example:"3"`
	CategoryName   string    `json:"category_name" example:"Alimentación"`
	Month          int       `json:"month" example:"11"`
	Year           int       `json:"year" example:"2026"`
	LimitAmount    int64     `json:"limit_amount" example:"500000"`
	Spent          int64     `json:"spent" example:"425000"`
	Remaining      int64     `json:"remaining" example:"75000"`
	PercentageUsed float64   `json:"percentage_used" example:"85"`
	Status         string    `json:"status" example:"warning"`
	CreatedAt      time.Time `json:"created_at"`
}

// BudgetResponseDoc documenta un presupuesto en Swagger.
type BudgetResponseDoc struct {
	Data BudgetResponse `json:"data"`
}

// BudgetsResponseDoc documenta un listado de presupuestos en Swagger.
type BudgetsResponseDoc struct {
	Data []BudgetResponse `json:"data"`
}
