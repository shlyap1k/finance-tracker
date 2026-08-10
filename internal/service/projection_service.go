package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"finance-tracker/internal/domain"
	"finance-tracker/internal/repository"
)

func pgTextToString(t pgtype.Text) string {
	if !t.Valid {
		return "" // или возвращайте пустую строку для NULL
	}
	return t.String
}

func pgDateToTime(date pgtype.Date) time.Time {
	if !date.Valid {
		return time.Time{} // возвращаем zero time для NULL
	}
	return date.Time
}

func pgNumericToDecimal(n pgtype.Numeric) (decimal.Decimal, error) {
	if !n.Valid {
		return decimal.Decimal{}, fmt.Errorf("pgtype.Numeric is NULL")
	}

	// Value() возвращает строковое представление числа
	val, err := n.Value()
	if err != nil {
		return decimal.Decimal{}, err
	}

	str, ok := val.(string)
	if !ok {
		return decimal.Decimal{}, fmt.Errorf("unexpected type from Value(): %T", val)
	}

	return decimal.NewFromString(str)
}

// ProjectionRepository определяет, какие данные нам нужны из БД для расчета прогноза
// Это интерфейс, чтобы можно было легко мокировать в тестах
type ProjectionRepository interface {
	GetLatestSnapshotAtOrBefore(ctx context.Context, params repository.GetLatestSnapshotAtOrBeforeParams) (repository.BalanceSnapshot, error)
	GetConfirmedTransactionsInRange(ctx context.Context, params repository.GetConfirmedTransactionsInRangeParams) ([]repository.Transaction, error)
	GetActiveIncomeSources(ctx context.Context, userID uuid.UUID) ([]repository.GetActiveIncomeSourcesRow, error)
	GetActiveExpenseObligations(ctx context.Context, userID uuid.UUID) ([]repository.GetActiveExpenseObligationsRow, error)
	GetActiveSavingsRulesForUser(ctx context.Context, userID uuid.UUID) ([]repository.GetActiveSavingsRulesForUserRow, error)
}

// ProjectionService координирует расчет баланса
type ProjectionService struct {
	repo ProjectionRepository
}

// NewProjectionService создает новый экземпляр сервиса
func NewProjectionService(repo ProjectionRepository) *ProjectionService {
	return &ProjectionService{repo: repo}
}

// GetBalanceProjection рассчитывает прогноз баланса на целевую дату
func (s *ProjectionService) GetBalanceProjection(
	ctx context.Context,
	userID uuid.UUID,
	targetDate time.Time,
) (domain.BalanceProjection, error) {

	today := time.Now().UTC().Truncate(24 * time.Hour)
	targetDate = targetDate.UTC().Truncate(24 * time.Hour)

	// 1. Получаем последний снапшот на или до целевой даты (или сегодня)
	snapshotDate := targetDate
	if today.Before(targetDate) {
		snapshotDate = today
	}

	snapshotDatePg := pgtype.Date{
		Time:  snapshotDate,
		Valid: true,
	}
	paramsSnapshot := repository.GetLatestSnapshotAtOrBeforeParams{
		UserID:   userID,
		AsOfDate: snapshotDatePg,
	}

	snapshot, err := s.repo.GetLatestSnapshotAtOrBefore(ctx, paramsSnapshot)
	if err != nil {
		return domain.BalanceProjection{}, fmt.Errorf("failed to get snapshot: %w", err)
	}

	// 2. Определяем диапазон для фактических транзакций: от даты снапшота до min(today, target_date)
	factEndDate := targetDate
	if today.Before(targetDate) {
		factEndDate = today
	}

	// Конвертируем time.Time в pgtype.Date
	snapshotTxnDatePg := pgtype.Date{
		Time:  pgDateToTime(snapshot.AsOfDate),
		Valid: true,
	}
	factEndDatePg := pgtype.Date{
		Time:  factEndDate,
		Valid: true,
	}

	params := repository.GetConfirmedTransactionsInRangeParams{
		UserID:    userID,
		TxnDate:   snapshotTxnDatePg,
		TxnDate_2: factEndDatePg,
	}
	confirmedTxns, err := s.repo.GetConfirmedTransactionsInRange(ctx, params)
	if err != nil {
		return domain.BalanceProjection{}, fmt.Errorf("failed to get transactions: %w", err)
	}

	// 3. Получаем все активные правила пользователя
	incomes, err := s.repo.GetActiveIncomeSources(ctx, userID)
	if err != nil {
		return domain.BalanceProjection{}, fmt.Errorf("failed to get income sources: %w", err)
	}

	expenses, err := s.repo.GetActiveExpenseObligations(ctx, userID)
	if err != nil {
		return domain.BalanceProjection{}, fmt.Errorf("failed to get expense obligations: %w", err)
	}

	savingsRows, err := s.repo.GetActiveSavingsRulesForUser(ctx, userID)
	if err != nil {
		return domain.BalanceProjection{}, fmt.Errorf("failed to get savings rules: %w", err)
	}

	// 4. Конвертируем модели репозитория в доменные модели
	domainSnapshot := mapSnapshotToDomain(snapshot)
	domainTxns := mapTransactionsToDomain(confirmedTxns)
	domainRules := mapRulesToDomain(incomes, expenses, savingsRows)

	// 5. Вызываем чистую бизнес-логику
	projection := domain.CalculateProjectedBalance(
		domainSnapshot.Amount,
		domainSnapshot.AsOfDate,
		domainTxns,
		domainRules,
		targetDate,
	)

	return projection, nil
}

