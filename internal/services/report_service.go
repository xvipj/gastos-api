package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/xavie/gastos-api/internal/cache"
	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/models"
	"github.com/xavie/gastos-api/internal/repositories"
)

const (
	defaultTrendMonths = 6
	maxTrendMonths     = 24
	reportCacheTTL     = 10 * time.Minute
)

// ReportService calcula los reportes financieros a partir de agregaciones SQL.
type ReportService struct {
	reports *repositories.ReportRepository
	cache   *cache.Client
}

// NewReportService construye el servicio de reportes.
func NewReportService(reports *repositories.ReportRepository, cacheClient *cache.Client) *ReportService {
	return &ReportService{reports: reports, cache: cacheClient}
}

// Summary devuelve los totales y el desglose de gastos de un mes.
func (s *ReportService) Summary(ctx context.Context, userID uint, month, year int) (*dto.SummaryResponse, error) {
	if month < 1 || month > 12 || year < 2000 || year > 2100 {
		return nil, NewValidationError("month debe estar entre 1 y 12, y year entre 2000 y 2100")
	}

	cacheKey := fmt.Sprintf("summary:%d:%04d-%02d", userID, year, month)
	var cached dto.SummaryResponse
	if s.cache.GetJSON(ctx, cacheKey, &cached) {
		return &cached, nil
	}

	from, to := monthBounds(month, year)
	totalIncome, totalExpense, err := s.reports.Totals(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}

	rows, err := s.reports.ExpensesByCategory(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}

	byCategory := make([]dto.CategoryExpense, 0, len(rows))
	for _, row := range rows {
		percentage := 0.0
		if totalExpense > 0 {
			percentage = math.Round(float64(row.Total)/float64(totalExpense)*10000) / 100
		}
		byCategory = append(byCategory, dto.CategoryExpense{
			CategoryID:   row.CategoryID,
			CategoryName: row.CategoryName,
			Amount:       row.Total,
			Percentage:   percentage,
		})
	}

	response := &dto.SummaryResponse{
		Month:        month,
		Year:         year,
		TotalIncome:  totalIncome,
		TotalExpense: totalExpense,
		Balance:      totalIncome - totalExpense,
		ByCategory:   byCategory,
	}
	s.cache.SetJSON(ctx, cacheKey, response, reportCacheTTL)
	return response, nil
}

// Trend devuelve ingresos, gastos y balance de los últimos N meses.
func (s *ReportService) Trend(ctx context.Context, userID uint, months int) (*dto.TrendResponse, error) {
	if months == 0 {
		months = defaultTrendMonths
	}
	if months < 1 || months > maxTrendMonths {
		return nil, NewValidationError(fmt.Sprintf("months debe estar entre 1 y %d", maxTrendMonths))
	}

	cacheKey := fmt.Sprintf("trend:%d:%d", userID, months)
	var cached dto.TrendResponse
	if s.cache.GetJSON(ctx, cacheKey, &cached) {
		return &cached, nil
	}

	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	start := currentMonth.AddDate(0, -(months - 1), 0)
	end := currentMonth.AddDate(0, 1, 0)

	rows, err := s.reports.Trend(ctx, userID, start.Format(dateLayout), end.Format(dateLayout))
	if err != nil {
		return nil, err
	}

	incomeByMonth := make(map[string]int64)
	expenseByMonth := make(map[string]int64)
	for _, row := range rows {
		switch row.Type {
		case models.TypeIncome:
			incomeByMonth[row.Month] = row.Total
		case models.TypeExpense:
			expenseByMonth[row.Month] = row.Total
		}
	}

	points := make([]dto.TrendPoint, 0, months)
	for i := 0; i < months; i++ {
		month := start.AddDate(0, i, 0).Format("2006-01")
		income := incomeByMonth[month]
		expense := expenseByMonth[month]
		points = append(points, dto.TrendPoint{
			Month:   month,
			Income:  income,
			Expense: expense,
			Balance: income - expense,
		})
	}

	response := &dto.TrendResponse{Months: months, Trend: points}
	s.cache.SetJSON(ctx, cacheKey, response, reportCacheTTL)
	return response, nil
}

// invalidateReportCache elimina las claves de reportes del usuario. Se usa
// context.WithoutCancel para que la invalidación no se pierda si el cliente
// corta la conexión después de una escritura.
func invalidateReportCache(ctx context.Context, cacheClient *cache.Client, userID uint) {
	ctx = context.WithoutCancel(ctx)
	cacheClient.DeleteByPattern(ctx, fmt.Sprintf("summary:%d:*", userID))
	cacheClient.DeleteByPattern(ctx, fmt.Sprintf("trend:%d:*", userID))
}

func monthBounds(month, year int) (string, string) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	return start.Format(dateLayout), start.AddDate(0, 1, 0).Format(dateLayout)
}
