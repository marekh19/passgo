package transfer

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/marekh19/passgo/internal/store"
	"github.com/marekh19/passgo/internal/tb"
)

// --- fakes: this slice's own, per the per-slice convention ---

// fakeTB records the last Transfer it was handed and how many times it was
// called. Set err to simulate a tb rejection (e.g. insufficient funds).
type fakeTB struct {
	last  tb.TransferReq
	calls int
	err   error
}

func (f *fakeTB) Transfer(_ context.Context, t tb.TransferReq) (tb.Uint128, error) {
	f.calls++
	f.last = t
	if f.err != nil {
		return tb.Uint128{}, f.err
	}
	return tb.IDFromBytes([16]byte{0xAA}), nil
}

func (f *fakeTB) CreateAccount(context.Context, uint32, uint16, tb.AccountFlags) (tb.Uint128, error) {
	return tb.Uint128{}, nil
}
func (f *fakeTB) Balances(context.Context, []tb.Uint128) (map[tb.Uint128]tb.Balance, error) {
	return nil, nil
}
func (f *fakeTB) Close() {}

// fakeStore serves one preset session + player list. getErr lets a test make
// GetSession fail (e.g. sql.ErrNoRows for an unknown session).
type fakeStore struct {
	ses     store.Session
	players []store.Player
	getErr  error
}

func (s *fakeStore) GetSession(_ context.Context, _ string) (store.Session, error) {
	if s.getErr != nil {
		return store.Session{}, s.getErr
	}
	return s.ses, nil
}
func (s *fakeStore) ListPlayers(_ context.Context, _ string) ([]store.Player, error) {
	return s.players, nil
}

// unused by the transfer slice -- present only to satisfy store.Store:
func (s *fakeStore) CreateSession(context.Context, store.Session) error  { return nil }
func (s *fakeStore) TouchSession(context.Context, string, int64) error   { return nil }
func (s *fakeStore) MarkStarted(context.Context, string) error           { return nil }
func (s *fakeStore) CreatePlayer(context.Context, store.Player) error    { return nil }
func (s *fakeStore) MaxLedgerID(context.Context) (uint32, error)         { return 0, nil }
func (s *fakeStore) SweepInactive(context.Context, int64) (int64, error) { return 0, nil }
func (s *fakeStore) Close() error                                        { return nil }

func newTestService(started bool) (Service, *fakeTB, *fakeStore) {
	fst := &fakeStore{
		ses: store.Session{
			Code:       "ABCD",
			LedgerID:   7,
			BankAcctID: [16]byte{0xB},
			Started:    started,
		},
		players: []store.Player{
			{ID: "p1", AcctID: [16]byte{0x01}},
			{ID: "p2", AcctID: [16]byte{0x02}},
		},
	}
	ftb := &fakeTB{}
	return New(ftb, fst), ftb, fst
}

// --- happy paths: the request reaches tb with the right fields ---

func TestExecuteBankToPlayer(t *testing.T) {
	svc, ftb, _ := newTestService(true)

	id, err := svc.Execute(context.Background(), "ABCD", Bank, "p1", 200, 23)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if (id == tb.Uint128{}) {
		t.Error("expected a non-zero transfer id")
	}
	if ftb.calls != 1 {
		t.Fatalf("tb.Transfer calls = %d, want 1", ftb.calls)
	}
	got := ftb.last
	if got.Ledger != 7 || got.Amount != 200 || got.Code != 23 {
		t.Errorf("transfer ledger/amount/code wrong: %+v", got)
	}
	if got.From != tb.IDFromBytes([16]byte{0xB}) {
		t.Errorf("From not the bank account: %+v", got.From)
	}
	if got.To != tb.IDFromBytes([16]byte{0x01}) {
		t.Errorf("To not p1's account: %+v", got.To)
	}
}

func TestExecutePlayerToPlayer(t *testing.T) {
	svc, ftb, _ := newTestService(true)

	if _, err := svc.Execute(context.Background(), "ABCD", "p1", "p2", 50, 10); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := ftb.last
	if got.From != tb.IDFromBytes([16]byte{0x01}) || got.To != tb.IDFromBytes([16]byte{0x02}) {
		t.Errorf("p2p resolved wrong: from=%+v to=%+v", got.From, got.To)
	}
}

// --- validation: bad input must never reach tb ---

func TestExecuteValidation(t *testing.T) {
	cases := []struct {
		name     string
		from, to string
		amount   uint64
		code     uint16
		want     error
	}{
		{"zero amount", Bank, "p1", 0, 23, ErrBadAmount},
		{"admin code 99", Bank, "p1", 100, 99, ErrBadCode},
		{"unknown code", Bank, "p1", 100, 5, ErrBadCode},
		{"p2p from bank", Bank, "p1", 100, 10, ErrBadDirection},
		{"bank code from player", "p1", "p2", 100, 23, ErrBadDirection},
		{"self transfer", "p1", "p1", 100, 10, ErrSelfTransfer},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, ftb, _ := newTestService(true)
			_, err := svc.Execute(context.Background(), "ABCD", c.from, c.to, c.amount, c.code)
			if !errors.Is(err, c.want) {
				t.Errorf("err = %v, want %v", err, c.want)
			}
			if ftb.calls != 0 {
				t.Errorf("tb was called %d times; bad input must not reach it", ftb.calls)
			}
		})
	}
}

func TestExecuteUnknownPlayer(t *testing.T) {
	svc, ftb, _ := newTestService(true)

	_, err := svc.Execute(context.Background(), "ABCD", Bank, "ghost", 100, 23)
	if !errors.Is(err, ErrUnknownPlayer) {
		t.Errorf("err = %v, want ErrUnknownPlayer", err)
	}
	if ftb.calls != 0 {
		t.Errorf("tb should not be called for an unknown player")
	}
}

func TestExecuteNotStarted(t *testing.T) {
	svc, ftb, _ := newTestService(false) // lobby, not started

	_, err := svc.Execute(context.Background(), "ABCD", Bank, "p1", 100, 23)
	if !errors.Is(err, ErrNotStarted) {
		t.Errorf("err = %v, want ErrNotStarted", err)
	}
	if ftb.calls != 0 {
		t.Errorf("tb should not be called before start")
	}
}

func TestExecuteUnknownSession(t *testing.T) {
	svc, ftb, fst := newTestService(true)
	fst.getErr = sql.ErrNoRows // session lookup misses

	_, err := svc.Execute(context.Background(), "ZZZZ", Bank, "p1", 100, 23)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("err = %v, want sql.ErrNoRows", err)
	}
	if ftb.calls != 0 {
		t.Errorf("tb should not be called when the session is unknown")
	}
}

func TestExecuteSurfacesTBError(t *testing.T) {
	svc, ftb, _ := newTestService(true)
	wantErr := errors.New("tb: transfer rejected: TransferExceedsCredits")
	ftb.err = wantErr // simulate insufficient funds

	_, err := svc.Execute(context.Background(), "ABCD", "p1", Bank, 9999, 32)
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want it to surface the tb error", err)
	}
}
