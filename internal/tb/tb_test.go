package tb

import (
	"context"
	"os"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	addr := os.Getenv("TB_ADDR")
	if addr == "" {
		addr = "127.0.0.1:33000" // dev compose host port
	}

	client, err := New(addr)
	if err != nil {
		t.Skipf("TigerBeetle unreachable at %s: %v", addr, err)
	}
	defer client.Close()

	ctx := context.Background()
	const ledger = 1

	// Bank (code 2): no flags - may go negative. Probes connectivity; if TB is
	// down the first call fails here, so we skip rather than fail the suite.
	bank, err := client.CreateAccount(ctx, ledger, 2, AccountFlags{})
	if err != nil {
		t.Skipf("TigerBeetle not responding (is `task tb:up` up?): %v", err)
	}

	// Player (code 1): debits must not exceed credits.
	player, err := client.CreateAccount(ctx, ledger, 1, AccountFlags{DebitsMustNotExceedCredits: true})
	if err != nil {
		t.Fatalf("create player: %v", err)
	}

	// Bank pays player $200: money leaves bank (debit) -> arrives at player (credit).
	if _, err := client.Transfer(ctx, TransferReq{
		Ledger: ledger,
		From:   bank,   // debit account
		To:     player, // credit account
		Amount: 200,
		Code:   23, // generic collect-from-bank
	}); err != nil {
		t.Fatalf("transfer: %v", err)
	}

	bals, err := client.Balances(ctx, []Uint128{bank, player})
	if err != nil {
		t.Fatalf("balances: %v", err)
	}

	if got := bals[player].Net(); got != 200 {
		t.Errorf("player net = %d, want 200", got)
	}
	if got := bals[bank].Net(); got != -200 {
		t.Errorf("bank net = %d, want -200", got)
	}
}
