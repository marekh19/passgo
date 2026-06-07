// Package session owns game lifecycle: create a game, join it, start it. It's
// the bridge slice -- the one place tb.Uint128 account IDs meet store's [16]byte
// BLOBs. Depends only on the tb.Client and store.Store interfaces.
package session

import (
	"context"
	"errors"

	"github.com/marekh19/passgo/internal/store"
	"github.com/marekh19/passgo/internal/tb"
)

// Sentinel errors callers can match with errors.Is. Handlers map these to
// HTTP status + user-facing toasts.
var (
	ErrAlreadyStarted = errors.New("session: game already started")
	ErrNotAdmin       = errors.New("session: only the admin can start the game")
	ErrCodeExhausted  = errors.New("session: could not allocate a unique code")
)

// Manager is the public interface for this slice.
type Manager interface {
	// Create makes a new game, creates the Bank account, and enrolls the caller
	// as the admin player. Returns the session and that admin player.
	Create(ctx context.Context, adminName string) (store.Session, store.Player, error)
	// Join enrolls a new player in an existing, not-yet-started game.
	Join(ctx context.Context, code, playerName string) (store.Player, error)
	// Start credits every player $1500 from the Bank and marks the game live,
	// adminID must match the sessions's admin.
	Start(ctx context.Context, code, adminID string) error
}

type manager struct {
	tb    tb.Client
	store store.Store
}

// New wires the session manager over its two dependencies. Concrete impls are
// injected in cmd/passgo/main.go; everything else sees the Manager interface.
func New(tbc tb.Client, st store.Store) Manager {
	return &manager{tb: tbc, store: st}
}
