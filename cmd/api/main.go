// Package main arranca la API de gestión de gastos personales.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xavie/gastos-api/internal/cache"
	"github.com/xavie/gastos-api/internal/config"
	"github.com/xavie/gastos-api/internal/database"
	"github.com/xavie/gastos-api/internal/handlers"
	"github.com/xavie/gastos-api/internal/repositories"
	"github.com/xavie/gastos-api/internal/router"
	"github.com/xavie/gastos-api/internal/services"
)

// @title                       Gastos API
// @version                     1.0
// @description                 API REST para la gestión de gastos personales: autenticación con JWT y refresh tokens, categorías, transacciones, presupuestos, reportes, caché Redis y exportación CSV. Los importes se expresan en centavos.
// @contact.name                Proyecto Gastos API
// @contact.url                 https://github.com/xavie/gastos-api
// @host                        localhost:8080
// @BasePath                    /
// @schemes                     http https
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Escribe "Bearer <access_token>" para probar las rutas protegidas.
func main() {
	if err := run(); err != nil {
		slog.Error("la aplicación terminó con error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	gin.SetMode(cfg.GinMode)

	db, err := database.Connect(cfg)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	cacheClient := cache.New(cfg, logger)
	defer cacheClient.Close()

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

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("servidor HTTP iniciado", "port", cfg.Port, "mode", cfg.GinMode)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("señal de apagado recibida")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("servidor detenido correctamente")
	return nil
}
