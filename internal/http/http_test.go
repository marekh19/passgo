package http

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/marekh19/passgo/internal/auth"
	"github.com/marekh19/passgo/internal/session"
	"github.com/marekh19/passgo/internal/store"
	"github.com/marekh19/passgo/internal/tb"
)

// --- fakes for the slices http drives. auth is NOT faked: we use the real
// signer so the cookie genuinely round-trips through the handlers. ---

type fakeManager struct {
	createSes    store.Session
	createPlayer store.Player
	createErr    error
	createCalls  int
	createdName  string

	joinPlayer store.Player
	joinErr    error

	startErr   error
	startCalls int
	startCode  string
	startAdmin string
}

func (m *fakeManager) Create(_ context.Context, adminName string) (store.Session, store.Player, error) {
	m.createCalls++
	m.createdName = adminName
	return m.createSes, m.createPlayer, m.createErr
}
func (m *fakeManager) Join(_ context.Context, _, playerName string) (store.Player, error) {
	return m.joinPlayer, m.joinErr
}
func (m *fakeManager) Start(_ context.Context, code, adminID string) error {
	m.startCalls++
	m.startCode, m.startAdmin = code, adminID
	return m.startErr
}

type fakeService struct {
	err    error
	calls  int
	from   string
	to     string
	amount uint64
	code   uint16
}

type fakeBalanceReader struct {
	balances map[tb.Uint128]tb.Balance
	err      error
}

func (r *fakeBalanceReader) Balances(context.Context, []tb.Uint128) (map[tb.Uint128]tb.Balance, error) {
	return r.balances, r.err
}

func (s *fakeService) Execute(_ context.Context, _, from, to string, amount uint64, code uint16) (tb.Uint128, error) {
	s.calls++
	s.from, s.to, s.amount, s.code = from, to, amount, code
	return tb.Uint128{}, s.err
}

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

// unused by http -- present only to satisfy store.Store:
func (s *fakeStore) CreateSession(context.Context, store.Session) error  { return nil }
func (s *fakeStore) TouchSession(context.Context, string, int64) error   { return nil }
func (s *fakeStore) MarkStarted(context.Context, string) error           { return nil }
func (s *fakeStore) CreatePlayer(context.Context, store.Player) error    { return nil }
func (s *fakeStore) MaxLedgerID(context.Context) (uint32, error)         { return 0, nil }
func (s *fakeStore) SweepInactive(context.Context, int64) (int64, error) { return 0, nil }
func (s *fakeStore) Close() error                                        { return nil }

func newTestServer() (*fakeManager, *fakeService, *fakeStore, auth.Authenticator, http.Handler) {
	fm := &fakeManager{}
	fs := &fakeService{}
	fst := &fakeStore{}
	a := auth.New([]byte("test-secret"), false)
	return fm, fs, fst, a, New(fst, fm, fs, a, &fakeBalanceReader{}).Handler()
}

// cookieFor mints a valid signed cookie via the real authenticator, so tests can
// act as an authenticated player without going through create/join first.
func cookieFor(a auth.Authenticator, code, playerID string) *http.Cookie {
	rec := httptest.NewRecorder()
	a.IssueCookie(rec, code, playerID)
	return rec.Result().Cookies()[0]
}

