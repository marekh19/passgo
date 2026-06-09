package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// issue runs IssueCookie into a recorder and returns the resulting cookie, so a
// test can replay it onto a request. Fails the test if no cookie was set.
func issue(t *testing.T, a Authenticator, code, playerID string) *http.Cookie {
	t.Helper()
	rec := httptest.NewRecorder()
	a.IssueCookie(rec, code, playerID)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("IssueCookie set %d cookies, want 1", len(cookies))
	}
	return cookies[0]
}

// requestWith builds a GET carrying the given cookies.
func requestWith(cookies ...*http.Cookie) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range cookies {
		r.AddCookie(c)
	}
	return r
}

func TestRoundTrip(t *testing.T) {
	a := New([]byte("test-secret"), false)
	c := issue(t, a, "ABCD", "player-1")

	// the cookie is named per-session and carries the signed identity
	if c.Name != "pg_ABCD" {
		t.Errorf("cookie name = %q, want pg_ABCD", c.Name)
	}
	if c.Value == "player-1" {
		t.Error("cookie value is the bare player ID -- not signed")
	}

	got, ok := a.Verify(requestWith(c), "ABCD")
	if !ok {
		t.Fatal("Verify rejected a freshly issued cookie")
	}
	if got != "player-1" {
		t.Errorf("Verify returned %q, want player-1", got)
	}
}

func TestCookieAttributes(t *testing.T) {
	a := New([]byte("test-secret"), true) // secure=true (behind TLS)
	c := issue(t, a, "ABCD", "player-1")

	if !c.HttpOnly {
		t.Error("cookie should be HttpOnly")
	}
	if !c.Secure {
		t.Error("cookie should be Secure when constructed with secure=true")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", c.SameSite)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, want /", c.Path)
	}
}

func TestVerifyMissingCookie(t *testing.T) {
	a := New([]byte("test-secret"), false)
	if _, ok := a.Verify(requestWith(), "ABCD"); ok {
		t.Error("Verify accepted a request with no cookie")
	}
}

func TestVerifyTampered(t *testing.T) {
	a := New([]byte("test-secret"), false)
	c := issue(t, a, "ABCD", "player-1")

	// flip the carried player ID but keep the original signature
	_, sig, _ := strings.Cut(c.Value, ".")
	c.Value = "player-2." + sig

	if _, ok := a.Verify(requestWith(c), "ABCD"); ok {
		t.Error("Verify accepted a cookie whose player ID was swapped")
	}
}

func TestVerifyWrongSession(t *testing.T) {
	a := New([]byte("test-secret"), false)
	c := issue(t, a, "ABCD", "player-1")

	// replay ABCD's value under another session's cookie name. The code is part
	// of the signed payload, so the signature no longer matches.
	c.Name = "pg_EFGH"
	if _, ok := a.Verify(requestWith(c), "EFGH"); ok {
		t.Error("Verify accepted a cookie replayed under a different session")
	}
}

func TestVerifyWrongSecret(t *testing.T) {
	issuer := New([]byte("secret-A"), false)
	other := New([]byte("secret-B"), false) // e.g. server restarted with a new key

	c := issue(t, issuer, "ABCD", "player-1")
	if _, ok := other.Verify(requestWith(c), "ABCD"); ok {
		t.Error("Verify accepted a cookie signed with a different secret")
	}
}

func TestVerifyMalformed(t *testing.T) {
	a := New([]byte("test-secret"), false)
	c := issue(t, a, "ABCD", "player-1")
	c.Value = "no-dot-here" // missing the "<id>.<sig>" structure

	if _, ok := a.Verify(requestWith(c), "ABCD"); ok {
		t.Error("Verify accepted a value with no signature separator")
	}
}
