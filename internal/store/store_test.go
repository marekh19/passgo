package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionPlayerRoundTrip(t *testing.T) {
	// t.TempDir() is auto-removed when the test ends -- no cleanup needed.
	// A real file (not :memory:) because WAL needs one.
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = st.Close() }()

	ctx := context.Background()
	now := time.Now().Unix()
	bankAcct := [16]byte{0xBA, 0x17, 0xAC} // distinct bytes + trailing zeros on purpose

	// --- session: write, read back, compare ---
	want := Session{
		Code:       "ABCD",
		LedgerID:   1,
		BankAcctID: bankAcct,
		AdminID:    "admin-1",
		CreatedAt:  now,
		LastActive: now,
	}
	if err := st.CreateSession(ctx, want); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, err := st.GetSession(ctx, "ABCD")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got != want { // Session is comparable: every field (incl. [16]byte) is
		t.Errorf("session round-trip mismatch:\n got  %+v\n want %+v", got, want)
	}
	if got.Started {
		t.Error("new session should have Started=false (schema default)")
	}

	// flip started, confirm it persists
	if err := st.MarkStarted(ctx, "ABCD"); err != nil {
		t.Fatalf("MarkStarted: %v", err)
	}
	if got, _ := st.GetSession(ctx, "ABCD"); !got.Started {
		t.Error("MarkStarted did not set Started=true")
	}

	// unknown code -> sql.ErrNoRows passes through unwrapped
	if _, err := st.GetSession(ctx, "NOPE"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("GetSession(unknown): got %v, want sql.ErrNoRows", err)
	}

	// MaxLedgerID sees our ledger as the high-water mark
	if max, err := st.MaxLedgerID(ctx); err != nil || max != 1 {
		t.Errorf("MaxLedgerID = %d, %v; want 1, nil", max, err)
	}

	// --- player: create, list ---
	p := Player{
		ID:          "p1",
		SessionCode: "ABCD",
		Name:        "Marek",
		AcctID:      [16]byte{0x01},
		JoinedAt:    now,
	}
	if err := st.CreatePlayer(ctx, p); err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}

	players, err := st.ListPlayers(ctx, "ABCD")
	if err != nil {
		t.Fatalf("ListPlayers: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("ListPlayers len = %d, want 1", len(players))
	}
	if players[0].Name != "Marek" || players[0].AcctID != p.AcctID {
		t.Errorf("player mismatch: %+v", players[0])
	}
	if !players[0].Active {
		t.Error("new player should have Active=true (schema default)")
	}

	// --- cascade: sweeping the session must delete its players too ---
	// This is the real test of deviation #1: ON DELETE CASCADE only fires
	// because we set foreign_keys(1) on the DSN. Drop that pragma and this fails.
	n, err := st.SweepInactive(ctx, now+1) // cutoff in the future -> our session is stale
	if err != nil {
		t.Fatalf("SweepInactive: %v", err)
	}
	if n != 1 {
		t.Errorf("SweepInactive removed %d sessions, want 1", n)
	}
	if remaining, _ := st.ListPlayers(ctx, "ABCD"); len(remaining) != 0 {
		t.Errorf("cascade failed: %d players remain after session swept", len(remaining))
	}
}
