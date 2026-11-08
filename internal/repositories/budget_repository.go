package repositories

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/xavie/gastos-api/internal/models"
)

// BudgetSpendKey identifica el gasto de una categoría en un periodo concreto.
type BudgetSpendKey struct {
	Year       int
	Month      int
	CategoryID uint
}

// BudgetRepository encapsula las consultas de presupuestos.
type BudgetRepository struct {
	db *gorm.DB
}

// NewBudgetRepository crea el repositorio de presupuestos.
func NewBudgetRepository(db *gorm.DB) *BudgetRepository {
	return &BudgetRepository{db: db}
}

// ListByUser devuelve los presupuestos del usuario, opcionalmente filtrados por mes y año.
func (r *BudgetRepository) ListByUser(ctx context.Context, userID uint, month, year int) ([]models.Budget, error) {
	query := r.db.WithContext(ctx).
		Preload("Category").
		Where("user_id = ?", userID)
	if month != 0 {
		query = query.Where("month = ?", month)
	}
	if year != 0 {
		query = query.Where("year = ?", year)
	}

	var budgets []models.Budget
	if err := query.
		Order("year DESC, month DESC, category_id ASC").
		Find(&budgets).Error; err != nil {
		return nil, fmt.Errorf("listando presupuestos: %w", err)
	}
	return budgets, nil
}

// FindByIDAndUser busca un presupuesto garantizando la pertenencia al usuario.
func (r *BudgetRepository) FindByIDAndUser(ctx context.Context, id, userID uint) (*models.Budget, error) {
	var budget models.Budget
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Where("id = ? AND user_id = ?", id, userID).
		First(&budget).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("buscando presupuesto: %w", err)
	}
	return &budget, nil
}

// Create inserta un presupuesto nuevo.
func (r *BudgetRepository) Create(ctx context.Context, budget *models.Budget) error {
	if err := r.db.WithContext(ctx).Create(budget).Error; err != nil {
		return fmt.Errorf("creando presupuesto: %w", err)
	}
	return nil
}

// Update guarda los campos modificables de un presupuesto.
func (r *BudgetRepository) Update(ctx context.Context, budget *models.Budget) error {
	result := r.db.WithContext(ctx).
		Model(&models.Budget{}).
		Where("id = ? AND user_id = ?", budget.ID, budget.UserID).
		Updates(map[string]any{
			"category_id":  budget.CategoryID,
			"month":        budget.Month,
			"year":         budget.Year,
			"limit_amount": budget.LimitAmount,
		})
	if result.Error != nil {
		return fmt.Errorf("actualizando presupuesto: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete elimina un presupuesto del usuario.
func (r *BudgetRepository) Delete(ctx context.Context, id, userID uint) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Budget{})
	if result.Error != nil {
		return fmt.Errorf("eliminando presupuesto: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ExistsByPeriod comprueba la unicidad de (usuario, categoría, mes, año).
func (r *BudgetRepository) ExistsByPeriod(ctx context.Context, userID, categoryID uint, month, year int, excludeID uint) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&models.Budget{}).
		Where("user_id = ? AND category_id = ? AND month = ? AND year = ?", userID, categoryID, month, year)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("comprobando presupuesto duplicado: %w", err)
	}
	return count > 0, nil
}

// SpentByPeriod agrega en SQL el gasto por categoría y periodo. Si month y year
// son cero se agrupan todos los periodos del usuario.
func (r *BudgetRepository) SpentByPeriod(ctx context.Context, userID uint, month, year int) (map[BudgetSpendKey]int64, error) {
	type row struct {
		Year       int
		Month      int
		CategoryID uint
		Total      int64
	}

	query := r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Select("CAST(substr(date, 1, 4) AS INTEGER) AS year, CAST(substr(date, 6, 2) AS INTEGER) AS month, category_id, COALESCE(SUM(amount), 0) AS total").
		Where("user_id = ? AND type = ?", userID, models.TypeExpense)

	if month != 0 && year != 0 {
		start, end := monthDateRange(month, year)
		query = query.Where("date >= ? AND date < ?", start, end)
	}

	var rows []row
	if err := query.Group("year, month, category_id").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("agregando gasto por periodo: %w", err)
	}

	spent := make(map[BudgetSpendKey]int64, len(rows))
	for _, item := range rows {
		spent[BudgetSpendKey{Year: item.Year, Month: item.Month, CategoryID: item.CategoryID}] = item.Total
	}
	return spent, nil
}

func monthDateRange(month, year int) (string, string) {
	start := fmt.Sprintf("%04d-%02d-01", year, month)
	if month == 12 {
		return start, fmt.Sprintf("%04d-01-01", year+1)
	}
	return start, fmt.Sprintf("%04d-%02d-01", year, month+1)
}
