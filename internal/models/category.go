package models

import "time"

// Category agrupa transacciones de un mismo tipo dentro de la cuenta de un usuario.
// El nombre es único por usuario y tipo.
type Category struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index;uniqueIndex:idx_categories_user_name_type" json:"user_id"`
	Name      string    `gorm:"size:100;not null;uniqueIndex:idx_categories_user_name_type" json:"name"`
	Type      string    `gorm:"size:20;not null;uniqueIndex:idx_categories_user_name_type" json:"type"`
	CreatedAt time.Time `json:"created_at"`
}
