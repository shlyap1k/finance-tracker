-- name: UpsertBalanceSnapshot :one
INSERT INTO balance_snapshots (user_id, amount, as_of_date)
VALUES ($1, $2, $3)
    ON CONFLICT (user_id, as_of_date)
DO UPDATE SET amount = EXCLUDED.amount, created_at = now()
           RETURNING id, user_id, amount, as_of_date, created_at;

-- name: GetLatestSnapshot :one
SELECT id, user_id, amount, as_of_date, created_at
FROM balance_snapshots
WHERE user_id = $1
ORDER BY as_of_date DESC
    LIMIT 1;

-- name: GetLatestSnapshotAtOrBefore :one
SELECT id, user_id, amount, as_of_date, created_at
FROM balance_snapshots
WHERE user_id = $1 AND as_of_date <= $2
ORDER BY as_of_date DESC
    LIMIT 1;