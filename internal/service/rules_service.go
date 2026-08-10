package service

import (
	"context"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"finance-tracker/internal/domain"
	"finance-tracker/internal/repository"
)

// RulesService управляет CRUD операциями для income sources, expense obligations, savings buckets и savings rules
type RulesService struct {
	repo *repository.Queries
}

// NewRulesService создает новый сервис правил
func NewRulesService(repo *repository.Queries) *RulesService {
	return &RulesService{repo: repo}
}

// ========== Income Sources ==========

// CreateIncomeSourceInput параметры для создания источника дохода
type CreateIncomeSourceInput struct {
	Name           string          `json:"name" binding:"required"`
	Amount         decimal.Decimal `json:"amount" binding:"required"`
	DayOfMonth     int             `json:"day_of_month" binding:"required,min=1,max=31"`
	OverflowPolicy string          `json:"overflow_policy" binding:"required,oneof=forward backward"`
}

// UpdateIncomeSourceInput параметры для обновления источника дохода
type UpdateIncomeSourceInput struct {
	Name           *string          `json:"name,omitempty"`
	Amount         *decimal.Decimal `json:"amount,omitempty"`
	DayOfMonth     *int             `json:"day_of_month,omitempty"`
	OverflowPolicy *string          `json:"overflow_policy,omitempty"`
}

func (s *RulesService) CreateIncomeSource(ctx context.Context, userID uuid.UUID, input CreateIncomeSourceInput) (domain.IncomeSource, error) {
	amount := decimalToPgNumeric(input.Amount)

	income, err := s.repo.CreateIncomeSource(ctx, repository.CreateIncomeSourceParams{
		UserID:         userID,
		Name:           input.Name,
		Amount:         amount,
		DayOfMonth:     int16(input.DayOfMonth),
		OverflowPolicy: input.OverflowPolicy,
	})
	if err != nil {
		return domain.IncomeSource{}, fmt.Errorf("failed to create income source: %w", err)
	}

	return mapIncomeSourceToDomain(income), nil
}

func (s *RulesService) GetIncomeSource(ctx context.Context, id uuid.UUID) (domain.IncomeSource, error) {
	income, err := s.repo.GetIncomeSourceByID(ctx, id)
	if err != nil {
		return domain.IncomeSource{}, fmt.Errorf("failed to get income source: %w", err)
	}

	return mapIncomeSourceToDomain(income), nil
}

func (s *RulesService) UpdateIncomeSource(ctx context.Context, userID uuid.UUID, id uuid.UUID, input UpdateIncomeSourceInput) (domain.IncomeSource, error) {
	params := repository.UpdateIncomeSourceParams{
		ID:             id,
		UserID:         userID,
		Name:           pgtype.Text{Valid: false},
		Amount:         pgtype.Numeric{Valid: false},
		DayOfMonth:     pgtype.Int2{Valid: false},
		OverflowPolicy: pgtype.Text{Valid: false},
	}

	if input.Name != nil {
		params.Name = pgtype.Text{String: *input.Name, Valid: true}
	}
	if input.Amount != nil {
		params.Amount = decimalToPgNumeric(*input.Amount)
	}
	if input.DayOfMonth != nil {
		params.DayOfMonth = pgtype.Int2{Int16: int16(*input.DayOfMonth), Valid: true}
	}
	if input.OverflowPolicy != nil {
		params.OverflowPolicy = pgtype.Text{String: *input.OverflowPolicy, Valid: true}
	}

	income, err := s.repo.UpdateIncomeSource(ctx, params)
	if err != nil {
		return domain.IncomeSource{}, fmt.Errorf("failed to update income source: %w", err)
	}

	return mapIncomeSourceToDomain(income), nil
}

func (s *RulesService) DeleteIncomeSource(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	err := s.repo.ArchiveIncomeSource(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete income source: %w", err)
	}
	return nil
}

func (s *RulesService) GetAllIncomeSources(ctx context.Context, userID uuid.UUID) ([]domain.IncomeSource, error) {
	incomes, err := s.repo.GetActiveIncomeSources(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get income sources: %w", err)
	}

	result := make([]domain.IncomeSource, 0, len(incomes))
	for _, inc := range incomes {
		result = append(result, mapGetActiveIncomeSourcesRowToDomain(inc))
	}
	return result, nil
}

// ========== Expense Obligations ==========

