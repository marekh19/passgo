package session

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/marekh19/passgo/internal/store"
	"github.com/marekh19/passgo/internal/tb"
)

// TigerBeetle account codes + transfer code/amount for the opening deal.
// Account codes mirror the data model in CLAUDE.md (1=Player, 2=Bank).
const (
	acctPlayer = 1 // debits_must_not_exceed_credits
	acctBank   = 2 // unconstrained, may go negative

	codeInitialDeal = 23   // generic Bank -> Player; the opening $1500 stack
	startingCash    = 1500 // dollars each player gets on Start

	maxCodeTries = 5 // bound the code-collision retry loop
)

// Create makes a new game: allocates a ledger, mints the Bank account, enrolls
// the caller as the admin player (with their own account), and persists both
// rows. Returns the session and the admin player (caller needs the player ID).
func (m *manager) Create(ctx context.Context, adminName string) (store.Session, store.Player, error) {
	ledger, err := m.nextLedger(ctx)
	if err != nil {
		return store.Session{}, store.Player{}, err
	}
	code, err := m.uniqueCode(ctx)
	if err != nil {
		return store.Session{}, store.Player{}, err
	}

	now := time.Now().Unix()

	// Bank first: if tb is unreachable we fail before writing any SQLite rows.
	bankAcct, err := m.tb.CreateAccount(ctx, ledger, acctBank, tb.AccountFlags{})
	if err != nil {
		return store.Session{}, store.Player{}, err
	}

	adminID, err := newPlayerID()
	if err != nil {
		return store.Session{}, store.Player{}, err
	}
	adminAcct, err := m.tb.CreateAccount(ctx, ledger, acctPlayer, tb.AccountFlags{DebitsMustNotExceedCredits: true})
	if err != nil {
		return store.Session{}, store.Player{}, err
	}

	ses := store.Session{
		Code:       code,
		LedgerID:   ledger,
		BankAcctID: bankAcct.Bytes(), // tb.Uint128 -> [16]byte, the bridge
		AdminID:    adminID,
		CreatedAt:  now,
		LastActive: now,
	}
	if err := m.store.CreateSession(ctx, ses); err != nil {
		return store.Session{}, store.Player{}, err
	}

	admin := store.Player{
		ID:          adminID,
		SessionCode: code,
		Name:        adminName,
		AcctID:      adminAcct.Bytes(),
		JoinedAt:    now,
		Active:      true,
	}
	if err := m.store.CreatePlayer(ctx, admin); err != nil {
		return store.Session{}, store.Player{}, err
	}

	return ses, admin, nil
}

// Join enrolls a new player in an existing, not-yet-started game.
func (m *manager) Join(ctx context.Context, code, playerName string) (store.Player, error) {
	ses, err := m.store.GetSession(ctx, code)
	if err != nil {
		return store.Player{}, err // sql.ErrNoRows => unknown code
	}
	if ses.Started {
		return store.Player{}, ErrAlreadyStarted
	}

	id, err := newPlayerID()
	if err != nil {
		return store.Player{}, err
	}
	acct, err := m.tb.CreateAccount(ctx, ses.LedgerID, acctPlayer, tb.AccountFlags{DebitsMustNotExceedCredits: true})
	if err != nil {
		return store.Player{}, err
	}

	p := store.Player{
		ID:          id,
		SessionCode: code,
		Name:        playerName,
		AcctID:      acct.Bytes(),
		JoinedAt:    time.Now().Unix(),
		Active:      true,
	}
	if err := m.store.CreatePlayer(ctx, p); err != nil {
		return store.Player{}, err
	}
	return p, nil
}

// Start credits every player the opening $1500 from the Bank, then marks the
// game live. Admin-only; rejects a double start.
func (m *manager) Start(ctx context.Context, code, adminID string) error {
	ses, err := m.store.GetSession(ctx, code)
	if err != nil {
		return err
	}
	if ses.AdminID != adminID {
		return ErrNotAdmin
	}
	if ses.Started {
		return ErrAlreadyStarted
	}

	players, err := m.store.ListPlayers(ctx, code)
	if err != nil {
		return err
	}

	bank := tb.IDFromBytes(ses.BankAcctID) // [16]byte -> tb.Uint128, the bridge
	for _, p := range players {
		if _, err := m.tb.Transfer(ctx, tb.TransferReq{
			Ledger: ses.LedgerID,
			From:   bank,
			To:     tb.IDFromBytes(p.AcctID),
			Amount: startingCash,
			Code:   codeInitialDeal,
		}); err != nil {
			return err
		}
	}

	return m.store.MarkStarted(ctx, code)
}

// nextLedger allocates the next ledger ID. The UNIQUE constraint on ledger_id is
// the real race guard; for single-instance v1, read-then-increment is fine.
func (m *manager) nextLedger(ctx context.Context) (uint32, error) {
	max, err := m.store.MaxLedgerID(ctx)
	if err != nil {
		return 0, err
	}
	return max + 1, nil
}

// uniqueCode returns a code not already used by a session, bounded retries.
// A free code is one GetSession reports as sql.ErrNoRows.
func (m *manager) uniqueCode(ctx context.Context) (string, error) {
	for range maxCodeTries {
		code, err := newCode()
		if err != nil {
			return "", err
		}
		if _, err := m.store.GetSession(ctx, code); errors.Is(err, sql.ErrNoRows) {
			return code, nil // free
		} else if err != nil {
			return "", err // real DB error, not "missing"
		}
		// code taken -> loop
	}
	return "", ErrCodeExhausted
}
