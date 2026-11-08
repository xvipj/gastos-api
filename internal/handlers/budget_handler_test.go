package handlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/models"
)

func TestBudgetSpentStatusAndLifecycle(t *testing.T) {
	app := newTestApp(t)
	user := app.register("Ana", "budgets@example.com", "super-secret-123")
	food := app.createCategory(user.AccessToken, "Alimentación", models.TypeExpense)
	salary := app.createCategory(user.AccessToken, "Salario", models.TypeIncome)

	// Presupuesto de 1.000,00 para noviembre de 2026.
	recorder := app.do(http.MethodPost, "/api/v1/budgets", dto.BudgetRequest{
		CategoryID:  food.ID,
		Month:       11,
		Year:        2026,
		LimitAmount: 100000,
	}, user.AccessToken)
	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
	budget := decodeData[dto.BudgetResponse](t, recorder)
	require.Equal(t, int64(0), budget.Spent)
	require.Equal(t, "ok", budget.Status)

	app.createTransaction(user.AccessToken, food.ID, models.TypeExpense, 85000, "2026-11-05")

	recorder = app.do(http.MethodGet, "/api/v1/budgets?month=11&year=2026", nil, user.AccessToken)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	budgets := decodeData[[]dto.BudgetResponse](t, recorder)
	require.Len(t, budgets, 1)
	require.Equal(t, int64(85000), budgets[0].Spent)
	require.Equal(t, int64(15000), budgets[0].Remaining)
	require.InDelta(t, 85.0, budgets[0].PercentageUsed, 0.01)
	require.Equal(t, "warning", budgets[0].Status)

	app.createTransaction(user.AccessToken, food.ID, models.TypeExpense, 20000, "2026-11-20")

	recorder = app.do(http.MethodGet, "/api/v1/budgets", nil, user.AccessToken)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	budgets = decodeData[[]dto.BudgetResponse](t, recorder)
	require.Len(t, budgets, 1)
	require.Equal(t, int64(105000), budgets[0].Spent)
	require.Equal(t, int64(-5000), budgets[0].Remaining)
	require.InDelta(t, 105.0, budgets[0].PercentageUsed, 0.01)
	require.Equal(t, "exceeded", budgets[0].Status)

	// Actualizar el límite devuelve el estado a ok.
	recorder = app.do(http.MethodPut, "/api/v1/budgets/"+itoa(budget.ID), dto.BudgetRequest{
		CategoryID:  food.ID,
		Month:       11,
		Year:        2026,
		LimitAmount: 200000,
	}, user.AccessToken)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	updated := decodeData[dto.BudgetResponse](t, recorder)
	require.InDelta(t, 52.5, updated.PercentageUsed, 0.01)
	require.Equal(t, "ok", updated.Status)

	// No se permiten presupuestos duplicados para la misma categoría y periodo.
	recorder = app.do(http.MethodPost, "/api/v1/budgets", dto.BudgetRequest{
		CategoryID:  food.ID,
		Month:       11,
		Year:        2026,
		LimitAmount: 300000,
	}, user.AccessToken)
	requireAPIError(t, recorder, http.StatusConflict, dto.CodeConflict)

	// Los presupuestos solo aplican a categorías expense.
	recorder = app.do(http.MethodPost, "/api/v1/budgets", dto.BudgetRequest{
		CategoryID:  salary.ID,
		Month:       11,
		Year:        2026,
		LimitAmount: 300000,
	}, user.AccessToken)
	requireAPIError(t, recorder, http.StatusUnprocessableEntity, dto.CodeUnprocessable)

	// Filtro parcial inválido.
	recorder = app.do(http.MethodGet, "/api/v1/budgets?month=11", nil, user.AccessToken)
	requireAPIError(t, recorder, http.StatusBadRequest, dto.CodeValidation)

	recorder = app.do(http.MethodDelete, "/api/v1/budgets/"+itoa(budget.ID), nil, user.AccessToken)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	recorder = app.do(http.MethodGet, "/api/v1/budgets", nil, user.AccessToken)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Empty(t, decodeData[[]dto.BudgetResponse](t, recorder))
}
