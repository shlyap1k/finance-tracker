package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"finance-tracker/internal/middleware"
	"finance-tracker/internal/service"
)

// SetupRouter настраивает Gin роутер со всеми маршрутами
func SetupRouter(projectionService *service.ProjectionService) *gin.Engine {
	r := gin.Default()

	// Публичные маршруты (без аутентификации)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Защищенные маршруты (требуют аутентификации)
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())

	// Projection endpoints
	projectionHandler := NewProjectionHandler(projectionService)

	projection := api.Group("/projection")
	{
		projection.GET("/balance", projectionHandler.GetBalanceProjection)
		projection.GET("/period-summary", projectionHandler.GetPeriodSummary)
	}

	// TODO: Добавить остальные группы endpoints
	// auth := api.Group("/auth")
	// incomeSources := api.Group("/income-sources")
	// expenseObligations := api.Group("/expense-obligations")
	// savingsBuckets := api.Group("/savings-buckets")
	// savingsRules := api.Group("/savings-rules")
	// balanceSnapshots := api.Group("/balance-snapshots")
	// transactions := api.Group("/transactions")

	return r
}
