package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marekh19/passgo/internal/auth"
	"github.com/marekh19/passgo/internal/session"
	"github.com/marekh19/passgo/internal/store"
	"github.com/marekh19/passgo/internal/tb"
	"github.com/marekh19/passgo/internal/transfer"
)

type testLedger struct {
	accounts map[tb.Uint128]tb.Balance
}

func (l *testLedger) CreateAccount(context.Context, uint32, uint16, tb.AccountFlags) (tb.Uint128, error) {
	id := tb.NewID()
	l.accounts[id] = tb.Balance{}
	return id, nil
}

func (l *testLedger) Transfer(_ context.Context, req tb.TransferReq) (tb.Uint128, error) {
	from := l.accounts[req.From]
	from.DebitsPosted += req.Amount
	l.accounts[req.From] = from
	to := l.accounts[req.To]
	to.CreditsPosted += req.Amount
	l.accounts[req.To] = to
	return tb.NewID(), nil
}

func (l *testLedger) Balances(_ context.Context, ids []tb.Uint128) (map[tb.Uint128]tb.Balance, error) {
	balances := make(map[tb.Uint128]tb.Balance, len(ids))
	for _, id := range ids {
		balances[id] = l.accounts[id]
	}
	return balances, nil
}

func (*testLedger) Close() {}

func TestGameFlow(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ledger := &testLedger{accounts: make(map[tb.Uint128]tb.Balance)}
	authn := auth.New([]byte("test-secret"), false)
	handler := New(st, session.New(ledger, st), transfer.New(ledger, st), authn, ledger).Handler()

	create := postForm(handler, "/api/sessions", "name=Alice")
	if create.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d: %s", create.Code, create.Body)
	}
	gameURL := create.Header().Get("Location")
	code := strings.TrimPrefix(gameURL, "/sessions/")
	creatorCookie := create.Result().Cookies()[0]

	getGame := func(cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, gameURL, nil)
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	if body := getGame(creatorCookie).Body.String(); !strings.Contains(body, "Start game") {
		t.Fatalf("creator cannot see lobby: %s", body)
	}
	if body := getGame(nil).Body.String(); !strings.Contains(body, "Join game "+code) {
		t.Fatalf("second player cannot see join form: %s", body)
	}

	join := postForm(handler, "/api/sessions/"+code+"/join", "name=Bob")
	if join.Code != http.StatusSeeOther {
		t.Fatalf("join status = %d: %s", join.Code, join.Body)
	}
	bobCookie := join.Result().Cookies()[0]
	start := postForm(handler, "/api/sessions/"+code+"/start", "", creatorCookie)
	if start.Code != http.StatusSeeOther {
		t.Fatalf("start status = %d: %s", start.Code, start.Body)
	}
	if body := getGame(creatorCookie).Body.String(); !strings.Contains(body, "$1,500") || !strings.Contains(body, "Bob") {
		t.Fatalf("starting balances missing: %s", body)
	}

	players, err := st.ListPlayers(context.Background(), code)
	if err != nil || len(players) != 2 {
		t.Fatalf("players = %v, err = %v", players, err)
	}
	payment := url.Values{
		"from":   {players[0].ID},
		"to":     {players[1].ID},
		"amount": {"200"},
		"code":   {"10"},
	}
	pay := postForm(handler, "/api/sessions/"+code+"/transfers", payment.Encode(), creatorCookie)
	if pay.Code != http.StatusSeeOther {
		t.Fatalf("transfer status = %d: %s", pay.Code, pay.Body)
	}
	if body := getGame(creatorCookie).Body.String(); !strings.Contains(body, "$1,300") || !strings.Contains(body, "$1,700") {
		t.Fatalf("updated balances missing: %s", body)
	}
	if body := getGame(bobCookie).Body.String(); !strings.Contains(body, "$1,700") {
		t.Fatalf("recipient balance missing: %s", body)
	}
}