func postForm(h http.Handler, target, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCreateIssuesCookieAndRedirects(t *testing.T) {
	fm, _, _, _, h := newTestServer()
	fm.createSes = store.Session{Code: "ABCD"}
	fm.createPlayer = store.Player{ID: "alice-id"}

	rec := postForm(h, "/api/sessions", "name=Alice")

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/sessions/ABCD" {
		t.Errorf("Location = %q, want /sessions/ABCD", loc)
	}
	if fm.createdName != "Alice" {
		t.Errorf("Create got name %q, want Alice", fm.createdName)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "pg_ABCD" {
		t.Fatalf("want one pg_ABCD cookie, got %+v", cookies)
	}
}

func TestCreateRejectsEmptyName(t *testing.T) {
	fm, _, _, _, h := newTestServer()

	rec := postForm(h, "/api/sessions", "name=")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if fm.createCalls != 0 {
		t.Error("Create must not run with an empty name")
	}
}

// The cookie issued by create must authenticate a later action and carry the
// same identity into the manager -- the core auth<->http contract.
func TestCreateThenStartCarriesIdentity(t *testing.T) {
	fm, _, _, _, h := newTestServer()
	fm.createSes = store.Session{Code: "ABCD"}
	fm.createPlayer = store.Player{ID: "alice-id"}

	createRec := postForm(h, "/api/sessions", "name=Alice")
	cookies := createRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("create issued no cookie")
	}

	startReq := httptest.NewRequest(http.MethodPost, "/api/sessions/ABCD/start", nil)
	for _, c := range cookies {
		startReq.AddCookie(c)
	}
	startRec := httptest.NewRecorder()
	h.ServeHTTP(startRec, startReq)

	if startRec.Code != http.StatusSeeOther {
		t.Fatalf("start status = %d, want 303 (body: %s)", startRec.Code, startRec.Body)
	}
	if fm.startCode != "ABCD" || fm.startAdmin != "alice-id" {
		t.Errorf("Start got (%q, %q), want (ABCD, alice-id) -- cookie identity not carried", fm.startCode, fm.startAdmin)
	}
}

func TestStartRequiresCookie(t *testing.T) {
	fm, _, _, _, h := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/ABCD/start", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if fm.startCalls != 0 {
		t.Error("Start must not run without a valid cookie")
	}
}

