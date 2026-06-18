package http

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	return fm, fs, fst, a, New(fst, fm, fs, a).Handler()
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