// CreateExpenseObligationInput параметры для создания обязательства по расходам
type CreateExpenseObligationInput struct {
	Name           string          `json:"name" binding:"required"`
	Amount         decimal.Decimal `json:"amount" binding:"required"`
	DayOfMonth     int             `json:"day_of_month" binding:"required,min=1,max=31"`
	OverflowPolicy string          `json:"overflow_policy" binding:"required,oneof=forward backward"`
}

// UpdateExpenseObligationInput параметры для обновления обязательства по расходам
type UpdateExpenseObligationInput struct {
	Name           *string          `json:"name,omitempty"`
	Amount         *decimal.Decimal `json:"amount,omitempty"`
	DayOfMonth     *int             `json:"day_of_month,omitempty"`
	OverflowPolicy *string          `json:"overflow_policy,omitempty"`
}

func (s *RulesService) CreateExpenseObligation(ctx context.Context, userID uuid.UUID, input CreateExpenseObligationInput) (domain.ExpenseObligation, error) {
	amount := decimalToPgNumeric(input.Amount)

	expense, err := s.repo.CreateExpenseObligation(ctx, repository.CreateExpenseObligationParams{
		UserID:         userID,
		Name:           input.Name,
		Amount:         amount,
		DayOfMonth:     int16(input.DayOfMonth),
		OverflowPolicy: input.OverflowPolicy,
	})
	if err != nil {
		return domain.ExpenseObligation{}, fmt.Errorf("failed to create expense obligation: %w", err)
	}

	return mapExpenseObligationToDomain(expense), nil
}

func (s *RulesService) GetExpenseObligation(ctx context.Context, id uuid.UUID) (domain.ExpenseObligation, error) {
	expense, err := s.repo.GetExpenseObligationByID(ctx, id)
	if err != nil {
		return domain.ExpenseObligation{}, fmt.Errorf("failed to get expense obligation: %w", err)
	}

	return mapExpenseObligationToDomain(expense), nil
}

func (s *RulesService) UpdateExpenseObligation(ctx context.Context, userID uuid.UUID, id uuid.UUID, input UpdateExpenseObligationInput) (domain.ExpenseObligation, error) {
	params := repository.UpdateExpenseObligationParams{
		ID:             id,
		UserID:         userID,
		Name:           pgtype.Text{Valid: false},
		Amount:         pgtype.Numeric{Valid: false},
		DayOfMonth:     pgtype.Int2{Valid: false},
		OverflowPolicy: pgtype.Text{Valid: false},
	}

	if input.Name != nil {
		params.Name = pgtype.Text{String: *input.Name, Valid: true}
	}
	if input.Amount != nil {
		params.Amount = decimalToPgNumeric(*input.Amount)
	}
	if input.DayOfMonth != nil {
		params.DayOfMonth = pgtype.Int2{Int16: int16(*input.DayOfMonth), Valid: true}
	}
	if input.OverflowPolicy != nil {
		params.OverflowPolicy = pgtype.Text{String: *input.OverflowPolicy, Valid: true}
	}

	expense, err := s.repo.UpdateExpenseObligation(ctx, params)
	if err != nil {
		return domain.ExpenseObligation{}, fmt.Errorf("failed to update expense obligation: %w", err)
	}

	return mapExpenseObligationToDomain(expense), nil
}

func (s *RulesService) DeleteExpenseObligation(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	err := s.repo.ArchiveExpenseObligation(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete expense obligation: %w", err)
	}
	return nil
}

func (s *RulesService) GetAllExpenseObligations(ctx context.Context, userID uuid.UUID) ([]domain.ExpenseObligation, error) {
	expenses, err := s.repo.GetActiveExpenseObligations(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get expense obligations: %w", err)
	}

	result := make([]domain.ExpenseObligation, 0, len(expenses))
	for _, exp := range expenses {
		result = append(result, mapGetActiveExpenseObligationsRowToDomain(exp))
	}
	return result, nil
}

// ========== Savings Buckets ==========

// CreateSavingsBucketInput параметры для создания накопительной цели
type CreateSavingsBucketInput struct {
	Name         string           `json:"name" binding:"required"`
	TargetAmount *decimal.Decimal `json:"target_amount,omitempty"`
}

// UpdateSavingsBucketInput параметры для обновления накопительной цели
type UpdateSavingsBucketInput struct {
	Name         *string          `json:"name,omitempty"`
	TargetAmount *decimal.Decimal `json:"target_amount,omitempty"`
}

