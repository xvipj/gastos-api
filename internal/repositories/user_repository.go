package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/xavie/gastos-api/internal/models"
)

// UserRepository encapsula las operaciones sobre usuarios.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository crea el repositorio de usuarios.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserta un usuario nuevo.
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("creando usuario: %w", err)
	}
	return nil
}

// FindByEmail busca un usuario por su correo normalizado.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("buscando usuario por email: %w", err)
	}
	return &user, nil
}

// FindByID busca un usuario por su identificador.
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("buscando usuario por id: %w", err)
	}
	return &user, nil
}
