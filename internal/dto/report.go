package dto

// CategoryExpense es el gasto agregado de una categoría dentro de un resumen.
type CategoryExpense struct {
	CategoryID   uint    `json:"category_id" example:"3"`
	CategoryName string  `json:"category_name" example:"Alimentación"`
	Amount       int64   `json:"amount" example:"125050"`
	Percentage   float64 `json:"percentage" example:"42.35"`
}

// SummaryResponse es el resumen financiero de un mes.
type SummaryResponse struct {
	Month        int               `json:"month" example:"11"`
	Year         int               `json:"year" example:"2026"`
	TotalIncome  int64             `json:"total_income" example:"2500000"`
	TotalExpense int64             `json:"total_expense" example:"1250500"`
	Balance      int64             `json:"balance" example:"1249500"`
	ByCategory   []CategoryExpense `json:"by_category"`
}

// SummaryResponseDoc documenta el resumen mensual en Swagger.
type SummaryResponseDoc struct {
	Data SummaryResponse `json:"data"`
}

// TrendPoint es el agregado de un mes dentro de la tendencia.
type TrendPoint struct {
	Month   string `json:"month" example:"2026-11"`
	Income  int64  `json:"income" example:"2500000"`
	Expense int64  `json:"expense" example:"1250500"`
	Balance int64  `json:"balance" example:"1249500"`
}

// TrendResponse contiene la evolución de los últimos meses.
type TrendResponse struct {
	Months int          `json:"months" example:"6"`
	Trend  []TrendPoint `json:"trend"`
}

// TrendResponseDoc documenta la tendencia en Swagger.
type TrendResponseDoc struct {
	Data TrendResponse `json:"data"`
}