func (s *RulesService) CreateSavingsBucket(ctx context.Context, userID uuid.UUID, input CreateSavingsBucketInput) (domain.SavingsBucket, error) {
	var targetAmount pgtype.Numeric
	if input.TargetAmount != nil {
		targetAmount = decimalToPgNumeric(*input.TargetAmount)
	} else {
		targetAmount = pgtype.Numeric{Valid: false}
	}

	bucket, err := s.repo.CreateSavingsBucket(ctx, repository.CreateSavingsBucketParams{
		UserID:       userID,
		Name:         input.Name,
		TargetAmount: targetAmount,
	})
	if err != nil {
		return domain.SavingsBucket{}, fmt.Errorf("failed to create savings bucket: %w", err)
	}

	return mapSavingsBucketToDomain(bucket), nil
}

func (s *RulesService) GetSavingsBucket(ctx context.Context, id uuid.UUID) (domain.SavingsBucket, error) {
	bucket, err := s.repo.GetSavingsBucketByID(ctx, id)
	if err != nil {
		return domain.SavingsBucket{}, fmt.Errorf("failed to get savings bucket: %w", err)
	}

	return mapSavingsBucketToDomain(bucket), nil
}

func (s *RulesService) UpdateSavingsBucket(ctx context.Context, userID uuid.UUID, id uuid.UUID, input UpdateSavingsBucketInput) (domain.SavingsBucket, error) {
	params := repository.UpdateSavingsBucketParams{
		ID:           id,
		UserID:       userID,
		Name:         pgtype.Text{Valid: false},
		TargetAmount: pgtype.Numeric{Valid: false},
	}

	if input.Name != nil {
		params.Name = pgtype.Text{String: *input.Name, Valid: true}
	}
	if input.TargetAmount != nil {
		params.TargetAmount = decimalToPgNumeric(*input.TargetAmount)
	}

	bucket, err := s.repo.UpdateSavingsBucket(ctx, params)
	if err != nil {
		return domain.SavingsBucket{}, fmt.Errorf("failed to update savings bucket: %w", err)
	}

	return mapSavingsBucketToDomain(bucket), nil
}

func (s *RulesService) DeleteSavingsBucket(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	err := s.repo.DeleteSavingsBucket(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete savings bucket: %w", err)
	}
	return nil
}

func (s *RulesService) GetAllSavingsBuckets(ctx context.Context, userID uuid.UUID) ([]domain.SavingsBucket, error) {
	buckets, err := s.repo.GetAllSavingsBucketsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get savings buckets: %w", err)
	}

	result := make([]domain.SavingsBucket, 0, len(buckets))
	for _, bucket := range buckets {
		result = append(result, mapSavingsBucketToDomain(bucket))
	}
	return result, nil
}

// ========== Savings Rules ==========

// CreateSavingsRuleInput параметры для создания правила накопления
type CreateSavingsRuleInput struct {
	BucketID       uuid.UUID       `json:"bucket_id" binding:"required"`
	Mode           string          `json:"mode" binding:"required,oneof=fixed percent"`
	Value          decimal.Decimal `json:"value" binding:"required"`
}

// UpdateSavingsRuleInput параметры для обновления правила накопления
type UpdateSavingsRuleInput struct {
	Mode  *string          `json:"mode,omitempty"`
	Value *decimal.Decimal `json:"value,omitempty"`
}

func (s *RulesService) CreateSavingsRule(ctx context.Context, incomeSourceID uuid.UUID, input CreateSavingsRuleInput) (domain.SavingsRule, error) {
	value := decimalToPgNumeric(input.Value)

	rule, err := s.repo.CreateSavingsRule(ctx, repository.CreateSavingsRuleParams{
		IncomeSourceID: incomeSourceID,
		BucketID:       input.BucketID,
		Mode:           input.Mode,
		Value:          value,
	})
	if err != nil {
		return domain.SavingsRule{}, fmt.Errorf("failed to create savings rule: %w", err)
	}

	return mapSavingsRuleToDomain(rule, ""), nil
}

func (s *RulesService) GetSavingsRule(ctx context.Context, id uuid.UUID) (domain.SavingsRule, error) {
	rule, err := s.repo.GetSavingsRuleByID(ctx, id)
	if err != nil {
		return domain.SavingsRule{}, fmt.Errorf("failed to get savings rule: %w", err)
	}

	return mapSavingsRuleToDomain(rule, ""), nil
}

