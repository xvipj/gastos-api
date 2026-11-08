package services

import (
	"context"
	"errors"
	"fmt"
	"math"

	"gorm.io/gorm"

	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/models"
	"github.com/xavie/gastos-api/internal/repositories"
)

const (
	BudgetStatusOK       = "ok"
	BudgetStatusWarning  = "warning"
	BudgetStatusExceeded = "exceeded"
)

// BudgetService implementa la lógica de negocio de los presupuestos.
type BudgetService struct {
	budgets    *repositories.BudgetRepository
	categories *repositories.CategoryRepository
}

// NewBudgetService construye el servicio de presupuestos.
func NewBudgetService(
	budgets *repositories.BudgetRepository,
	categories *repositories.CategoryRepository,
) *BudgetService {
	return &BudgetService{budgets: budgets, categories: categories}
}

// List devuelve los presupuestos del usuario con su gasto y estado.
func (s *BudgetService) List(ctx context.Context, userID uint, filter dto.BudgetFilter) ([]dto.BudgetResponse, error) {
	if (filter.Month == 0) != (filter.Year == 0) {
		return nil, NewValidationError("month y year deben indicarse juntos")
	}

	budgets, err := s.budgets.ListByUser(ctx, userID, filter.Month, filter.Year)
	if err != nil {
		return nil, err
	}

	spent, err := s.budgets.SpentByPeriod(ctx, userID, filter.Month, filter.Year)
	if err != nil {
		return nil, err
	}

	response := make([]dto.BudgetResponse, 0, len(budgets))
	for i := range budgets {
		response = append(response, budgetResponse(&budgets[i], spent))
	}
	return response, nil
}

// Create registra un presupuesto para una categoría de gasto.
func (s *BudgetService) Create(ctx context.Context, userID uint, req dto.BudgetRequest) (*dto.BudgetResponse, error) {
	category, err := s.validateExpenseCategory(ctx, userID, req.CategoryID)
	if err != nil {
		return nil, err
	}

	exists, err := s.budgets.ExistsByPeriod(ctx, userID, req.CategoryID, req.Month, req.Year, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, NewConflictError("ya existe un presupuesto para esa categoría y periodo")
	}

	budget := &models.Budget{
		UserID:      userID,
		CategoryID:  req.CategoryID,
		Month:       req.Month,
		Year:        req.Year,
		LimitAmount: req.LimitAmount,
	}
	if err := s.budgets.Create(ctx, budget); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, NewConflictError("ya existe un presupuesto para esa categoría y periodo")
		}
		return nil, err
	}
	budget.Category = *category

	spent, err := s.budgets.SpentByPeriod(ctx, userID, budget.Month, budget.Year)
	if err != nil {
		return nil, err
	}
	response := budgetResponse(budget, spent)
	return &response, nil
}

// Update modifica un presupuesto existente del usuario.
func (s *BudgetService) Update(ctx context.Context, userID, budgetID uint, req dto.BudgetRequest) (*dto.BudgetResponse, error) {
	budget, err := s.budgets.FindByIDAndUser(ctx, budgetID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewNotFoundError("presupuesto no encontrado")
		}
		return nil, err
	}

	category, err := s.validateExpenseCategory(ctx, userID, req.CategoryID)
	if err != nil {
		return nil, err
	}

	exists, err := s.budgets.ExistsByPeriod(ctx, userID, req.CategoryID, req.Month, req.Year, budget.ID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, NewConflictError("ya existe un presupuesto para esa categoría y periodo")
	}

	budget.CategoryID = req.CategoryID
	budget.Month = req.Month
	budget.Year = req.Year
	budget.LimitAmount = req.LimitAmount
	budget.Category = *category

	if err := s.budgets.Update(ctx, budget); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewNotFoundError("presupuesto no encontrado")
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, NewConflictError("ya existe un presupuesto para esa categoría y periodo")
		}
		return nil, err
	}

	spent, err := s.budgets.SpentByPeriod(ctx, userID, budget.Month, budget.Year)
	if err != nil {
		return nil, err
	}
	response := budgetResponse(budget, spent)
	return &response, nil
}

// Delete elimina un presupuesto del usuario.
func (s *BudgetService) Delete(ctx context.Context, userID, budgetID uint) error {
	if err := s.budgets.Delete(ctx, budgetID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewNotFoundError("presupuesto no encontrado")
		}
		return fmt.Errorf("eliminando presupuesto: %w", err)
	}
	return nil
}

func (s *BudgetService) validateExpenseCategory(ctx context.Context, userID, categoryID uint) (*models.Category, error) {
	category, err := s.categories.FindByIDAndUser(ctx, categoryID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUnprocessableError("la categoría no existe o no pertenece al usuario")
		}
		return nil, err
	}
	if category.Type != models.TypeExpense {
		return nil, NewUnprocessableError("los presupuestos solo aplican a categorías de tipo expense")
	}
	return category, nil
}

func budgetResponse(budget *models.Budget, spentByPeriod map[repositories.BudgetSpendKey]int64) dto.BudgetResponse {
	key := repositories.BudgetSpendKey{
		Year:       budget.Year,
		Month:      budget.Month,
		CategoryID: budget.CategoryID,
	}
	spent := spentByPeriod[key]
	remaining := budget.LimitAmount - spent

	percentage := 0.0
	if budget.LimitAmount > 0 {
		percentage = math.Round(float64(spent)/float64(budget.LimitAmount)*10000) / 100
	}

	status := BudgetStatusOK
	switch {
	case percentage >= 100:
		status = BudgetStatusExceeded
	case percentage >= 80:
		status = BudgetStatusWarning
	}

	return dto.BudgetResponse{
		ID:             budget.ID,
		CategoryID:     budget.CategoryID,
		CategoryName:   budget.Category.Name,
		Month:          budget.Month,
		Year:           budget.Year,
		LimitAmount:    budget.LimitAmount,
		Spent:          spent,
		Remaining:      remaining,
		PercentageUsed: percentage,
		Status:         status,
		CreatedAt:      budget.CreatedAt,
	}
}
