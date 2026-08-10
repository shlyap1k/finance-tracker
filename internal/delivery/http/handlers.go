package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

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

// RulesHandler обрабатывает HTTP-запросы для CRUD операций с правилами
type RulesHandler struct {
	rulesService *service.RulesService
}

// NewRulesHandler создает новый handler для правил
func NewRulesHandler(rulesService *service.RulesService) *RulesHandler {
	return &RulesHandler{rulesService: rulesService}
}

// ========== Income Sources Handlers ==========

// GetAllIncomeSources обрабатывает GET /api/income-sources
func (h *RulesHandler) GetAllIncomeSources(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	incomes, err := h.rulesService.GetAllIncomeSources(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get income sources"})
		return
	}

	c.JSON(http.StatusOK, incomes)
}

// CreateIncomeSource обрабатывает POST /api/income-sources
func (h *RulesHandler) CreateIncomeSource(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var input service.CreateIncomeSourceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	income, err := h.rulesService.CreateIncomeSource(c.Request.Context(), userID.(uuid.UUID), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create income source"})
		return
	}

	c.JSON(http.StatusCreated, income)
}

// GetIncomeSourceByID обрабатывает GET /api/income-sources/{id}
func (h *RulesHandler) GetIncomeSourceByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	income, err := h.rulesService.GetIncomeSource(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get income source"})
		return
	}

	c.JSON(http.StatusOK, income)
}

// UpdateIncomeSource обрабатывает PUT /api/income-sources/{id}
func (h *RulesHandler) UpdateIncomeSource(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	var input service.UpdateIncomeSourceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	income, err := h.rulesService.UpdateIncomeSource(c.Request.Context(), userID.(uuid.UUID), id, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update income source"})
		return
	}

	c.JSON(http.StatusOK, income)
}

