-- queries.sql -- sqlc reads these + schema.sql and generates type-safe Go in
-- generated/. The `-- name: X :mode` line is the directive; @foo are named params.

-- name: CreateSession :exec
INSERT INTO sessions (code, ledger_id, bank_acct_id, admin_id, created_at, last_active)
VALUES (@code, @ledger_id, @bank_acct_id, @admin_id, @created_at, @last_active);

-- name: GetSession :one
SELECT * FROM sessions WHERE code = @code;

-- name: UpdateSessionLastActive :exec
UPDATE sessions SET last_active = @last_active WHERE code = @code;

-- name: CreatePlayer :exec
INSERT INTO players (id, session_code, name, acct_id, joined_at)
VALUES (@id, @session_code, @name, @acct_id, @joined_at);

-- name: ListPlayersBySession :many
SELECT * FROM players WHERE session_code = @session_code ORDER BY joined_at;

-- name: GetMaxLedgerID :one
SELECT CAST(COALESCE(MAX(ledger_id), 0) AS INTEGER) AS max_ledger FROM sessions;

-- name: SweepInactiveSessions :execrows
DELETE FROM sessions WHERE last_active < @cutoff;

-- name: MarkSessionStarted :exec
UPDATE sessions SET started = 1 WHERE code = @code;
