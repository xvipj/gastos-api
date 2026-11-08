package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/xavie/gastos-api/internal/models"
)

// TransactionFilters define los criterios de búsqueda aplicables a transacciones.
type TransactionFilters struct {
	UserID     uint
	From       string
	To         string
	CategoryID uint
	Type       string
	MinAmount  *int64
	MaxAmount  *int64
	Page       int
	Limit      int
}

// TransactionRepository encapsula las consultas de transacciones.
type TransactionRepository struct {
	db *gorm.DB
}

// NewTransactionRepository crea el repositorio de transacciones.
func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// List devuelve una página de transacciones y el total que cumple los filtros.
func (r *TransactionRepository) List(ctx context.Context, filters TransactionFilters) ([]models.Transaction, int64, error) {
	var total int64
	if err := r.baseQuery(ctx, filters).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("contando transacciones: %w", err)
	}

	var transactions []models.Transaction
	if err := r.baseQuery(ctx, filters).
		Preload("Category").
		Order("date DESC, id DESC").
		Offset((filters.Page - 1) * filters.Limit).
		Limit(filters.Limit).
		Find(&transactions).Error; err != nil {
		return nil, 0, fmt.Errorf("listando transacciones: %w", err)
	}
	return transactions, total, nil
}

// ListAll devuelve todas las transacciones que cumplen los filtros (para CSV).
func (r *TransactionRepository) ListAll(ctx context.Context, filters TransactionFilters) ([]models.Transaction, error) {
	var transactions []models.Transaction
	if err := r.baseQuery(ctx, filters).
		Preload("Category").
		Order("date DESC, id DESC").
		Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("exportando transacciones: %w", err)
	}
	return transactions, nil
}

// FindByIDAndUser busca una transacción garantizando la pertenencia al usuario.
func (r *TransactionRepository) FindByIDAndUser(ctx context.Context, id, userID uint) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Where("id = ? AND user_id = ?", id, userID).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("buscando transacción: %w", err)
	}
	return &transaction, nil
}

// Create inserta una transacción nueva.
func (r *TransactionRepository) Create(ctx context.Context, transaction *models.Transaction) error {
	if err := r.db.WithContext(ctx).Create(transaction).Error; err != nil {
		return fmt.Errorf("creando transacción: %w", err)
	}
	return nil
}

// Update guarda los campos modificables de una transacción.
func (r *TransactionRepository) Update(ctx context.Context, transaction *models.Transaction) error {
	result := r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("id = ? AND user_id = ?", transaction.ID, transaction.UserID).
		Updates(map[string]any{
			"category_id": transaction.CategoryID,
			"type":        transaction.Type,
			"amount":      transaction.Amount,
			"description": transaction.Description,
			"date":        transaction.Date,
			"updated_at":  time.Now().UTC(),
		})
	if result.Error != nil {
		return fmt.Errorf("actualizando transacción: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete elimina una transacción del usuario.
func (r *TransactionRepository) Delete(ctx context.Context, id, userID uint) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Transaction{})
	if result.Error != nil {
		return fmt.Errorf("eliminando transacción: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *TransactionRepository) baseQuery(ctx context.Context, filters TransactionFilters) *gorm.DB {
	query := r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("user_id = ?", filters.UserID)

	if filters.From != "" {
		query = query.Where("date >= ?", filters.From)
	}
	if filters.To != "" {
		query = query.Where("date <= ?", filters.To)
	}
	if filters.CategoryID != 0 {
		query = query.Where("category_id = ?", filters.CategoryID)
	}
	if filters.Type != "" {
		query = query.Where("type = ?", filters.Type)
	}
	if filters.MinAmount != nil {
		query = query.Where("amount >= ?", *filters.MinAmount)
	}
	if filters.MaxAmount != nil {
		query = query.Where("amount <= ?", *filters.MaxAmount)
	}
	return query
}