// --- Функции маппинга из repository моделей в domain модели ---

func mapSnapshotToDomain(s repository.BalanceSnapshot) domain.BalanceSnapshot {
	amount, err := pgNumericToDecimal(s.Amount)
	if err != nil {
		amount = decimal.Decimal{}
	}
	return domain.BalanceSnapshot{
		ID:        s.ID,
		UserID:    s.UserID,
		Amount:    amount,
		AsOfDate:  pgDateToTime(s.AsOfDate),
		CreatedAt: s.CreatedAt,
	}
}

func mapTransactionsToDomain(txns []repository.Transaction) []domain.ProjectedEvent {
	events := make([]domain.ProjectedEvent, 0, len(txns))
	for _, t := range txns {
		amount, err := pgNumericToDecimal(t.Amount)
		if err != nil {
			amount = decimal.Decimal{}
		}
		events = append(events, domain.ProjectedEvent{
			Date:        pgDateToTime(t.TxnDate),
			Type:        domain.TransactionType(t.Type),
			Amount:      amount,
			Description: pgTextToString(t.Note),
			SourceID:    uuid.Nil, // Для фактических транзакций источник не важен
		})
	}
	return events
}

func mapRulesToDomain(
	incomes []repository.GetActiveIncomeSourcesRow,
	expenses []repository.GetActiveExpenseObligationsRow,
	savingsRows []repository.GetActiveSavingsRulesForUserRow,
) domain.RulesSet {

	domainIncomes := make([]domain.IncomeSource, 0, len(incomes))
	for _, inc := range incomes {
		amount, err := pgNumericToDecimal(inc.Amount)
		if err != nil {
			amount = decimal.Decimal{}
		}
		domainIncomes = append(domainIncomes, domain.IncomeSource{
			ID:             inc.ID,
			Name:           inc.Name,
			Amount:         amount,
			DayOfMonth:     int(inc.DayOfMonth),
			OverflowPolicy: domain.OverflowPolicy(inc.OverflowPolicy),
		})
	}

	domainExpenses := make([]domain.ExpenseObligation, 0, len(expenses))
	for _, exp := range expenses {
		amount, err := pgNumericToDecimal(exp.Amount)
		if err != nil {
			amount = decimal.Decimal{}
		}
		domainExpenses = append(domainExpenses, domain.ExpenseObligation{
			ID:             exp.ID,
			Name:           exp.Name,
			Amount:         amount,
			DayOfMonth:     int(exp.DayOfMonth),
			OverflowPolicy: domain.OverflowPolicy(exp.OverflowPolicy),
		})
	}

	domainSavings := make([]domain.SavingsRule, 0, len(savingsRows))
	for _, sr := range savingsRows {
		value, err := pgNumericToDecimal(sr.Value)
		if err != nil {
			value = decimal.Decimal{}
		}
		domainSavings = append(domainSavings, domain.SavingsRule{
			ID:             sr.ID,
			IncomeSourceID: sr.IncomeSourceID,
			BucketName:     sr.BucketName,
			Mode:           domain.SavingsMode(sr.Mode),
			Value:          value,
		})
	}

	return domain.RulesSet{
		Incomes:  domainIncomes,
		Expenses: domainExpenses,
		Savings:  domainSavings,
	}
}
