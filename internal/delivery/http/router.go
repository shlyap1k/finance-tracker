package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"finance-tracker/internal/middleware"
	"finance-tracker/internal/service"
)

// SetupRouter настраивает Gin роутер со всеми маршрутами
func SetupRouter(projectionService *service.ProjectionService, authService *service.AuthService) *gin.Engine {
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

	return r
}
