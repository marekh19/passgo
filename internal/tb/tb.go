// Package tb wraps the TigerBeetle Go client behind a small interface, so the
// rest of the app depends on tb, never the SDK directly.
package tb

import (
	"context"
	"fmt"

	tigerbeetle "github.com/tigerbeetle/tigerbeetle-go"
)

// Uint128 is TigerBeetle's 128-bit type for account IDs, transfer IDs, and
// amounts. Aliased (not redefined) so it's interchangeable with SDK's type
// at call sites while callers still import only tb.
type Uint128 = tigerbeetle.Uint128

// AccountFlags re-exports the SDK flag set. v1 sets only DebitsMustNotExceedCredits,
// on player accounts; the Bank runs unconstrained (can go negative).
type AccountFlags = tigerbeetle.AccountFlags

// NewID returns a fresh time-sortable u128 (ULID-style, monotonic) - TigerBeetle
// is optimized for IDs that increase over time, so prefer this over random.
func NewID() Uint128 { return tigerbeetle.ID() }

// IDFromBytes rebuilds a Uint128 from its 16-byte form -- the inverse of
// Uint128.Bytes(). Used to turn account IDs stored as BLOBs back into tb IDs.
func IDFromBytes(b [16]byte) Uint128 { return tigerbeetle.BytesToUint128(b) }

// TransferReq describes one money movement. Amount is whole dollars.
type TransferReq struct {
	Ledger   uint32  // session ledger
	From     Uint128 // debit account
	To       Uint128 // credit account
	Amount   uint64  // whole dollars ($200 = 200)
	Code     uint16  // 10 P2P, 20-23 bank->player, 30-32 player->bank, 99 admin
	UserData Uint128 // optional: original trasfer ID for an undo (code 99)
}

// Balance is an account's posted position, in whole dollars.
type Balance struct {
	DebitsPosted  uint64
	CreditsPosted uint64
}

// Net is the spendable balance (credits - debits). Player accounts stay >= 0;
// the Bank may be negative.
func (b Balance) Net() int64 { return int64(b.CreditsPosted) - int64(b.DebitsPosted) }

// Client is the public interface for this slice.
type Client interface {
	CreateAccount(ctx context.Context, ledger uint32, code uint16, flags AccountFlags) (Uint128, error)
	Transfer(ctx context.Context, t TransferReq) (Uint128, error)
	Balances(ctx context.Context, ids []Uint128) (map[Uint128]Balance, error)
	Close()
}

type client struct {
	tb tigerbeetle.Client
}

// New connects to a TigerBeetle cluster. addr is host:port (dev: 127.0.0.1:33000).
// Cluster 0 matches the dev compose (--cluster=0)
func New(addr string) (Client, error) {
	c, err := tigerbeetle.NewClient(tigerbeetle.ToUint128(0), []string{addr})
	if err != nil {
		return nil, fmt.Errorf("tb: connect %q: %w", addr, err)
	}
	return &client{tb: c}, nil
}

func (c *client) Close() { c.tb.Close() }

func (c *client) CreateAccount(ctx context.Context, ledger uint32, code uint16, flags AccountFlags) (Uint128, error) {
	id := NewID()
	res, err := c.tb.CreateAccounts([]tigerbeetle.Account{{
		ID:     id,
		Ledger: ledger,
		Code:   code,
		Flags:  flags.ToUint16(),
	}})
	if err != nil { // transport/connection failure
		return Uint128{}, fmt.Errorf("tb: create account: %w", err)
	}
	// Business failures arrive as per-event statuses, NOT as err. Empty slice or
	// an AccountCreated status = success; anything else we surface.
	for _, r := range res {
		if r.Status != tigerbeetle.AccountCreated {
			return Uint128{}, fmt.Errorf("tb: account rejected: %s", r.Status)
		}
	}
	return id, nil
}

func (c *client) Transfer(ctx context.Context, t TransferReq) (Uint128, error) {
	id := NewID()
	res, err := c.tb.CreateTransfers([]tigerbeetle.Transfer{{
		ID:              id,
		DebitAccountID:  t.From,
		CreditAccountID: t.To,
		Amount:          tigerbeetle.ToUint128(t.Amount),
		UserData128:     t.UserData,
		Ledger:          t.Ledger,
		Code:            t.Code,
	}})
	if err != nil {
		return Uint128{}, fmt.Errorf("tb: transfer: %w", err)
	}
	for _, r := range res {
		if r.Status != tigerbeetle.TransferCreated {
			// e.g. TransferExceedsCredits = insufficient funds. .String() is human-readable.
			return Uint128{}, fmt.Errorf("tb: transfer rejected: %s", r.Status)
		}
	}
	return id, nil
}

func (c *client) Balances(ctx context.Context, ids []Uint128) (map[Uint128]Balance, error) {
	accounts, err := c.tb.LookupAccounts(ids)
	if err != nil {
		return nil, fmt.Errorf("tb: lookup accounts: %w", err)
	}
	out := make(map[Uint128]Balance, len(accounts))
	for _, a := range accounts {
		// BigInt().Uint64() takes the low 64 bits unambigously. (Uint128.Uint64()
		// returns two words and invites lo/hi mistakes - avoid it for clarity).
		out[a.ID] = Balance{
			DebitsPosted:  a.DebitsPosted.BigInt().Uint64(),
			CreditsPosted: a.CreditsPosted.BigInt().Uint64(),
		}
	}
	return out, nil
}
