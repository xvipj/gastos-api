package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/models"
	"github.com/xavie/gastos-api/internal/repositories"
)

// CategoryService implementa la lógica de negocio de las categorías.
type CategoryService struct {
	categories *repositories.CategoryRepository
}

// NewCategoryService construye el servicio de categorías.
func NewCategoryService(categories *repositories.CategoryRepository) *CategoryService {
	return &CategoryService{categories: categories}
}

// List devuelve todas las categorías del usuario.
func (s *CategoryService) List(ctx context.Context, userID uint) ([]dto.CategoryResponse, error) {
	categories, err := s.categories.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	response := make([]dto.CategoryResponse, 0, len(categories))
	for i := range categories {
		response = append(response, categoryResponse(&categories[i]))
	}
	return response, nil
}

// Create registra una categoría nueva para el usuario.
func (s *CategoryService) Create(ctx context.Context, userID uint, req dto.CategoryRequest) (*dto.CategoryResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, NewValidationError("el nombre de la categoría es obligatorio")
	}

	exists, err := s.categories.ExistsByNameAndType(ctx, userID, name, req.Type, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, NewConflictError("ya existe una categoría con ese nombre y tipo")
	}

	category := &models.Category{UserID: userID, Name: name, Type: req.Type}
	if err := s.categories.Create(ctx, category); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, NewConflictError("ya existe una categoría con ese nombre y tipo")
		}
		return nil, err
	}

	response := categoryResponse(category)
	return &response, nil
}

// Update modifica una categoría existente del usuario.
func (s *CategoryService) Update(ctx context.Context, userID, categoryID uint, req dto.CategoryRequest) (*dto.CategoryResponse, error) {
	category, err := s.categories.FindByIDAndUser(ctx, categoryID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewNotFoundError("categoría no encontrada")
		}
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, NewValidationError("el nombre de la categoría es obligatorio")
	}

	if req.Type != category.Type {
		count, err := s.categories.CountTransactions(ctx, category.ID)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, NewConflictError("no se puede cambiar el tipo de una categoría con transacciones asociadas")
		}
	}

	exists, err := s.categories.ExistsByNameAndType(ctx, userID, name, req.Type, category.ID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, NewConflictError("ya existe una categoría con ese nombre y tipo")
	}

	category.Name = name
	category.Type = req.Type
	if err := s.categories.Update(ctx, category); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, NewConflictError("ya existe una categoría con ese nombre y tipo")
		}
		return nil, err
	}

	response := categoryResponse(category)
	return &response, nil
}

// Delete elimina una categoría sin transacciones asociadas.
func (s *CategoryService) Delete(ctx context.Context, userID, categoryID uint) error {
	if _, err := s.categories.FindByIDAndUser(ctx, categoryID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewNotFoundError("categoría no encontrada")
		}
		return err
	}

	count, err := s.categories.CountTransactions(ctx, categoryID)
	if err != nil {
		return err
	}
	if count > 0 {
		return NewConflictError("la categoría tiene transacciones asociadas")
	}

	if err := s.categories.Delete(ctx, categoryID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewNotFoundError("categoría no encontrada")
		}
		return fmt.Errorf("eliminando categoría: %w", err)
	}
	return nil
}

func categoryResponse(category *models.Category) dto.CategoryResponse {
	return dto.CategoryResponse{
		ID:        category.ID,
		Name:      category.Name,
		Type:      category.Type,
		CreatedAt: category.CreatedAt,
	}
}
