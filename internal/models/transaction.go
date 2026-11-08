package models

import "time"

// Transaction representa un ingreso o gasto. El importe se almacena como un
// entero en la unidad mínima de la moneda (centavos) para evitar errores de
// redondeo propios de los tipos de coma flotante.
type Transaction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index;index:idx_transactions_user_date,priority:1" json:"user_id"`
	CategoryID  uint      `gorm:"not null;index" json:"category_id"`
	Category    Category  `gorm:"constraint:OnDelete:RESTRICT" json:"category,omitempty"`
	Type        string    `gorm:"size:20;not null" json:"type"`
	Amount      int64     `gorm:"not null" json:"amount"`
	Description string    `gorm:"size:255" json:"description"`
	Date        string    `gorm:"type:date;not null;index:idx_transactions_user_date,priority:2" json:"date"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
