package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/xavie/gastos-api/internal/models"
)

// RefreshTokenRepository persiste y revoca tokens de refresco.
type RefreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository crea el repositorio de refresh tokens.
func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create guarda un refresh token (siempre con su hash, nunca en claro).
func (r *RefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("guardando refresh token: %w", err)
	}
	return nil
}

// FindByHash localiza un token por su hash, incluso si ya fue revocado.
// Devolver también los revocados permite detectar intentos de reutilización.
func (r *RefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("buscando refresh token: %w", err)
	}
	return &token, nil
}

// Revoke revoca un token concreto si todavía estaba activo.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", time.Now().UTC())
	if result.Error != nil {
		return fmt.Errorf("revocando refresh token: %w", result.Error)
	}
	return nil
}

// Rotate revoca el token anterior y guarda el nuevo dentro de una transacción.
// El UPDATE condicional garantiza que solo una petición pueda rotar el token.
func (r *RefreshTokenRepository) Rotate(ctx context.Context, oldID uint, newToken *models.RefreshToken) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.RefreshToken{}).
			Where("id = ? AND revoked_at IS NULL", oldID).
			Update("revoked_at", time.Now().UTC())
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrTokenAlreadyRevoked
		}
		return tx.Create(newToken).Error
	})
	if err != nil {
		if errors.Is(err, ErrTokenAlreadyRevoked) {
			return ErrTokenAlreadyRevoked
		}
		return fmt.Errorf("rotando refresh token: %w", err)
	}
	return nil
}

// RevokeAllForUser revoca todas las sesiones activas del usuario.
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uint) error {
	result := r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now().UTC())
	if result.Error != nil {
		return fmt.Errorf("revocando refresh tokens del usuario: %w", result.Error)
	}
	return nil
}
