// Package router define las rutas HTTP y conecta handlers y middleware.
package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/xavie/gastos-api/docs"
	"github.com/xavie/gastos-api/internal/cache"
	"github.com/xavie/gastos-api/internal/config"
	"github.com/xavie/gastos-api/internal/handlers"
	"github.com/xavie/gastos-api/internal/middleware"
	"github.com/xavie/gastos-api/internal/services"
)

// Deps agrupa las dependencias que necesita el router.
type Deps struct {
	Config             *config.Config
	Logger             *slog.Logger
	Cache              *cache.Client
	AuthService        *services.AuthService
	AuthHandler        *handlers.AuthHandler
	CategoryHandler    *handlers.CategoryHandler
	TransactionHandler *handlers.TransactionHandler
	BudgetHandler      *handlers.BudgetHandler
	ReportHandler      *handlers.ReportHandler
	HealthHandler      *handlers.HealthHandler
}

// New construye el engine de Gin con todas las rutas de la API.
func New(deps Deps) *gin.Engine {
	gin.SetMode(deps.Config.GinMode)

	engine := gin.New()
	engine.Use(
		middleware.Recovery(deps.Logger),
		middleware.Logger(deps.Logger),
		middleware.CORS(),
	)

	engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "NOT_FOUND", "message": "ruta no encontrada"},
		})
	})

	engine.GET("/health", deps.HealthHandler.Check)
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	rateLimiter := middleware.NewRateLimiter(deps.Cache.Redis(), deps.Logger, 5, time.Minute)

	v1 := engine.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		auth.POST("/register", rateLimiter.Limit("register"), deps.AuthHandler.Register)
		auth.POST("/login", rateLimiter.Limit("login"), deps.AuthHandler.Login)
		auth.POST("/refresh", deps.AuthHandler.Refresh)
		auth.POST("/logout", deps.AuthHandler.Logout)
	}

	protected := v1.Group("")
	protected.Use(middleware.Auth(deps.AuthService))
	{
		protected.GET("/categories", deps.CategoryHandler.List)
		protected.POST("/categories", deps.CategoryHandler.Create)
		protected.PUT("/categories/:id", deps.CategoryHandler.Update)
		protected.DELETE("/categories/:id", deps.CategoryHandler.Delete)

		protected.GET("/transactions/export", deps.TransactionHandler.Export)
		protected.GET("/transactions", deps.TransactionHandler.List)
		protected.POST("/transactions", deps.TransactionHandler.Create)
		protected.GET("/transactions/:id", deps.TransactionHandler.Get)
		protected.PUT("/transactions/:id", deps.TransactionHandler.Update)
		protected.DELETE("/transactions/:id", deps.TransactionHandler.Delete)

		protected.GET("/budgets", deps.BudgetHandler.List)
		protected.POST("/budgets", deps.BudgetHandler.Create)
		protected.PUT("/budgets/:id", deps.BudgetHandler.Update)
		protected.DELETE("/budgets/:id", deps.BudgetHandler.Delete)

		protected.GET("/reports/summary", deps.ReportHandler.Summary)
		protected.GET("/reports/trend", deps.ReportHandler.Trend)
	}

	return engine
}
