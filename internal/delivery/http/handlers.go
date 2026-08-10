package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"finance-tracker/internal/service"
)

// AuthHandler обрабатывает HTTP-запросы для аутентификации
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler создает новый handler для аутентификации
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register обрабатывает POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var input service.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	tokens, err := h.authService.Register(c.Request.Context(), input)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, tokens)
}

// Login обрабатывает POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var input service.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	tokens, err := h.authService.Login(c.Request.Context(), input)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to login"})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// ProjectionHandler обрабатывает HTTP-запросы для прогноза баланса
type ProjectionHandler struct {
	projectionService *service.ProjectionService
}

// NewProjectionHandler создает новый handler
func NewProjectionHandler(projectionService *service.ProjectionService) *ProjectionHandler {
	return &ProjectionHandler{projectionService: projectionService}
}

// GetBalanceProjection обрабатывает GET /api/projection/balance?date=YYYY-MM-DD
func (h *ProjectionHandler) GetBalanceProjection(c *gin.Context) {
	// 1. Получаем userID из контекста (установлен middleware аутентификации)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// 2. Парсим query параметр date
	dateStr := c.Query("date")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date parameter is required"})
		return
	}

	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use YYYY-MM-DD"})
		return
	}

	// 3. Вызываем сервис
	projection, err := h.projectionService.GetBalanceProjection(c.Request.Context(), userID.(uuid.UUID), targetDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate projection"})
		return
	}

	// 4. Возвращаем результат
	c.JSON(http.StatusOK, gin.H{
		"target_date":       projection.TargetDate.Format("2006-01-02"),
		"starting_balance":  projection.StartingBalance.String(),
		"fact_change":       projection.FactChange.String(),
		"future_change":     projection.FutureChange.String(),
		"final_balance":     projection.FinalBalance.String(),
		"projected_income":  projection.ProjectedIncome.String(),
		"projected_expense": projection.ProjectedExpense.String(),
		"projected_saving":  projection.ProjectedSaving.String(),
	})
}

// GetPeriodSummary обрабатывает GET /api/projection/period-summary?from=YYYY-MM-DD&to=YYYY-MM-DD
func (h *ProjectionHandler) GetPeriodSummary(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	fromStr := c.Query("from")
	toStr := c.Query("to")

	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to parameters are required"})
		return
	}

	fromDate, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from date format"})
		return
	}

	toDate, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to date format"})
		return
	}

	// Для period-summary мы просто вызываем тот же сервис, но с целевой датой = to
	projection, err := h.projectionService.GetBalanceProjection(c.Request.Context(), userID.(uuid.UUID), toDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate period summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"from":             fromDate.Format("2006-01-02"),
		"to":               toDate.Format("2006-01-02"),
		"total_income":     projection.ProjectedIncome.String(),
		"total_expense":    projection.ProjectedExpense.String(),
		"total_saving":     projection.ProjectedSaving.String(),
		"free_income":      projection.ProjectedIncome.Sub(projection.ProjectedExpense).Sub(projection.ProjectedSaving).String(),
		"starting_balance": projection.StartingBalance.String(),
		"final_balance":    projection.FinalBalance.String(),
	})
}