func (s *RulesService) UpdateSavingsRule(ctx context.Context, id uuid.UUID, input UpdateSavingsRuleInput) (domain.SavingsRule, error) {
	params := repository.UpdateSavingsRuleParams{
		ID:    id,
		Mode:  pgtype.Text{Valid: false},
		Value: pgtype.Numeric{Valid: false},
	}

	if input.Mode != nil {
		params.Mode = pgtype.Text{String: *input.Mode, Valid: true}
	}
	if input.Value != nil {
		params.Value = decimalToPgNumeric(*input.Value)
	}

	rule, err := s.repo.UpdateSavingsRule(ctx, params)
	if err != nil {
		return domain.SavingsRule{}, fmt.Errorf("failed to update savings rule: %w", err)
	}

	return mapSavingsRuleToDomain(rule, ""), nil
}

func (s *RulesService) DeleteSavingsRule(ctx context.Context, id uuid.UUID) error {
	err := s.repo.DeleteSavingsRule(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete savings rule: %w", err)
	}
	return nil
}

func (s *RulesService) GetAllSavingsRules(ctx context.Context, userID uuid.UUID) ([]domain.SavingsRule, error) {
	rules, err := s.repo.GetAllSavingsRulesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get savings rules: %w", err)
	}

	result := make([]domain.SavingsRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, mapSavingsRuleToDomain(rule, ""))
	}
	return result, nil
}

// ========== Helper functions ==========

func decimalToPgNumeric(d decimal.Decimal) pgtype.Numeric {
	str := d.String()
	intVal, _ := new(big.Int).SetString(str, 10)
	return pgtype.Numeric{
		Int:   intVal,
		Exp:   0,
		Valid: true,
	}
}

func mapIncomeSourceToDomain(r repository.IncomeSource) domain.IncomeSource {
	amount, _ := pgNumericToDecimal(r.Amount)
	return domain.IncomeSource{
		ID:             r.ID,
		Name:           r.Name,
		Amount:         amount,
		DayOfMonth:     int(r.DayOfMonth),
		OverflowPolicy: domain.OverflowPolicy(r.OverflowPolicy),
	}
}

func mapGetActiveIncomeSourcesRowToDomain(r repository.GetActiveIncomeSourcesRow) domain.IncomeSource {
	amount, _ := pgNumericToDecimal(r.Amount)
	return domain.IncomeSource{
		ID:             r.ID,
		Name:           r.Name,
		Amount:         amount,
		DayOfMonth:     int(r.DayOfMonth),
		OverflowPolicy: domain.OverflowPolicy(r.OverflowPolicy),
	}
}

func mapExpenseObligationToDomain(r repository.ExpenseObligation) domain.ExpenseObligation {
	amount, _ := pgNumericToDecimal(r.Amount)
	return domain.ExpenseObligation{
		ID:             r.ID,
		Name:           r.Name,
		Amount:         amount,
		DayOfMonth:     int(r.DayOfMonth),
		OverflowPolicy: domain.OverflowPolicy(r.OverflowPolicy),
	}
}

func mapGetActiveExpenseObligationsRowToDomain(r repository.GetActiveExpenseObligationsRow) domain.ExpenseObligation {
	amount, _ := pgNumericToDecimal(r.Amount)
	return domain.ExpenseObligation{
		ID:             r.ID,
		Name:           r.Name,
		Amount:         amount,
		DayOfMonth:     int(r.DayOfMonth),
		OverflowPolicy: domain.OverflowPolicy(r.OverflowPolicy),
	}
}

func mapSavingsBucketToDomain(r repository.SavingsBucket) domain.SavingsBucket {
	targetAmount, _ := pgNumericToDecimal(r.TargetAmount)
	currentAmount, _ := pgNumericToDecimal(r.CurrentAmount)
	return domain.SavingsBucket{
		ID:            r.ID,
		UserID:        r.UserID,
		Name:          r.Name,
		TargetAmount:  targetAmount,
		CurrentAmount: currentAmount,
		CreatedAt:     r.CreatedAt,
	}
}

func mapSavingsRuleToDomain(r repository.SavingsRule, bucketName string) domain.SavingsRule {
	value, _ := pgNumericToDecimal(r.Value)
	return domain.SavingsRule{
		ID:             r.ID,
		IncomeSourceID: r.IncomeSourceID,
		BucketName:     bucketName,
		Mode:           domain.SavingsMode(r.Mode),
		Value:          value,
	}
}
