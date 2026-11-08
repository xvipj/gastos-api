package handlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/models"
)

func TestTransactionLifecycleFiltersAndCSV(t *testing.T) {
	app := newTestApp(t)
	owner := app.register("Ana", "transactions@example.com", "super-secret-123")
	other := app.register("Luis", "other@example.com", "super-secret-123")

	food := app.createCategory(owner.AccessToken, "Alimentación", models.TypeExpense)
	salary := app.createCategory(owner.AccessToken, "Salario", models.TypeIncome)

	expense := app.createTransaction(owner.AccessToken, food.ID, models.TypeExpense, 125050, "2026-01-10")
	app.createTransaction(owner.AccessToken, salary.ID, models.TypeIncome, 2500000, "2026-02-01")
	require.Equal(t, "Alimentación", expense.CategoryName)
	require.Equal(t, int64(125050), expense.Amount)

	// El tipo debe coincidir con la categoría.
	recorder := app.do(http.MethodPost, "/api/v1/transactions", dto.TransactionRequest{
		CategoryID:  food.ID,
		Type:        models.TypeIncome,
		Amount:      1000,
		Description: "tipo incorrecto",
		Date:        "2026-01-12",
	}, owner.AccessToken)
	requireAPIError(t, recorder, http.StatusUnprocessableEntity, dto.CodeUnprocessable)

	// El monto debe ser mayor que cero.
	recorder = app.do(http.MethodPost, "/api/v1/transactions", dto.TransactionRequest{
		CategoryID:  food.ID,
		Type:        models.TypeExpense,
		Amount:      0,
		Description: "monto inválido",
		Date:        "2026-01-12",
	}, owner.AccessToken)
	requireAPIError(t, recorder, http.StatusBadRequest, dto.CodeValidation)

	// Un usuario no puede ver transacciones de otro.
	recorder = app.do(http.MethodGet, "/api/v1/transactions/"+itoa(expense.ID), nil, other.AccessToken)
	requireAPIError(t, recorder, http.StatusNotFound, dto.CodeNotFound)

	// Listado con paginación por defecto.
	recorder = app.do(http.MethodGet, "/api/v1/transactions", nil, owner.AccessToken)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	page := decodePage[dto.TransactionResponse](t, recorder)
	require.Len(t, page.Data, 2)
	require.Equal(t, 1, page.Meta.Page)
	require.Equal(t, 20, page.Meta.Limit)
	require.Equal(t, int64(2), page.Meta.Total)

	// Filtros combinados.
	recorder = app.do(http.MethodGet,
		"/api/v1/transactions?from=2026-01-01&to=2026-01-31&type=expense&min_amount=100000&max_amount=200000",
		nil, owner.AccessToken)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	filtered := decodePage[dto.TransactionResponse](t, recorder)
	require.Len(t, filtered.Data, 1)
	require.Equal(t, expense.ID, filtered.Data[0].ID)

	// Paginación explícita.
	recorder = app.do(http.MethodGet, "/api/v1/transactions?page=2&limit=1", nil, owner.AccessToken)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	paged := decodePage[dto.TransactionResponse](t, recorder)
	require.Len(t, paged.Data, 1)
	require.Equal(t, expense.ID, paged.Data[0].ID)
	require.Equal(t, int64(2), paged.Meta.Total)

	recorder = app.do(http.MethodGet, "/api/v1/transactions?limit=101", nil, owner.AccessToken)
	requireAPIError(t, recorder, http.StatusBadRequest, dto.CodeValidation)

	// No se puede borrar una categoría con transacciones.
	recorder = app.do(http.MethodDelete, "/api/v1/categories/"+itoa(food.ID), nil, owner.AccessToken)
	requireAPIError(t, recorder, http.StatusConflict, dto.CodeConflict)

	// Exportación CSV con filtros.
	recorder = app.do(http.MethodGet, "/api/v1/transactions/export?type=expense", nil, owner.AccessToken)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/csv")
	require.Contains(t, recorder.Header().Get("Content-Disposition"), "attachment")
	csvBody := recorder.Body.String()
	require.Contains(t, csvBody, "id,date,type,category,amount,description")
	require.Contains(t, csvBody, "1250.50")
	require.Contains(t, csvBody, "Alimentación")

	// Actualización.
	recorder = app.do(http.MethodPut, "/api/v1/transactions/"+itoa(expense.ID), dto.TransactionRequest{
		CategoryID:  food.ID,
		Type:        models.TypeExpense,
		Amount:      130000,
		Description: "compra actualizada",
		Date:        "2026-01-11",
	}, owner.AccessToken)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	updated := decodeData[dto.TransactionResponse](t, recorder)
	require.Equal(t, int64(130000), updated.Amount)
	require.Equal(t, "compra actualizada", updated.Description)

	// Eliminación y posterior borrado de la categoría.
	recorder = app.do(http.MethodDelete, "/api/v1/transactions/"+itoa(expense.ID), nil, owner.AccessToken)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	recorder = app.do(http.MethodDelete, "/api/v1/categories/"+itoa(food.ID), nil, owner.AccessToken)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	recorder = app.do(http.MethodGet, "/api/v1/transactions/"+itoa(expense.ID), nil, owner.AccessToken)
	requireAPIError(t, recorder, http.StatusNotFound, dto.CodeNotFound)
}

func TestTransactionCategoryFromAnotherUserIsRejected(t *testing.T) {
	app := newTestApp(t)
	owner := app.register("Ana", "owner@example.com", "super-secret-123")
	other := app.register("Luis", "intruder@example.com", "super-secret-123")

	category := app.createCategory(owner.AccessToken, "Transporte", models.TypeExpense)

	recorder := app.do(http.MethodPost, "/api/v1/transactions", dto.TransactionRequest{
		CategoryID:  category.ID,
		Type:        models.TypeExpense,
		Amount:      5000,
		Description: "ajena",
		Date:        "2026-03-01",
	}, other.AccessToken)
	requireAPIError(t, recorder, http.StatusUnprocessableEntity, dto.CodeUnprocessable)
}

func itoa(value uint) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	buffer := make([]byte, 0, 10)
	for value > 0 {
		buffer = append(buffer, digits[value%10])
		value /= 10
	}
	for left, right := 0, len(buffer)-1; left < right; left, right = left+1, right-1 {
		buffer[left], buffer[right] = buffer[right], buffer[left]
	}
	return string(buffer)
}
