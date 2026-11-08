package models

import "time"

// Budget define un límite de gasto mensual para una categoría concreta.
type Budget struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index;uniqueIndex:idx_budgets_user_category_period" json:"user_id"`
	CategoryID  uint      `gorm:"not null;index;uniqueIndex:idx_budgets_user_category_period" json:"category_id"`
	Category    Category  `gorm:"constraint:OnDelete:RESTRICT" json:"category,omitempty"`
	Month       int       `gorm:"not null;uniqueIndex:idx_budgets_user_category_period" json:"month"`
	Year        int       `gorm:"not null;uniqueIndex:idx_budgets_user_category_period" json:"year"`
	LimitAmount int64     `gorm:"not null" json:"limit_amount"`
	CreatedAt   time.Time `json:"created_at"`
}
