package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"finance-tracker/internal/middleware"
	"finance-tracker/internal/service"
)

// SetupRouter настраивает Gin роутер со всеми маршрутами
func SetupRouter(projectionService *service.ProjectionService, authService *service.AuthService, rulesService *service.RulesService) *gin.Engine {
	r := gin.Default()

	// Публичные маршруты (без аутентификации)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth endpoints (публичные)
	authHandler := NewAuthHandler(authService)
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// Защищенные маршруты (требуют аутентификации)
	api := r.Group("/api")
	api.Use(middleware.JWTAuthMiddleware(authService))

	// Projection endpoints
	projectionHandler := NewProjectionHandler(projectionService)

	projection := api.Group("/projection")
	{
		projection.GET("/balance", projectionHandler.GetBalanceProjection)
		projection.GET("/period-summary", projectionHandler.GetPeriodSummary)
	}

	// Rules CRUD endpoints
	rulesHandler := NewRulesHandler(rulesService)

	// Income Sources
	incomeSources := api.Group("/income-sources")
	{
		incomeSources.GET("", rulesHandler.GetAllIncomeSources)
		incomeSources.POST("", rulesHandler.CreateIncomeSource)
		incomeSources.GET("/:id", rulesHandler.GetIncomeSourceByID)
		incomeSources.PUT("/:id", rulesHandler.UpdateIncomeSource)
		incomeSources.DELETE("/:id", rulesHandler.DeleteIncomeSource)
	}

	// Expense Obligations
	expenseObligations := api.Group("/expense-obligations")
	{
		expenseObligations.GET("", rulesHandler.GetAllExpenseObligations)
		expenseObligations.POST("", rulesHandler.CreateExpenseObligation)
		expenseObligations.GET("/:id", rulesHandler.GetExpenseObligationByID)
		expenseObligations.PUT("/:id", rulesHandler.UpdateExpenseObligation)
		expenseObligations.DELETE("/:id", rulesHandler.DeleteExpenseObligation)
	}

	// Savings Buckets
	savingsBuckets := api.Group("/savings-buckets")
	{
		savingsBuckets.GET("", rulesHandler.GetAllSavingsBuckets)
		savingsBuckets.POST("", rulesHandler.CreateSavingsBucket)
		savingsBuckets.GET("/:id", rulesHandler.GetSavingsBucketByID)
		savingsBuckets.PUT("/:id", rulesHandler.UpdateSavingsBucket)
		savingsBuckets.DELETE("/:id", rulesHandler.DeleteSavingsBucket)
	}

	// Savings Rules
	savingsRules := api.Group("/savings-rules")
	{
		savingsRules.GET("", rulesHandler.GetAllSavingsRules)
		savingsRules.POST("", rulesHandler.CreateSavingsRule)
		savingsRules.GET("/:id", rulesHandler.GetSavingsRuleByID)
		savingsRules.PUT("/:id", rulesHandler.UpdateSavingsRule)
		savingsRules.DELETE("/:id", rulesHandler.DeleteSavingsRule)
	}

	return r
}
