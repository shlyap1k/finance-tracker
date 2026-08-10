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