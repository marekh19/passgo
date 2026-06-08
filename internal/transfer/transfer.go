// Package transfer executes one money movement: validate, resolve player IDs to
// TigerBeetle accounts, post to tb. Depends only on tb.Client + store.Store.
package transfer

import (
	"context"

	"github.com/marekh19/passgo/internal/store"
	"github.com/marekh19/passgo/internal/tb"
)

// Service is the public interface for this slice.
type Service interface {
	// Execute posts one transfer. from/to are player IDs or the Bank sentinel.
	// Returns the new transfer's tb ID.
	Execute(ctx context.Context, sessionCode, from, to string, amount uint64, code uint16) (tb.Uint128, error)
}

type service struct {
	tb    tb.Client
	store store.Store
}

func New(tbc tb.Client, st store.Store) Service {
	return &service{tb: tbc, store: st}
}

func (s *service) Execute(ctx context.Context, sessionCode, from, to string, amount uint64, code uint16) (tb.Uint128, error) {
	if err := validate(from, to, amount, code); err != nil {
		return tb.Uint128{}, err
	}

	ses, err := s.store.GetSession(ctx, sessionCode)
	if err != nil {
		return tb.Uint128{}, err // sql.ErrNoRows => unknown session
	}
	if !ses.Started {
		return tb.Uint128{}, ErrNotStarted
	}

	players, err := s.store.ListPlayers(ctx, sessionCode)
	if err != nil {
		return tb.Uint128{}, err
	}

	bank := tb.IDFromBytes(ses.BankAcctID)
	resolve := func(ref string) (tb.Uint128, error) {
		if ref == Bank {
			return bank, nil
		}
		for _, p := range players {
			if p.ID == ref {
				return tb.IDFromBytes(p.AcctID), nil
			}
		}
		return tb.Uint128{}, ErrUnknownPlayer
	}

	fromAcct, err := resolve(from)
	if err != nil {
		return tb.Uint128{}, err
	}
	toAcct, err := resolve(to)
	if err != nil {
		return tb.Uint128{}, err
	}

	// tb enforces the hard money rule: a player paying more than they have is
	// rejected (debits_must_not_exceed_credits). We surface that error, not swallow it.
	return s.tb.Transfer(ctx, tb.TransferReq{
		Ledger: ses.LedgerID,
		From:   fromAcct,
		To:     toAcct,
		Amount: amount,
		Code:   code,
	})
}