func TestStartNotAdminMapsTo403(t *testing.T) {
	fm, _, _, a, h := newTestServer()
	fm.startErr = session.ErrNotAdmin

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/ABCD/start", nil)
	req.AddCookie(cookieFor(a, "ABCD", "bob-id"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

// A POST-only API route must reject a GET with 405, not silently match.
func TestStartRejectsWrongMethod(t *testing.T) {
	_, _, _, _, h := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/ABCD/start", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestGamePage(t *testing.T) {
	members := []store.Player{
		{ID: "alice-id", Name: "Alice"},
		{ID: "bob-id", Name: "Bob"},
	}
	cases := []struct {
		name         string
		getErr       error
		cookieID     string // "" = no cookie
		wantStatus   int
		wantContains string
	}{
		{"unknown session -> 404", sql.ErrNoRows, "", http.StatusNotFound, ""},
		{"no cookie -> join form", nil, "", http.StatusOK, "Join game ABCD"},
		{"member -> game page", nil, "alice-id", http.StatusOK, "Alice"},
		// valid signature but the id isn't in this session (stale cookie after a
		// code was reused post-sweep) -> not a member.
		{"valid cookie, not a member -> join form", nil, "ghost-id", http.StatusOK, "Join game ABCD"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, fst, a, h := newTestServer()
			fst.getErr = c.getErr
			fst.ses = store.Session{Code: "ABCD", AdminID: "alice-id"}
			fst.players = members

			req := httptest.NewRequest(http.MethodGet, "/sessions/ABCD", nil)
			if c.cookieID != "" {
				req.AddCookie(cookieFor(a, "ABCD", c.cookieID))
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, c.wantStatus)
			}
			if c.wantContains != "" && !strings.Contains(rec.Body.String(), c.wantContains) {
				t.Errorf("body missing %q:\n%s", c.wantContains, rec.Body.String())
			}
		})
	}
}

func TestJoinLookup(t *testing.T) {
	_, _, _, _, handler := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/join?code=ab2d", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/sessions/AB2D" {
		t.Fatalf("join lookup: status %d, location %q", rec.Code, rec.Header().Get("Location"))
	}

	for _, code := range []string{"A/B", "ABCD/", ""} {
		req := httptest.NewRequest(http.MethodGet, "/join?code="+code, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("code %q: status %d, want 400", code, rec.Code)
		}
	}
}

func TestStartedGamePageShowsBalancesAndActions(t *testing.T) {
	manager := &fakeManager{}
	transfers := &fakeService{}
	st := &fakeStore{ses: store.Session{Code: "ABCD", AdminID: "alice-id", Started: true}}
	authn := auth.New([]byte("test-secret"), false)
	aliceAccount, bobAccount := tb.NewID(), tb.NewID()
	st.players = []store.Player{
		{ID: "alice-id", Name: "Alice", AcctID: aliceAccount.Bytes()},
		{ID: "bob-id", Name: "Bob", AcctID: bobAccount.Bytes()},
	}
	reader := &fakeBalanceReader{balances: map[tb.Uint128]tb.Balance{
		aliceAccount: {CreditsPosted: 1500, DebitsPosted: 200},
		bobAccount:   {CreditsPosted: 1600},
	}}
	handler := New(st, manager, transfers, authn, reader).Handler()

	req := httptest.NewRequest(http.MethodGet, "/sessions/ABCD", nil)
	req.AddCookie(cookieFor(authn, "ABCD", "alice-id"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	for _, text := range []string{"Alice", "$1,300", "Bob", "$1,600", "Collect $200", "Pay player", "Pay bank", "Collect from bank"} {
		if !strings.Contains(rec.Body.String(), text) {
			t.Errorf("body missing %q", text)
		}
	}
	if strings.Contains(rec.Body.String(), "Transfer</h2>") {
		t.Error("page still shows the raw transfer form")
	}
}

func TestStartedGamePageFailsWhenBalanceIsMissing(t *testing.T) {
	st := &fakeStore{ses: store.Session{Code: "ABCD", Started: true}}
	account := tb.NewID()
	st.players = []store.Player{{ID: "alice-id", Name: "Alice", AcctID: account.Bytes()}}
	authn := auth.New([]byte("test-secret"), false)
	handler := New(st, &fakeManager{}, &fakeService{}, authn, &fakeBalanceReader{balances: map[tb.Uint128]tb.Balance{}}).Handler()
	req := httptest.NewRequest(http.MethodGet, "/sessions/ABCD", nil)
	req.AddCookie(cookieFor(authn, "ABCD", "alice-id"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestLobbyShowsStartOnlyToAdmin(t *testing.T) {
	for _, tc := range []struct {
		name         string
		playerID     string
		wantText     string
		unwantedText string
	}{
		{"admin", "alice-id", "Start game", "Waiting for host"},
		{"player", "bob-id", "Waiting for host", "Start game"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, st, authn, handler := newTestServer()
			st.ses = store.Session{Code: "ABCD", AdminID: "alice-id"}
			st.players = []store.Player{
				{ID: "alice-id", Name: "Alice"},
				{ID: "bob-id", Name: "Bob"},
			}

			req := httptest.NewRequest(http.MethodGet, "/sessions/ABCD", nil)
			req.AddCookie(cookieFor(authn, "ABCD", tc.playerID))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			body := rec.Body.String()
			for _, text := range []string{"Share this code", "ABCD", "Alice", "Bob", tc.wantText} {
				if !strings.Contains(body, text) {
					t.Errorf("body missing %q", text)
				}
			}
			if strings.Contains(body, tc.unwantedText) {
				t.Errorf("body unexpectedly contains %q", tc.unwantedText)
			}
		})
	}
}

func TestTransferForwardsParsedArgs(t *testing.T) {
	_, fs, _, a, h := newTestServer()

	rec := postForm(h, "/api/sessions/ABCD/transfers",
		"from=bob-id&to=alice-id&amount=250&code=10",
		cookieFor(a, "ABCD", "bob-id"))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 (body: %s)", rec.Code, rec.Body)
	}
	if fs.calls != 1 {
		t.Fatalf("Execute calls = %d, want 1", fs.calls)
	}
	if fs.from != "bob-id" || fs.to != "alice-id" || fs.amount != 250 || fs.code != 10 {
		t.Errorf("Execute got from=%q to=%q amount=%d code=%d", fs.from, fs.to, fs.amount, fs.code)
	}
}

func TestTransferRejectsBadAmount(t *testing.T) {
	_, fs, _, a, h := newTestServer()

	rec := postForm(h, "/api/sessions/ABCD/transfers",
		"from=bob-id&to=alice-id&amount=lots&code=10",
		cookieFor(a, "ABCD", "bob-id"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if fs.calls != 0 {
		t.Error("Execute must not run with an unparseable amount")
	}
}

func TestLiveEndpointsRequireSessionCookieAndMembership(t *testing.T) {
	members := []store.Player{{ID: "alice-id", Name: "Alice"}}
	for _, tc := range []struct {
		name   string
		path   string
		getErr error
		cookie string
		want   int
	}{
		{"missing stream session", "/sessions/ABCD/events", sql.ErrNoRows, "alice-id", http.StatusNotFound},
		{"stream no cookie", "/sessions/ABCD/events", nil, "", http.StatusUnauthorized},
		{"stream stale cookie", "/sessions/ABCD/events", nil, "ghost-id", http.StatusUnauthorized},
		{"stream other-game cookie", "/sessions/ABCD/events", nil, "other-game", http.StatusUnauthorized},
		{"roster no cookie", "/sessions/ABCD/lobby/roster", nil, "", http.StatusUnauthorized},
		{"balances stale cookie", "/sessions/ABCD/balances", nil, "ghost-id", http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, st, authn, handler := newTestServer()
			st.getErr = tc.getErr
			st.ses = store.Session{Code: "ABCD", AdminID: "alice-id", Started: true}
			st.players = members
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.cookie == "other-game" {
				req.AddCookie(cookieFor(authn, "WXYZ", "alice-id"))
			} else if tc.cookie != "" {
				req.AddCookie(cookieFor(authn, "ABCD", tc.cookie))
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestPublishOnlyAfterSuccessfulMutations(t *testing.T) {
	fm := &fakeManager{joinPlayer: store.Player{ID: "bob-id"}}
	fs := &fakeService{}
	st := &fakeStore{}
	authn := auth.New([]byte("test-secret"), false)
	srv := New(st, fm, fs, authn, &fakeBalanceReader{})
	_, events := srv.live.subscribe("ABCD")
	_, otherEvents := srv.live.subscribe("WXYZ")

	rec := postForm(srv.Handler(), "/api/sessions/ABCD/join", "name=Bob")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("join status = %d", rec.Code)
	}
	wantEvent(t, events, liveChanged)
	wantNoEvent(t, otherEvents)

	fm.joinErr = sql.ErrNoRows
	rec = postForm(srv.Handler(), "/api/sessions/ABCD/join", "name=Bob")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("failed join status = %d", rec.Code)
	}
	wantNoEvent(t, events)

	fm.joinErr = nil
	fm.startErr = session.ErrNotAdmin
	rec = postForm(srv.Handler(), "/api/sessions/ABCD/start", "", cookieFor(authn, "ABCD", "alice-id"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("failed start status = %d", rec.Code)
	}
	wantNoEvent(t, events)

	fm.startErr = nil
	rec = postForm(srv.Handler(), "/api/sessions/ABCD/start", "", cookieFor(authn, "ABCD", "alice-id"))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("start status = %d", rec.Code)
	}
	wantEvent(t, events, liveStarted)

	fs.err = tb.ErrInsufficientFunds
	rec = postForm(srv.Handler(), "/api/sessions/ABCD/transfers", "from=alice-id&to=bob-id&amount=2000&code=10", cookieFor(authn, "ABCD", "alice-id"))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("failed transfer status = %d", rec.Code)
	}
	wantNoEvent(t, events)

	fs.err = nil
	rec = postForm(srv.Handler(), "/api/sessions/ABCD/transfers", "from=alice-id&to=bob-id&amount=200&code=10", cookieFor(authn, "ABCD", "alice-id"))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("transfer status = %d", rec.Code)
	}
	wantEvent(t, events, liveChanged)
}

func wantEvent(t *testing.T, ch <-chan liveEvent, want liveEvent) {
	t.Helper()
	select {
	case got := <-ch:
		if got != want {
			t.Fatalf("event = %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %q", want)
	}
}

func wantNoEvent(t *testing.T, ch <-chan liveEvent) {
	t.Helper()
	select {
	case got := <-ch:
		t.Fatalf("unexpected event %q", got)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestTransferReportsInsufficientFunds(t *testing.T) {
	_, transfers, _, authn, handler := newTestServer()
	transfers.err = tb.ErrInsufficientFunds
	rec := postForm(handler, "/api/sessions/ABCD/transfers",
		"from=alice-id&to=bob-id&amount=2000&code=10",
		cookieFor(authn, "ABCD", "alice-id"))
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "not enough money") {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
}
