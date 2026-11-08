package models

import "time"

// User representa a una persona registrada en la API.
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:100;not null" json:"name"`
	Email        string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
