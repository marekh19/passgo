package session

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/marekh19/passgo/internal/store"
	"github.com/marekh19/passgo/internal/tb"
)

// --- fakes: hand-written, no mocking framework (per project convention) ---

// fakeTB records every account + transfer it's asked to make and hands back
// deterministic, distinct IDs so tests can match them back.
type fakeTB struct {
	nextID    byte
	accounts  []fakeAccount
	transfers []tb.TransferReq
}

type fakeAccount struct {
	id     tb.Uint128
	ledger uint32
	code   uint16
	flags  tb.AccountFlags
}

func (f *fakeTB) CreateAccount(_ context.Context, ledger uint32, code uint16, flags tb.AccountFlags) (tb.Uint128, error) {
	f.nextID++
	id := tb.IDFromBytes([16]byte{f.nextID}) // {1}, {2}, {3}, ... all distinct
	f.accounts = append(f.accounts, fakeAccount{id, ledger, code, flags})
	return id, nil
}

func (f *fakeTB) Transfer(_ context.Context, t tb.TransferReq) (tb.Uint128, error) {
	f.transfers = append(f.transfers, t)
	return tb.IDFromBytes([16]byte{0xFF, f.nextID}), nil
}

func (f *fakeTB) Balances(context.Context, []tb.Uint128) (map[tb.Uint128]tb.Balance, error) {
	return nil, nil
}
func (f *fakeTB) Close() {}

// fakeStore is an in-memory store.Store. It returns sql.ErrNoRows for missing
// sessions -- the same contract the real store promises, which uniqueCode/Join
// depend on.
type fakeStore struct {
	sessions map[string]store.Session
	players  map[string][]store.Player // keyed by session code
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		sessions: map[string]store.Session{},
		players:  map[string][]store.Player{},
	}
}

func (s *fakeStore) CreateSession(_ context.Context, ses store.Session) error {
	if _, ok := s.sessions[ses.Code]; ok {
		return errors.New("fake: duplicate code")
	}
	s.sessions[ses.Code] = ses
	return nil
}

func (s *fakeStore) GetSession(_ context.Context, code string) (store.Session, error) {
	ses, ok := s.sessions[code]
	if !ok {
		return store.Session{}, sql.ErrNoRows
	}
	return ses, nil
}

func (s *fakeStore) MarkStarted(_ context.Context, code string) error {
	ses, ok := s.sessions[code]
	if !ok {
		return sql.ErrNoRows
	}
	ses.Started = true
	s.sessions[code] = ses
	return nil
}

func (s *fakeStore) CreatePlayer(_ context.Context, p store.Player) error {
	s.players[p.SessionCode] = append(s.players[p.SessionCode], p)
	return nil
}

func (s *fakeStore) ListPlayers(_ context.Context, code string) ([]store.Player, error) {
	return s.players[code], nil
}

func (s *fakeStore) MaxLedgerID(context.Context) (uint32, error) {
	var max uint32
	for _, ses := range s.sessions {
		if ses.LedgerID > max {
			max = ses.LedgerID
		}
	}
	return max, nil
}

func (s *fakeStore) TouchSession(context.Context, string, int64) error   { return nil }
func (s *fakeStore) SweepInactive(context.Context, int64) (int64, error) { return 0, nil }
func (s *fakeStore) Close() error                                        { return nil }

func newTestManager() (Manager, *fakeTB, *fakeStore) {
	ftb, fst := &fakeTB{}, newFakeStore()
	return New(ftb, fst), ftb, fst
}

// --- tests ---

