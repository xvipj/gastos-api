package services_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xavie/gastos-api/internal/models"
	"github.com/xavie/gastos-api/internal/repositories"
	"github.com/xavie/gastos-api/internal/services"
)

func TestReportServiceSummaryAggregatesByCategory(t *testing.T) {
	db, _, cacheClient := newServiceTestEnv(t)
	reportService := services.NewReportService(repositories.NewReportRepository(db), cacheClient)

	user := createTestUser(t, db, "summary@example.com")
	salary := createTestCategory(t, db, user.ID, "Salario", models.TypeIncome)
	food := createTestCategory(t, db, user.ID, "Alimentación", models.TypeExpense)
	transport := createTestCategory(t, db, user.ID, "Transporte", models.TypeExpense)

	createTestTransaction(t, db, user.ID, salary.ID, models.TypeIncome, 2500000, "2026-11-01")
	createTestTransaction(t, db, user.ID, food.ID, models.TypeExpense, 1000500, "2026-11-05")
	createTestTransaction(t, db, user.ID, food.ID, models.TypeExpense, 250000, "2026-11-20")
	createTestTransaction(t, db, user.ID, transport.ID, models.TypeExpense, 500000, "2026-11-07")

	summary, err := reportService.Summary(context.Background(), user.ID, 11, 2026)
	require.NoError(t, err)
	require.Equal(t, int64(2500000), summary.TotalIncome)
	require.Equal(t, int64(1750500), summary.TotalExpense)
	require.Equal(t, int64(749500), summary.Balance)
	require.Len(t, summary.ByCategory, 2)

	require.Equal(t, food.ID, summary.ByCategory[0].CategoryID)
	require.Equal(t, int64(1250500), summary.ByCategory[0].Amount)
	require.InDelta(t, 71.44, summary.ByCategory[0].Percentage, 0.01)
	require.Equal(t, transport.ID, summary.ByCategory[1].CategoryID)
	require.Equal(t, int64(500000), summary.ByCategory[1].Amount)
	require.InDelta(t, 28.56, summary.ByCategory[1].Percentage, 0.01)

	// La agregación respeta el mes solicitado.
	empty, err := reportService.Summary(context.Background(), user.ID, 10, 2026)
	require.NoError(t, err)
	require.Zero(t, empty.TotalIncome)
	require.Zero(t, empty.TotalExpense)
	require.NotNil(t, empty.ByCategory)
	require.Empty(t, empty.ByCategory)

	_, err = reportService.Summary(context.Background(), user.ID, 13, 2026)
	require.ErrorIs(t, err, services.ErrValidation)
}

func TestReportServiceTrendFillsMonths(t *testing.T) {
	db, _, cacheClient := newServiceTestEnv(t)
	reportService := services.NewReportService(repositories.NewReportRepository(db), cacheClient)

	user := createTestUser(t, db, "trend@example.com")
	salary := createTestCategory(t, db, user.ID, "Salario", models.TypeIncome)
	food := createTestCategory(t, db, user.ID, "Alimentación", models.TypeExpense)

	createTestTransaction(t, db, user.ID, salary.ID, models.TypeIncome, 100000, dateMonthsAgo(0, 10))
	createTestTransaction(t, db, user.ID, food.ID, models.TypeExpense, 40000, dateMonthsAgo(0, 12))
	createTestTransaction(t, db, user.ID, salary.ID, models.TypeIncome, 80000, dateMonthsAgo(1, 10))
	createTestTransaction(t, db, user.ID, food.ID, models.TypeExpense, 20000, dateMonthsAgo(1, 12))
	createTestTransaction(t, db, user.ID, food.ID, models.TypeExpense, 10000, dateMonthsAgo(2, 12))

	trend, err := reportService.Trend(context.Background(), user.ID, 3)
	require.NoError(t, err)
	require.Equal(t, 3, trend.Months)
	require.Len(t, trend.Trend, 3)

	require.Equal(t, monthMonthsAgo(2), trend.Trend[0].Month)
	require.Equal(t, int64(0), trend.Trend[0].Income)
	require.Equal(t, int64(10000), trend.Trend[0].Expense)
	require.Equal(t, int64(-10000), trend.Trend[0].Balance)

	require.Equal(t, monthMonthsAgo(1), trend.Trend[1].Month)
	require.Equal(t, int64(80000), trend.Trend[1].Income)
	require.Equal(t, int64(20000), trend.Trend[1].Expense)
	require.Equal(t, int64(60000), trend.Trend[1].Balance)

	require.Equal(t, monthMonthsAgo(0), trend.Trend[2].Month)
	require.Equal(t, int64(100000), trend.Trend[2].Income)
	require.Equal(t, int64(40000), trend.Trend[2].Expense)
	require.Equal(t, int64(60000), trend.Trend[2].Balance)

	defaultTrend, err := reportService.Trend(context.Background(), user.ID, 0)
	require.NoError(t, err)
	require.Equal(t, 6, defaultTrend.Months)

	_, err = reportService.Trend(context.Background(), user.ID, 25)
	require.ErrorIs(t, err, services.ErrValidation)
}
