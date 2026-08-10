-- name: GetConfirmedTransactionsInRange :many
SELECT
    id, user_id, type, amount, txn_date, status,
    income_source_id, expense_obligation_id, savings_rule_id, note, created_at, confirmed_at
FROM transactions
WHERE user_id = $1
  AND status IN ('confirmed', 'manual')
  AND txn_date >= $2
  AND txn_date <= $3
ORDER BY txn_date ASC;

-- name: CreateTransaction :one
INSERT INTO transactions (
    user_id, type, amount, txn_date, status,
    income_source_id, expense_obligation_id, savings_rule_id, note
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    RETURNING id, user_id, type, amount, txn_date, status, created_at;

-- name: ConfirmTransaction :one
UPDATE transactions
SET status = 'confirmed', confirmed_at = now(), amount = COALESCE($3, amount)
WHERE id = $1 AND user_id = $2 AND status = 'planned'
    RETURNING id, user_id, type, amount, txn_date, status, confirmed_at;