func TestCreate(t *testing.T) {
	m, ftb, fst := newTestManager()
	ctx := context.Background()

	ses, admin, err := m.Create(ctx, "Marek")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if ses.LedgerID != 1 {
		t.Errorf("ledger = %d, want 1", ses.LedgerID)
	}
	if len(ses.Code) != codeLen {
		t.Errorf("code %q len = %d, want %d", ses.Code, len(ses.Code), codeLen)
	}
	if ses.AdminID != admin.ID {
		t.Errorf("admin_id %q != admin player id %q", ses.AdminID, admin.ID)
	}
	if ses.Started {
		t.Error("new session should not be started")
	}
	if admin.Name != "Marek" || !admin.Active {
		t.Errorf("admin player = %+v", admin)
	}

	// persisted, not just returned
	if _, err := fst.GetSession(ctx, ses.Code); err != nil {
		t.Errorf("session not persisted: %v", err)
	}
	if players, _ := fst.ListPlayers(ctx, ses.Code); len(players) != 1 {
		t.Errorf("want 1 persisted player, got %d", len(players))
	}

	// two tb accounts: bank (code 2, unconstrained) + admin (code 1, dmnec)
	if len(ftb.accounts) != 2 {
		t.Fatalf("want 2 tb accounts, got %d", len(ftb.accounts))
	}
	bank, player := ftb.accounts[0], ftb.accounts[1]
	if bank.code != acctBank || bank.flags.DebitsMustNotExceedCredits {
		t.Errorf("bank account wrong: %+v", bank)
	}
	if player.code != acctPlayer || !player.flags.DebitsMustNotExceedCredits {
		t.Errorf("player account wrong: %+v", player)
	}
}

func TestJoin(t *testing.T) {
	m, ftb, _ := newTestManager()
	ctx := context.Background()

	ses, _, err := m.Create(ctx, "Marek")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	bob, err := m.Join(ctx, ses.Code, "Bob")
	if err != nil {
		t.Fatalf("Join: %v", err)
	}
	if bob.Name != "Bob" || bob.SessionCode != ses.Code || !bob.Active {
		t.Errorf("joined player = %+v", bob)
	}

	// bank + admin + bob = 3 accounts; bob's is a constrained player account
	if len(ftb.accounts) != 3 {
		t.Fatalf("want 3 tb accounts, got %d", len(ftb.accounts))
	}
	if last := ftb.accounts[2]; last.code != acctPlayer || !last.flags.DebitsMustNotExceedCredits {
		t.Errorf("bob account wrong: %+v", last)
	}

	// unknown code surfaces the store's not-found contract
	if _, err := m.Join(ctx, "ZZZZ", "Nobody"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Join(unknown) = %v, want sql.ErrNoRows", err)
	}
}

func TestStart(t *testing.T) {
	m, ftb, fst := newTestManager()
	ctx := context.Background()

	ses, admin, _ := m.Create(ctx, "Marek")
	bob, _ := m.Join(ctx, ses.Code, "Bob")

	// non-admin can't start
	if err := m.Start(ctx, ses.Code, "not-the-admin"); !errors.Is(err, ErrNotAdmin) {
		t.Errorf("Start(non-admin) = %v, want ErrNotAdmin", err)
	}

	if err := m.Start(ctx, ses.Code, admin.ID); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// every player paid the opening stack, from the bank, on the right ledger
	if len(ftb.transfers) != 2 {
		t.Fatalf("want 2 opening transfers, got %d", len(ftb.transfers))
	}
	bank := tb.IDFromBytes(ses.BankAcctID)
	for _, tr := range ftb.transfers {
		if tr.Amount != startingCash || tr.Code != codeInitialDeal {
			t.Errorf("opening transfer wrong amount/code: %+v", tr)
		}
		if tr.From != bank || tr.Ledger != ses.LedgerID {
			t.Errorf("opening transfer wrong from/ledger: %+v", tr)
		}
	}
	// recipients are exactly the two players' accounts
	paid := map[tb.Uint128]bool{ftb.transfers[0].To: true, ftb.transfers[1].To: true}
	for _, p := range []store.Player{admin, bob} {
		if !paid[tb.IDFromBytes(p.AcctID)] {
			t.Errorf("player %s was not paid", p.Name)
		}
	}

	if got, _ := fst.GetSession(ctx, ses.Code); !got.Started {
		t.Error("session not marked started")
	}

	// double start + late join both rejected
	if err := m.Start(ctx, ses.Code, admin.ID); !errors.Is(err, ErrAlreadyStarted) {
		t.Errorf("double Start = %v, want ErrAlreadyStarted", err)
	}
	if _, err := m.Join(ctx, ses.Code, "TooLate"); !errors.Is(err, ErrAlreadyStarted) {
		t.Errorf("Join(started) = %v, want ErrAlreadyStarted", err)
	}
}
