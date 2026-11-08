package repositories

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/xavie/gastos-api/internal/models"
)

// CategoryTotal es el gasto agregado de una categoría.
type CategoryTotal struct {
	CategoryID   uint
	CategoryName string
	Total        int64
}

// MonthlyTotal es el agregado de ingresos o gastos de un mes.
type MonthlyTotal struct {
	Month string
	Type  string
	Total int64
}

// ReportRepository ejecuta las agregaciones SQL de los reportes.
type ReportRepository struct {
	db *gorm.DB
}

// NewReportRepository crea el repositorio de reportes.
func NewReportRepository(db *gorm.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

// Totals calcula ingresos y gastos de un rango de fechas en una sola consulta.
func (r *ReportRepository) Totals(ctx context.Context, userID uint, from, to string) (income, expense int64, err error) {
	type row struct {
		Type  string
		Total int64
	}

	var rows []row
	if err := r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Select("type, COALESCE(SUM(amount), 0) AS total").
		Where("user_id = ? AND date >= ? AND date < ?", userID, from, to).
		Group("type").
		Scan(&rows).Error; err != nil {
		return 0, 0, fmt.Errorf("agregando totales del periodo: %w", err)
	}

	for _, item := range rows {
		switch item.Type {
		case models.TypeIncome:
			income = item.Total
		case models.TypeExpense:
			expense = item.Total
		}
	}
	return income, expense, nil
}

// ExpensesByCategory agrega el gasto del periodo por categoría.
func (r *ReportRepository) ExpensesByCategory(ctx context.Context, userID uint, from, to string) ([]CategoryTotal, error) {
	var rows []CategoryTotal
	if err := r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Select("categories.id AS category_id, categories.name AS category_name, COALESCE(SUM(transactions.amount), 0) AS total").
		Joins("JOIN categories ON categories.id = transactions.category_id").
		Where("transactions.user_id = ? AND transactions.type = ? AND transactions.date >= ? AND transactions.date < ?",
			userID, models.TypeExpense, from, to).
		Group("categories.id, categories.name").
		Order("total DESC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("agregando gastos por categoría: %w", err)
	}
	return rows, nil
}

// Trend agrega ingresos y gastos por mes en el rango indicado.
func (r *ReportRepository) Trend(ctx context.Context, userID uint, from, to string) ([]MonthlyTotal, error) {
	var rows []MonthlyTotal
	if err := r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Select("substr(date, 1, 7) AS month, type, COALESCE(SUM(amount), 0) AS total").
		Where("user_id = ? AND date >= ? AND date < ?", userID, from, to).
		Group("month, type").
		Order("month ASC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("agregando tendencia mensual: %w", err)
	}
	return rows, nil
}
