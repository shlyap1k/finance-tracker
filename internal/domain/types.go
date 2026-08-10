package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TransactionType определяет тип операции
type TransactionType string

const (
	TypeIncome  TransactionType = "income"
	TypeExpense TransactionType = "expense"
	TypeSaving  TransactionType = "saving"
	TypeAdhoc   TransactionType = "adhoc"
)

// OverflowPolicy определяет поведение, если дня (например, 31-го) нет в месяце
type OverflowPolicy string

const (
	PolicyForward  OverflowPolicy = "forward"  // Перенос на 1-е число следующего месяца
	PolicyBackward OverflowPolicy = "backward" // Перенос на последний день текущего месяца
)

// SavingsMode определяет, как рассчитывается накопление
type SavingsMode string

const (
	ModeFixed   SavingsMode = "fixed"   // Фиксированная сумма
	ModePercent SavingsMode = "percent" // Процент от суммы дохода
)

// --- Модели правил ---

type IncomeSource struct {
	ID             uuid.UUID
	Name           string
	Amount         decimal.Decimal
	DayOfMonth     int
	OverflowPolicy OverflowPolicy
}

type ExpenseObligation struct {
	ID             uuid.UUID
	Name           string
	Amount         decimal.Decimal
	DayOfMonth     int
	OverflowPolicy OverflowPolicy
}

type SavingsRule struct {
	ID             uuid.UUID
	IncomeSourceID uuid.UUID // Привязка к конкретному источнику дохода
	BucketName     string
	Mode           SavingsMode
	Value          decimal.Decimal // Если percent, то 0.1 = 10%
}

// --- Модели для расчетов ---

// ProjectedEvent представляет собой гипотетическое или реальное событие, влияющее на баланс
type ProjectedEvent struct {
	Date        time.Time
	Type        TransactionType
	Amount      decimal.Decimal
	Description string
	SourceID    uuid.UUID // ID правила, которое породило это событие (для отслеживания)
}

// BalanceProjection итоговый результат расчета
type BalanceProjection struct {
	TargetDate      time.Time
	StartingBalance decimal.Decimal
	FactChange      decimal.Decimal // Изменение баланса по подтвержденным транзакциям
	FutureChange    decimal.Decimal // Изменение баланса по прогнозу правил
	FinalBalance    decimal.Decimal

	// Детализация для периода (для эндпоинта period-summary)
	ProjectedIncome  decimal.Decimal
	ProjectedExpense decimal.Decimal
	ProjectedSaving  decimal.Decimal
}

type BalanceSnapshot struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Amount    decimal.Decimal
	AsOfDate  time.Time
	CreatedAt time.Time
}

// SavingsBucket представляет цель накопления
type SavingsBucket struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Name          string
	TargetAmount  decimal.Decimal
	CurrentAmount decimal.Decimal
	CreatedAt     time.Time
}
