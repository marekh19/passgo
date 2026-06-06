// Package store persists session and player metadata in SQLite (the money lives
// in TigerBeetle, not here). It wraps the sqlc-generated code behind a small
// Store interface and translates raw column types (int64, []byte) into domain
// types. Account IDs are opaque [16]byte values -- store knows nothing about
// TigerBeetle; the session slice bridges these to tb.Uint128.
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/marekh19/passgo/internal/store/generated"
	_ "modernc.org/sqlite" // registers the pure-Go "sqlite" driver
)

// schema.sql is baked into the binary so New can apply it without a file path
// at runtime. Every CREATE is IF NOT EXISTS, so applying is safe.
//
//go:embed schema.sql
var schema string

// Session is one game's metadata row. LedgerID is the TigerBeetle ledger for
// this session; BankAcctID is the Bank account's u128, stored as raw bytes.
type Session struct {
	Code       string
	LedgerID   uint32
	BankAcctID [16]byte
	AdminID    string
	Started    bool
	CreatedAt  int64 // unix seconds
	LastActive int64 // unix seconds
}

// Player is one participant in a session. AcctID is their TigerBeetle account.
type Player struct {
	ID          string
	SessionCode string
	Name        string
	AcctID      [16]byte
	JoinedAt    int64 // unix seconds
	Active      bool
}

// Store is the public interface for this slice. Callers depend on this, never
// on *store or the generated package.
type Store interface {
	CreateSession(ctx context.Context, s Session) error
	GetSession(ctx context.Context, code string) (Session, error)
	TouchSession(ctx context.Context, code string, at int64) error
	CreatePlayer(ctx context.Context, p Player) error
	ListPlayers(ctx context.Context, sessionCode string) ([]Player, error)
	MaxLedgerID(ctx context.Context) (uint32, error)
	SweepInactive(ctx context.Context, cutoff int64) (int64, error)
	Close() error
}

type store struct {
	db *sql.DB
	q  *generated.Queries
}

// New opens (creating if absent) the SQLite DB at path and applies the schema.
// Pragmas ride on the DSN because foreign_keys is per-connection and NOT
// persisted in the file -- without it the ON DELETE CASCADE silently no-ops.
// WAL gives readers-don't-block-writer; busy_timeout retries instead of erroring
// on a momentarily locked DB.
func New(path string) (Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open %q: %w", path, err)
	}
	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: apply schema: %w", err)
	}
	return &store{db: db, q: generated.New(db)}, nil
}

func (s *store) Close() error { return s.db.Close() }

func (s *store) CreateSession(ctx context.Context, ses Session) error {
	return s.q.CreateSession(ctx, generated.CreateSessionParams{
		Code:       ses.Code,
		LedgerID:   int64(ses.LedgerID),
		BankAcctID: ses.BankAcctID[:], // [16]byte -> []byte for the BLOB
		AdminID:    ses.AdminID,
		CreatedAt:  ses.CreatedAt,
		LastActive: ses.LastActive,
	})
}

// GetSession returns sql.ErrNoRows (unwrapped) when the code is unknown, so
// callers can errors.Is(err, sql.ErrNoRows) to tell "missing" from "broken".
func (s *store) GetSession(ctx context.Context, code string) (Session, error) {
	row, err := s.q.GetSession(ctx, code)
	if err != nil {
		return Session{}, err
	}
	return Session{
		Code:       row.Code,
		LedgerID:   uint32(row.LedgerID),
		BankAcctID: toAcct(row.BankAcctID),
		AdminID:    row.AdminID,
		Started:    row.Started != 0, // SQLite int 0/1 -> bool
		CreatedAt:  row.CreatedAt,
		LastActive: row.LastActive,
	}, nil
}

func (s *store) TouchSession(ctx context.Context, code string, at int64) error {
	return s.q.UpdateSessionLastActive(ctx, generated.UpdateSessionLastActiveParams{
		LastActive: at,
		Code:       code,
	})
}

func (s *store) CreatePlayer(ctx context.Context, p Player) error {
	return s.q.CreatePlayer(ctx, generated.CreatePlayerParams{
		ID:          p.ID,
		SessionCode: p.SessionCode,
		Name:        p.Name,
		AcctID:      p.AcctID[:],
		JoinedAt:    p.JoinedAt,
	})
}

func (s *store) ListPlayers(ctx context.Context, sessionCode string) ([]Player, error) {
	rows, err := s.q.ListPlayersBySession(ctx, sessionCode)
	if err != nil {
		return nil, err
	}
	players := make([]Player, len(rows))
	for i, r := range rows {
		players[i] = Player{
			ID:          r.ID,
			SessionCode: r.SessionCode,
			Name:        r.Name,
			AcctID:      toAcct(r.AcctID),
			JoinedAt:    r.JoinedAt,
			Active:      r.Active != 0,
		}
	}
	return players, nil
}

func (s *store) MaxLedgerID(ctx context.Context) (uint32, error) {
	m, err := s.q.GetMaxLedgerID(ctx)
	return uint32(m), err
}

func (s *store) SweepInactive(ctx context.Context, cutoff int64) (int64, error) {
	return s.q.SweepInactiveSessions(ctx, cutoff)
}

// toAcct copies a BLOB column ([]byte) into a fixed [16]byte account Id. Stored
// values are always exactly 16 bytes; a short read just leaves trailing zeros.
func toAcct(b []byte) [16]byte {
	var a [16]byte
	copy(a[:], b)
	return a
}
