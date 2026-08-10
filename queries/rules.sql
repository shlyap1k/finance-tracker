-- name: GetActiveIncomeSources :many
SELECT id, user_id, name, amount, day_of_month, overflow_policy, created_at
FROM income_sources
WHERE user_id = $1 AND archived_at IS NULL;

-- name: GetActiveExpenseObligations :many
SELECT id, user_id, name, amount, day_of_month, overflow_policy, created_at
FROM expense_obligations
WHERE user_id = $1 AND archived_at IS NULL;

-- name: GetActiveSavingsRulesForUser :many
SELECT
    sr.id, sr.income_source_id, sr.bucket_id, sr.mode, sr.value, sr.created_at,
    sb.name AS bucket_name, sb.target_amount, sb.current_amount
FROM savings_rules sr
         JOIN savings_buckets sb ON sr.bucket_id = sb.id
         JOIN income_sources ins ON sr.income_source_id = ins.id
WHERE ins.user_id = $1 AND ins.archived_at IS NULL;

-- name: CreateIncomeSource :one
INSERT INTO income_sources (user_id, name, amount, day_of_month, overflow_policy)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, name, amount, day_of_month, overflow_policy, archived_at, created_at;

-- name: UpdateIncomeSource :one
UPDATE income_sources
SET name = COALESCE($3, name),
    amount = COALESCE($4, amount),
    day_of_month = COALESCE($5, day_of_month),
    overflow_policy = COALESCE($6, overflow_policy)
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, name, amount, day_of_month, overflow_policy, archived_at, created_at;

-- name: ArchiveIncomeSource :exec
UPDATE income_sources
SET archived_at = now()
WHERE id = $1 AND user_id = $2;

-- name: CreateExpenseObligation :one
INSERT INTO expense_obligations (user_id, name, amount, day_of_month, overflow_policy)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, name, amount, day_of_month, overflow_policy, archived_at, created_at;

-- name: UpdateExpenseObligation :one
UPDATE expense_obligations
SET name = COALESCE($3, name),
    amount = COALESCE($4, amount),
    day_of_month = COALESCE($5, day_of_month),
    overflow_policy = COALESCE($6, overflow_policy)
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, name, amount, day_of_month, overflow_policy, archived_at, created_at;

-- name: ArchiveExpenseObligation :exec
UPDATE expense_obligations
SET archived_at = now()
WHERE id = $1 AND user_id = $2;

-- name: CreateSavingsBucket :one
INSERT INTO savings_buckets (user_id, name, target_amount)
VALUES ($1, $2, $3)
RETURNING id, user_id, name, target_amount, current_amount, created_at;

-- name: UpdateSavingsBucket :one
UPDATE savings_buckets
SET name = COALESCE($3, name),
    target_amount = COALESCE($4, target_amount)
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, name, target_amount, current_amount, created_at;

-- name: DeleteSavingsBucket :exec
DELETE FROM savings_buckets
WHERE id = $1 AND user_id = $2;

-- name: CreateSavingsRule :one
INSERT INTO savings_rules (income_source_id, bucket_id, mode, value)
VALUES ($1, $2, $3, $4)
RETURNING id, income_source_id, bucket_id, mode, value, created_at;

-- name: UpdateSavingsRule :one
UPDATE savings_rules
SET mode = COALESCE($3, mode),
    value = COALESCE($4, value)
WHERE id = $1
RETURNING id, income_source_id, bucket_id, mode, value, created_at;

-- name: DeleteSavingsRule :exec
DELETE FROM savings_rules
WHERE id = $1;

-- name: GetIncomeSourceByID :one
SELECT id, user_id, name, amount, day_of_month, overflow_policy, archived_at, created_at
FROM income_sources
WHERE id = $1;

-- name: GetExpenseObligationByID :one
SELECT id, user_id, name, amount, day_of_month, overflow_policy, archived_at, created_at
FROM expense_obligations
WHERE id = $1;

-- name: GetSavingsBucketByID :one
SELECT id, user_id, name, target_amount, current_amount, created_at
FROM savings_buckets
WHERE id = $1;

-- name: GetSavingsRuleByID :one
SELECT id, income_source_id, bucket_id, mode, value, created_at
FROM savings_rules
WHERE id = $1;

-- name: GetAllSavingsBucketsByUserID :many
SELECT id, user_id, name, target_amount, current_amount, created_at
FROM savings_buckets
WHERE user_id = $1;

-- name: GetAllSavingsRulesByUserID :many
SELECT id, income_source_id, bucket_id, mode, value, created_at
FROM savings_rules
WHERE income_source_id IN (
    SELECT id FROM income_sources WHERE user_id = $1
);