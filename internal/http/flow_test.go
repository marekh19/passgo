package http

import (
	"bufio"
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestLiveUpdatesStreamAndFragments(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ledger := &testLedger{accounts: make(map[tb.Uint128]tb.Balance)}
	authn := auth.New([]byte("test-secret"), false)
	srv := New(st, session.New(ledger, st), transfer.New(ledger, st), authn, ledger)
	httpSrv := httptest.NewServer(srv.Handler())
	defer httpSrv.Close()

	alice := testClient(t)
	bob := testClient(t)

	create := livePost(t, alice, httpSrv.URL+"/api/sessions", "name=Alice")
	code := strings.TrimPrefix(create.Header.Get("Location"), "/sessions/")

	stream := openLiveStream(t, alice, httpSrv.URL+"/sessions/"+code+"/events")
	defer stream.Body.Close()

	join := livePost(t, bob, httpSrv.URL+"/api/sessions/"+code+"/join", "name=Bob")
	if join.StatusCode != http.StatusSeeOther {
		t.Fatalf("join status = %d", join.StatusCode)
	}
	if got := readSSEEvent(t, stream); got != "changed" {
		t.Fatalf("event = %q, want changed", got)
	}
	roster := liveGet(t, alice, httpSrv.URL+"/sessions/"+code+"/lobby/roster")
	if roster.StatusCode != http.StatusOK {
		t.Fatalf("roster status = %d", roster.StatusCode)
	}
	body := readBody(t, roster)
	for _, want := range []string{"id=\"roster\"", "Alice", "Bob", "(you)", "host"} {
		if !strings.Contains(body, want) {
			t.Fatalf("roster missing %q: %s", want, body)
		}
	}

	bobStream := openLiveStream(t, bob, httpSrv.URL+"/sessions/"+code+"/events")
	defer bobStream.Body.Close()
	start := livePost(t, alice, httpSrv.URL+"/api/sessions/"+code+"/start", "")
	if start.StatusCode != http.StatusSeeOther {
		t.Fatalf("start status = %d", start.StatusCode)
	}
	if got := readSSEEvent(t, bobStream); got != "started" {
		t.Fatalf("event = %q, want started", got)
	}
	if rec := liveGet(t, bob, httpSrv.URL+"/sessions/"+code+"/lobby/roster"); rec.StatusCode != http.StatusConflict {
		t.Fatalf("started roster status = %d, want 409", rec.StatusCode)
	}
	balances := liveGet(t, bob, httpSrv.URL+"/sessions/"+code+"/balances")
	if balances.StatusCode != http.StatusOK {
		t.Fatalf("balances status = %d", balances.StatusCode)
	}
	if body := readBody(t, balances); !strings.Contains(body, "id=\"balance\"") || !strings.Contains(body, "id=\"players\"") || !strings.Contains(body, "$1,500") {
		t.Fatalf("balances fragment missing expected content: %s", body)
	}
	if count := srv.live.subscriberCount(code); count == 0 {
		t.Fatal("expected active subscribers")
	}
	bobStream.Body.Close()
	waitFor(t, func() bool { return srv.live.subscriberCount(code) == 1 })
}

func testClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func livePost(t *testing.T, client *http.Client, target, body string) *http.Response {
	t.Helper()
	resp, err := client.Post(target, "application/x-www-form-urlencoded", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func liveGet(t *testing.T, client *http.Client, target string) *http.Response {
	t.Helper()
	resp, err := client.Get(target)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func openLiveStream(t *testing.T, client *http.Client, target string) *http.Response {
	t.Helper()
	resp := liveGet(t, client, target)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stream status = %d: %s", resp.StatusCode, readBody(t, resp))
	}
	return resp
}

func readSSEEvent(t *testing.T, resp *http.Response) string {
	t.Helper()
	got := make(chan string, 1)
	go func() {
		s := bufio.NewScanner(resp.Body)
		for s.Scan() {
			line := s.Text()
			if strings.HasPrefix(line, "event: ") {
				got <- strings.TrimPrefix(line, "event: ")
				return
			}
		}
		got <- ""
	}()
	select {
	case ev := <-got:
		return ev
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE event")
		return ""
	}
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b := new(strings.Builder)
	_, _ = bufio.NewReader(resp.Body).WriteTo(b)
	return b.String()
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

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
