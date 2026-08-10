package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CalculateProjectedBalance реализует алгоритм ProjectMainBalance
func CalculateProjectedBalance(
	snapshotAmount decimal.Decimal,
	snapshotDate time.Time,
	confirmedTransactions []ProjectedEvent,
	rules RulesSet,
	targetDate time.Time,
) BalanceProjection {

	today := time.Now().UTC().Truncate(24 * time.Hour)
	targetDate = targetDate.UTC().Truncate(24 * time.Hour)

	projection := BalanceProjection{
		TargetDate:      targetDate,
		StartingBalance: snapshotAmount,
		FactChange:      decimal.Zero,
		FutureChange:    decimal.Zero,
	}

	// 1. Считаем факт: от даты снапшота до min(сегодня, target_date)
	factEndDate := targetDate
	if today.Before(targetDate) {
		factEndDate = today
	}

	if !snapshotDate.After(factEndDate) {
		for _, txn := range confirmedTransactions {
			if !txn.Date.Before(snapshotDate) && !txn.Date.After(factEndDate) {
				projection.FactChange = projection.FactChange.Add(signedAmount(txn))

				// Собираем статистику для period-summary
				accumulateSummary(&projection, txn)
			}
		}
	}

	// 2. Считаем прогноз: от сегодня + 1 день до target_date
	if targetDate.After(today) {
		fromDate := today.AddDate(0, 0, 1)
		futureEvents := ExpandRules(fromDate, targetDate, rules)

		for _, event := range futureEvents {
			projection.FutureChange = projection.FutureChange.Add(signedAmount(event))
			accumulateSummary(&projection, event)
		}
	}

	projection.FinalBalance = projection.StartingBalance.Add(projection.FactChange).Add(projection.FutureChange)
	return projection
}

// ExpandRules разворачивает правила в список событий на заданный период
func ExpandRules(from, to time.Time, rules RulesSet) []ProjectedEvent {
	var events []ProjectedEvent

	// Нормализуем даты до начала и конца дня в UTC
	current := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	if current.Before(from) {
		current = from
	}
	endDate := time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.UTC)

	// Проходим по каждому месяцу в диапазоне
	for !current.After(endDate) {
		year, month, _ := current.Date()

		// 1. Генерируем доходы за этот месяц, чтобы знать, какие накопления активируются
		monthIncomeEvents := make(map[uuid.UUID]ProjectedEvent)
		for _, inc := range rules.Incomes {
			date := resolveDate(year, month, inc.DayOfMonth, inc.OverflowPolicy)
			if !date.Before(from) && !date.After(to) {
				ev := ProjectedEvent{
					Date:        date,
					Type:        TypeIncome,
					Amount:      inc.Amount,
					Description: inc.Name,
					SourceID:    inc.ID,
				}
				events = append(events, ev)
				monthIncomeEvents[inc.ID] = ev
			}
		}

		// 2. Генерируем обязательные расходы
		for _, exp := range rules.Expenses {
			date := resolveDate(year, month, exp.DayOfMonth, exp.OverflowPolicy)
			if !date.Before(from) && !date.After(to) {
				events = append(events, ProjectedEvent{
					Date:        date,
					Type:        TypeExpense,
					Amount:      exp.Amount,
					Description: exp.Name,
					SourceID:    exp.ID,
				})
			}
		}

		// 3. Генерируем накопления (ТОЛЬКО если сработал привязанный доход в этом месяце)
		for _, sav := range rules.Savings {
			incomeEvent, incomeExists := monthIncomeEvents[sav.IncomeSourceID]
			if !incomeExists {
				continue // Доход не пришел, накопление не делаем
			}

			// Накопление происходит в тот же день, что и доход (можно вынести в отдельное поле, если нужно)
			savingAmount := decimal.Zero
			if sav.Mode == ModeFixed {
				savingAmount = sav.Value
			} else if sav.Mode == ModePercent {
				savingAmount = incomeEvent.Amount.Mul(sav.Value)
			}

			if savingAmount.GreaterThan(decimal.Zero) {
				events = append(events, ProjectedEvent{
					Date:        incomeEvent.Date,
					Type:        TypeSaving,
					Amount:      savingAmount,
					Description: "Накопление: " + sav.BucketName,
					SourceID:    sav.ID,
				})
			}
		}

		// Переход к следующему месяцу
		current = current.AddDate(0, 1, 0)
	}

	return events
}

// --- Вспомогательные функции ---

// resolveDate определяет реальную дату события с учетом политики переполнения
func resolveDate(year int, month time.Month, dayOfMonth int, policy OverflowPolicy) time.Time {
	// Последний день текущего месяца
	lastDayOfMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()

	if dayOfMonth <= lastDayOfMonth {
		return time.Date(year, month, dayOfMonth, 0, 0, 0, 0, time.UTC)
	}

	// Если дня нет в месяце (например, 31-е в феврале)
	if policy == PolicyBackward {
		// Переносим на последний день текущего месяца (28/29/30)
		return time.Date(year, month, lastDayOfMonth, 0, 0, 0, 0, time.UTC)
	}

	// PolicyForward: переносим на 1-е число следующего месяца
	nextMonth := month + 1
	nextYear := year
	if nextMonth > 12 {
		nextMonth = 1
		nextYear++
	}
	return time.Date(nextYear, nextMonth, 1, 0, 0, 0, 0, time.UTC)
}

func signedAmount(ev ProjectedEvent) decimal.Decimal {
	switch ev.Type {
	case TypeIncome:
		return ev.Amount
	case TypeExpense, TypeSaving:
		return ev.Amount.Neg() // Вычитаем
	default:
		return decimal.Zero
	}
}

func accumulateSummary(p *BalanceProjection, ev ProjectedEvent) {
	switch ev.Type {
	case TypeIncome:
		p.ProjectedIncome = p.ProjectedIncome.Add(ev.Amount)
	case TypeExpense:
		p.ProjectedExpense = p.ProjectedExpense.Add(ev.Amount)
	case TypeSaving:
		p.ProjectedSaving = p.ProjectedSaving.Add(ev.Amount)
	}
}

// RulesSet объединяет все правила пользователя для удобства передачи
type RulesSet struct {
	Incomes  []IncomeSource
	Expenses []ExpenseObligation
	Savings  []SavingsRule
}