// DeleteIncomeSource обрабатывает DELETE /api/income-sources/{id}
func (h *RulesHandler) DeleteIncomeSource(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	if err := h.rulesService.DeleteIncomeSource(c.Request.Context(), userID.(uuid.UUID), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete income source"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ========== Expense Obligations Handlers ==========

// GetAllExpenseObligations обрабатывает GET /api/expense-obligations
func (h *RulesHandler) GetAllExpenseObligations(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	expenses, err := h.rulesService.GetAllExpenseObligations(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get expense obligations"})
		return
	}

	c.JSON(http.StatusOK, expenses)
}

// CreateExpenseObligation обрабатывает POST /api/expense-obligations
func (h *RulesHandler) CreateExpenseObligation(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var input service.CreateExpenseObligationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	expense, err := h.rulesService.CreateExpenseObligation(c.Request.Context(), userID.(uuid.UUID), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create expense obligation"})
		return
	}

	c.JSON(http.StatusCreated, expense)
}

// GetExpenseObligationByID обрабатывает GET /api/expense-obligations/{id}
func (h *RulesHandler) GetExpenseObligationByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	expense, err := h.rulesService.GetExpenseObligation(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get expense obligation"})
		return
	}

	c.JSON(http.StatusOK, expense)
}

// UpdateExpenseObligation обрабатывает PUT /api/expense-obligations/{id}
func (h *RulesHandler) UpdateExpenseObligation(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	var input service.UpdateExpenseObligationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	expense, err := h.rulesService.UpdateExpenseObligation(c.Request.Context(), userID.(uuid.UUID), id, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update expense obligation"})
		return
	}

	c.JSON(http.StatusOK, expense)
}

// DeleteExpenseObligation обрабатывает DELETE /api/expense-obligations/{id}
func (h *RulesHandler) DeleteExpenseObligation(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	if err := h.rulesService.DeleteExpenseObligation(c.Request.Context(), userID.(uuid.UUID), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete expense obligation"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ========== Savings Buckets Handlers ==========

// GetAllSavingsBuckets обрабатывает GET /api/savings-buckets
func (h *RulesHandler) GetAllSavingsBuckets(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	buckets, err := h.rulesService.GetAllSavingsBuckets(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get savings buckets"})
		return
	}

	c.JSON(http.StatusOK, buckets)
}

// CreateSavingsBucket обрабатывает POST /api/savings-buckets
func (h *RulesHandler) CreateSavingsBucket(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var input service.CreateSavingsBucketInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	bucket, err := h.rulesService.CreateSavingsBucket(c.Request.Context(), userID.(uuid.UUID), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create savings bucket"})
		return
	}

	c.JSON(http.StatusCreated, bucket)
}

// GetSavingsBucketByID обрабатывает GET /api/savings-buckets/{id}
func (h *RulesHandler) GetSavingsBucketByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	bucket, err := h.rulesService.GetSavingsBucket(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get savings bucket"})
		return
	}

	c.JSON(http.StatusOK, bucket)
}

// UpdateSavingsBucket обрабатывает PUT /api/savings-buckets/{id}
func (h *RulesHandler) UpdateSavingsBucket(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	var input service.UpdateSavingsBucketInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	bucket, err := h.rulesService.UpdateSavingsBucket(c.Request.Context(), userID.(uuid.UUID), id, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update savings bucket"})
		return
	}

	c.JSON(http.StatusOK, bucket)
}

// DeleteSavingsBucket обрабатывает DELETE /api/savings-buckets/{id}
func (h *RulesHandler) DeleteSavingsBucket(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	if err := h.rulesService.DeleteSavingsBucket(c.Request.Context(), userID.(uuid.UUID), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete savings bucket"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ========== Savings Rules Handlers ==========

// GetAllSavingsRules обрабатывает GET /api/savings-rules
func (h *RulesHandler) GetAllSavingsRules(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	rules, err := h.rulesService.GetAllSavingsRules(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get savings rules"})
		return
	}

	c.JSON(http.StatusOK, rules)
}

// CreateSavingsRule обрабатывает POST /api/savings-rules
func (h *RulesHandler) CreateSavingsRule(c *gin.Context) {
	var input struct {
		IncomeSourceID string          `json:"income_source_id" binding:"required"`
		BucketID       string          `json:"bucket_id" binding:"required"`
		Mode           string          `json:"mode" binding:"required,oneof=fixed percent"`
		Value          decimal.Decimal `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	incomeSourceID, err := uuid.Parse(input.IncomeSourceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid income_source_id format"})
		return
	}

	bucketID, err := uuid.Parse(input.BucketID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bucket_id format"})
		return
	}

	ruleInput := service.CreateSavingsRuleInput{
		BucketID: bucketID,
		Mode:     input.Mode,
		Value:    input.Value,
	}

	rule, err := h.rulesService.CreateSavingsRule(c.Request.Context(), incomeSourceID, ruleInput)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create savings rule"})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// GetSavingsRuleByID обрабатывает GET /api/savings-rules/{id}
func (h *RulesHandler) GetSavingsRuleByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	rule, err := h.rulesService.GetSavingsRule(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get savings rule"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// UpdateSavingsRule обрабатывает PUT /api/savings-rules/{id}
func (h *RulesHandler) UpdateSavingsRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	var input service.UpdateSavingsRuleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}

	rule, err := h.rulesService.UpdateSavingsRule(c.Request.Context(), id, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update savings rule"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteSavingsRule обрабатывает DELETE /api/savings-rules/{id}
func (h *RulesHandler) DeleteSavingsRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	if err := h.rulesService.DeleteSavingsRule(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete savings rule"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Helper function to parse UUID from param
func parseUUIDParam(c *gin.Context, paramName string) (uuid.UUID, error) {
	idStr := c.Param(paramName)
	return uuid.Parse(idStr)
}

// Helper function to get userID from context
func getUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, false
	}
	return userID.(uuid.UUID), true
}

// Helper function to parse decimal from string
func parseDecimal(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}

// Helper function to parse int from string
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}
