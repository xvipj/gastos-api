package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/xavie/gastos-api/internal/models"
)

// CategoryRepository encapsula las consultas de categorías.
type CategoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository crea el repositorio de categorías.
func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// ListByUser devuelve las categorías del usuario ordenadas por nombre.
func (r *CategoryRepository) ListByUser(ctx context.Context, userID uint) ([]models.Category, error) {
	var categories []models.Category
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("name ASC").
		Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("listando categorías: %w", err)
	}
	return categories, nil
}

// FindByIDAndUser busca una categoría garantizando la pertenencia al usuario.
func (r *CategoryRepository) FindByIDAndUser(ctx context.Context, id, userID uint) (*models.Category, error) {
	var category models.Category
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("buscando categoría: %w", err)
	}
	return &category, nil
}

// Create inserta una categoría nueva.
func (r *CategoryRepository) Create(ctx context.Context, category *models.Category) error {
	if err := r.db.WithContext(ctx).Create(category).Error; err != nil {
		return fmt.Errorf("creando categoría: %w", err)
	}
	return nil
}

// Update guarda los campos modificables de una categoría.
func (r *CategoryRepository) Update(ctx context.Context, category *models.Category) error {
	if err := r.db.WithContext(ctx).
		Model(category).
		Updates(map[string]any{"name": category.Name, "type": category.Type}).Error; err != nil {
		return fmt.Errorf("actualizando categoría: %w", err)
	}
	return nil
}

// Delete elimina una categoría del usuario.
func (r *CategoryRepository) Delete(ctx context.Context, id, userID uint) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Category{})
	if result.Error != nil {
		return fmt.Errorf("eliminando categoría: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ExistsByNameAndType comprueba la unicidad de nombre por usuario y tipo.
// excludeID permite ignorar la propia categoría durante una actualización.
func (r *CategoryRepository) ExistsByNameAndType(ctx context.Context, userID uint, name, categoryType string, excludeID uint) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&models.Category{}).
		Where("user_id = ? AND name = ? AND type = ?", userID, name, categoryType)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("comprobando categoría duplicada: %w", err)
	}
	return count > 0, nil
}

// CountTransactions cuenta las transacciones asociadas a una categoría.
func (r *CategoryRepository) CountTransactions(ctx context.Context, categoryID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("category_id = ?", categoryID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("contando transacciones de la categoría: %w", err)
	}
	return count, nil
}
