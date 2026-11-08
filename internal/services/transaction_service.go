package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/xavie/gastos-api/internal/cache"
	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/models"
	"github.com/xavie/gastos-api/internal/repositories"
)

const dateLayout = "2006-01-02"

// TransactionService implementa la lógica de negocio de las transacciones.
type TransactionService struct {
	transactions *repositories.TransactionRepository
	categories   *repositories.CategoryRepository
	cache        *cache.Client
}

// NewTransactionService construye el servicio de transacciones.
func NewTransactionService(
	transactions *repositories.TransactionRepository,
	categories *repositories.CategoryRepository,
	cacheClient *cache.Client,
) *TransactionService {
	return &TransactionService{transactions: transactions, categories: categories, cache: cacheClient}
}

// List devuelve una página de transacciones aplicando los filtros recibidos.
func (s *TransactionService) List(ctx context.Context, userID uint, filter dto.TransactionFilter) ([]dto.TransactionResponse, int64, dto.TransactionFilter, error) {
	filter.Normalize()
	if err := validateTransactionFilter(filter); err != nil {
		return nil, 0, filter, err
	}

	transactions, total, err := s.transactions.List(ctx, toRepositoryFilters(userID, filter))
	if err != nil {
		return nil, 0, filter, err
	}

	return transactionResponses(transactions), total, filter, nil
}

// Export devuelve todas las transacciones que cumplen los filtros, sin paginar.
func (s *TransactionService) Export(ctx context.Context, userID uint, filter dto.TransactionFilter) ([]dto.TransactionResponse, error) {
	if err := validateTransactionFilter(filter); err != nil {
		return nil, err
	}

	transactions, err := s.transactions.ListAll(ctx, toRepositoryFilters(userID, filter))
	if err != nil {
		return nil, err
	}
	return transactionResponses(transactions), nil
}

// Get devuelve una transacción del usuario.
func (s *TransactionService) Get(ctx context.Context, userID, transactionID uint) (*dto.TransactionResponse, error) {
	transaction, err := s.transactions.FindByIDAndUser(ctx, transactionID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewNotFoundError("transacción no encontrada")
		}
		return nil, err
	}
	response := transactionResponse(transaction)
	return &response, nil
}

// Create valida y registra una transacción nueva.
func (s *TransactionService) Create(ctx context.Context, userID uint, req dto.TransactionRequest) (*dto.TransactionResponse, error) {
	category, err := s.validateRequest(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	transaction := &models.Transaction{
		UserID:      userID,
		CategoryID:  category.ID,
		Type:        req.Type,
		Amount:      req.Amount,
		Description: strings.TrimSpace(req.Description),
		Date:        req.Date,
	}
	if err := s.transactions.Create(ctx, transaction); err != nil {
		return nil, err
	}
	transaction.Category = *category
	invalidateReportCache(ctx, s.cache, userID)

	response := transactionResponse(transaction)
	return &response, nil
}

// Update modifica una transacción existente del usuario.
func (s *TransactionService) Update(ctx context.Context, userID, transactionID uint, req dto.TransactionRequest) (*dto.TransactionResponse, error) {
	transaction, err := s.transactions.FindByIDAndUser(ctx, transactionID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewNotFoundError("transacción no encontrada")
		}
		return nil, err
	}

	category, err := s.validateRequest(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	transaction.CategoryID = category.ID
	transaction.Type = req.Type
	transaction.Amount = req.Amount
	transaction.Description = strings.TrimSpace(req.Description)
	transaction.Date = req.Date

	if err := s.transactions.Update(ctx, transaction); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewNotFoundError("transacción no encontrada")
		}
		return nil, err
	}

	invalidateReportCache(ctx, s.cache, userID)

	updated, err := s.transactions.FindByIDAndUser(ctx, transactionID, userID)
	if err != nil {
		return nil, fmt.Errorf("recargando transacción actualizada: %w", err)
	}
	response := transactionResponse(updated)
	return &response, nil
}

// Delete elimina una transacción del usuario.
func (s *TransactionService) Delete(ctx context.Context, userID, transactionID uint) error {
	if err := s.transactions.Delete(ctx, transactionID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewNotFoundError("transacción no encontrada")
		}
		return err
	}
	invalidateReportCache(ctx, s.cache, userID)
	return nil
}

func (s *TransactionService) validateRequest(ctx context.Context, userID uint, req dto.TransactionRequest) (*models.Category, error) {
	if req.Amount <= 0 {
		return nil, NewUnprocessableError("el monto debe ser mayor que cero")
	}
	if _, err := time.Parse(dateLayout, req.Date); err != nil {
		return nil, NewValidationError("la fecha debe tener el formato YYYY-MM-DD")
	}

	category, err := s.categories.FindByIDAndUser(ctx, req.CategoryID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewUnprocessableError("la categoría no existe o no pertenece al usuario")
		}
		return nil, err
	}
	if category.Type != req.Type {
		return nil, NewUnprocessableError("el tipo de la transacción debe coincidir con el tipo de la categoría")
	}
	return category, nil
}

func validateTransactionFilter(filter dto.TransactionFilter) error {
	if filter.From != "" && filter.To != "" && filter.From > filter.To {
		return NewValidationError("el parámetro from no puede ser posterior a to")
	}
	if filter.MinAmount != nil && filter.MaxAmount != nil && *filter.MinAmount > *filter.MaxAmount {
		return NewValidationError("min_amount no puede ser mayor que max_amount")
	}
	return nil
}

func toRepositoryFilters(userID uint, filter dto.TransactionFilter) repositories.TransactionFilters {
	return repositories.TransactionFilters{
		UserID:     userID,
		From:       filter.From,
		To:         filter.To,
		CategoryID: filter.CategoryID,
		Type:       filter.Type,
		MinAmount:  filter.MinAmount,
		MaxAmount:  filter.MaxAmount,
		Page:       filter.Page,
		Limit:      filter.Limit,
	}
}

func transactionResponses(transactions []models.Transaction) []dto.TransactionResponse {
	response := make([]dto.TransactionResponse, 0, len(transactions))
	for i := range transactions {
		response = append(response, transactionResponse(&transactions[i]))
	}
	return response
}

func transactionResponse(transaction *models.Transaction) dto.TransactionResponse {
	return dto.TransactionResponse{
		ID:           transaction.ID,
		CategoryID:   transaction.CategoryID,
		CategoryName: transaction.Category.Name,
		Type:         transaction.Type,
		Amount:       transaction.Amount,
		Description:  transaction.Description,
		Date:         transaction.Date,
		CreatedAt:    transaction.CreatedAt,
		UpdatedAt:    transaction.UpdatedAt,
	}
}
