-- schema.sql is the single source of truth for the DB shape. sqlc reads it to
-- generate Go types; store.New applies it on open; `task db:migrate` re-applies
-- it. Hence IF NOT EXISTS everywhere -- every CREATE must be safe to run twice.
--
-- No PRAGMAs here. WAL + foreign_keys + busy_timeout are set per-connection in
-- store.go (foreign_keys especially is NOT persisted in the file, so the
-- ON DELETE CASCADE below only fires when the connection enabled it).

CREATE TABLE IF NOT EXISTS sessions (
  code         TEXT PRIMARY KEY,
  ledger_id    INTEGER NOT NULL UNIQUE,
  bank_acct_id BLOB NOT NULL,
  admin_id     TEXT NOT NULL,
  started      INTEGER NOT NULL DEFAULT 0,
  created_at   INTEGER NOT NULL,
  last_active  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS players (
  id           TEXT PRIMARY KEY,
  session_code TEXT NOT NULL REFERENCES sessions(code) ON DELETE CASCADE,
  name         TEXT NOT NULL,
  acct_id      BLOB NOT NULL,
  joined_at    INTEGER NOT NULL,
  active       INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_players_session ON players(session_code);
CREATE INDEX IF NOT EXISTS idx_sessions_last_active ON sessions(last_active);

