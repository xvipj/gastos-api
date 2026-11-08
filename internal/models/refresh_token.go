package models

import "time"

// RefreshToken guarda únicamente el hash SHA-256 del token de refresco.
// El token en claro solo se entrega al cliente en la respuesta HTTP.
type RefreshToken struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"not null;index" json:"user_id"`
	TokenHash string     `gorm:"size:64;uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt *time.Time `gorm:"index" json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
