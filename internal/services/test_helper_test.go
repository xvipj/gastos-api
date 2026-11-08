package services_test

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xavie/gastos-api/internal/cache"
	"github.com/xavie/gastos-api/internal/config"
	"github.com/xavie/gastos-api/internal/database"
	"github.com/xavie/gastos-api/internal/models"
)

func newServiceTestEnv(t *testing.T) (*gorm.DB, *config.Config, *cache.Client) {
	t.Helper()

	cfg := &config.Config{
		Port:            "0",
		GinMode:         "test",
		DBPath:          ":memory:",
		JWTSecret:       "test-secret-con-longitud-suficiente",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		RedisEnabled:    false,
	}

	db, err := database.Connect(cfg)
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return db, cfg, cache.New(cfg, logger)
}

func createTestUser(t *testing.T, db *gorm.DB, email string) *models.User {
	t.Helper()

	user := &models.User{Name: "Usuario Test", Email: email, PasswordHash: "hash-de-prueba"}
	require.NoError(t, db.Create(user).Error)
	return user
}

func createTestCategory(t *testing.T, db *gorm.DB, userID uint, name, categoryType string) *models.Category {
	t.Helper()

	category := &models.Category{UserID: userID, Name: name, Type: categoryType}
	require.NoError(t, db.Create(category).Error)
	return category
}

func createTestTransaction(t *testing.T, db *gorm.DB, userID, categoryID uint, transactionType string, amount int64, date string) *models.Transaction {
	t.Helper()

	transaction := &models.Transaction{
		UserID:      userID,
		CategoryID:  categoryID,
		Type:        transactionType,
		Amount:      amount,
		Description: "prueba",
		Date:        date,
	}
	require.NoError(t, db.Create(transaction).Error)
	return transaction
}

func dateMonthsAgo(monthsAgo, day int) string {
	now := time.Now().UTC()
	date := time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, time.UTC).AddDate(0, -monthsAgo, 0)
	return date.Format("2006-01-02")
}

func monthMonthsAgo(monthsAgo int) string {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).
		AddDate(0, -monthsAgo, 0).
		Format("2006-01")
}
