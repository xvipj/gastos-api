package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xavie/gastos-api/internal/cache"
	"github.com/xavie/gastos-api/internal/config"
	"github.com/xavie/gastos-api/internal/database"
	"github.com/xavie/gastos-api/internal/dto"
	"github.com/xavie/gastos-api/internal/handlers"
	"github.com/xavie/gastos-api/internal/repositories"
	"github.com/xavie/gastos-api/internal/router"
	"github.com/xavie/gastos-api/internal/services"
)

type testApp struct {
	t      *testing.T
	engine *gin.Engine
	db     *gorm.DB
}

type authData struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	TokenType    string           `json:"token_type"`
	ExpiresIn    int64            `json:"expires_in"`
	User         dto.UserResponse `json:"user"`
}

type pageData[T any] struct {
	Data []T      `json:"data"`
	Meta dto.Meta `json:"meta"`
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	cfg := &config.Config{
		Port:            "0",
		GinMode:         gin.TestMode,
		DBPath:          ":memory:",
		JWTSecret:       "test-secret-con-longitud-suficiente",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		RedisEnabled:    false,
		RedisAddr:       "localhost:6379",
	}

	db, err := database.Connect(cfg)
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cacheClient := cache.New(cfg, logger)

	userRepo := repositories.NewUserRepository(db)
	refreshTokenRepo := repositories.NewRefreshTokenRepository(db)
	categoryRepo := repositories.NewCategoryRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)
	budgetRepo := repositories.NewBudgetRepository(db)
	reportRepo := repositories.NewReportRepository(db)

	authService := services.NewAuthService(userRepo, refreshTokenRepo, cfg)
	categoryService := services.NewCategoryService(categoryRepo)
	transactionService := services.NewTransactionService(transactionRepo, categoryRepo, cacheClient)
	budgetService := services.NewBudgetService(budgetRepo, categoryRepo)
	reportService := services.NewReportService(reportRepo, cacheClient)

	authHandler := handlers.NewAuthHandler(authService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	transactionHandler := handlers.NewTransactionHandler(transactionService)
	budgetHandler := handlers.NewBudgetHandler(budgetService)
	reportHandler := handlers.NewReportHandler(reportService)
	healthHandler := handlers.NewHealthHandler(db, cacheClient)

	engine := router.New(router.Deps{
		Config:             cfg,
		Logger:             logger,
		Cache:              cacheClient,
		AuthService:        authService,
		AuthHandler:        authHandler,
		CategoryHandler:    categoryHandler,
		TransactionHandler: transactionHandler,
		BudgetHandler:      budgetHandler,
		ReportHandler:      reportHandler,
		HealthHandler:      healthHandler,
	})

	return &testApp{t: t, engine: engine, db: db}
}

func (a *testApp) do(method, path string, body any, token string) *httptest.ResponseRecorder {
	a.t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(a.t, err)
		reader = bytes.NewReader(raw)
	}

	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	a.engine.ServeHTTP(recorder, request)
	return recorder
}

func (a *testApp) register(name, email, password string) authData {
	a.t.Helper()

	recorder := a.do(http.MethodPost, "/api/v1/auth/register", dto.RegisterRequest{
		Name:     name,
		Email:    email,
		Password: password,
	}, "")
	require.Equal(a.t, http.StatusCreated, recorder.Code, recorder.Body.String())

	return decodeData[authData](a.t, recorder)
}

func (a *testApp) createCategory(token, name, categoryType string) dto.CategoryResponse {
	a.t.Helper()

	recorder := a.do(http.MethodPost, "/api/v1/categories", dto.CategoryRequest{
		Name: name,
		Type: categoryType,
	}, token)
	require.Equal(a.t, http.StatusCreated, recorder.Code, recorder.Body.String())
	return decodeData[dto.CategoryResponse](a.t, recorder)
}

func (a *testApp) createTransaction(token string, categoryID uint, transactionType string, amount int64, date string) dto.TransactionResponse {
	a.t.Helper()

	recorder := a.do(http.MethodPost, "/api/v1/transactions", dto.TransactionRequest{
		CategoryID:  categoryID,
		Type:        transactionType,
		Amount:      amount,
		Description: "transacción de prueba",
		Date:        date,
	}, token)
	require.Equal(a.t, http.StatusCreated, recorder.Code, recorder.Body.String())
	return decodeData[dto.TransactionResponse](a.t, recorder)
}

func decodeData[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()

	var envelope struct {
		Data T `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope), recorder.Body.String())
	return envelope.Data
}

func decodePage[T any](t *testing.T, recorder *httptest.ResponseRecorder) pageData[T] {
	t.Helper()

	var envelope pageData[T]
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope), recorder.Body.String())
	return envelope
}

func requireAPIError(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()

	require.Equal(t, status, recorder.Code, recorder.Body.String())
	var envelope dto.ErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope), recorder.Body.String())
	require.Equal(t, code, envelope.Error.Code)
	require.NotEmpty(t, envelope.Error.Message)
}
