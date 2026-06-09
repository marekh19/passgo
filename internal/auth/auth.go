// Package auth issues and verifies a per-session signed cookie that identifies a
// player. The cookie value is a "<playerID>.<sig>" where sig is the HMAC over the
// session code + playerID -- so a client can't forge a different identity, and a
// valid value can't be replayed under another session's cookie.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
)

type Authenticator interface {
	// IssueCookie sets the signed per-session cookie on the response.
	IssueCookie(w http.ResponseWriter, code, playerID string)
	// Verify reads the cookie for this session and returns the playerID if its
	// signature checks out. ok is false on missing/malformed/forged cookies.
	Verify(r *http.Request, code string) (playerID string, ok bool)
}

type authenticator struct {
	secret []byte
	secure bool // set the Secure flag (true behind TLS; false for dev http)
}

func New(secret []byte, secure bool) Authenticator {
	return &authenticator{secret: secret, secure: secure}
}

// cookieMaxAge keeps you logged into a game across browser restarts. A cookie
// outliving its session is harmless: handlers 404 once the session is swept.
const cookieMaxAge = 7 * 24 * 60 * 60 // seconds (7 days)

func cookieName(code string) string { return "pg_" + code }

// sign returns base64url(HMAC-SHA256(code || 0x00 || playerID)). The code is part
// of the signed payload, not just the cookie name, so a value can't be replayed
// under a different session's cookie.
func (a *authenticator) sign(code, playerID string) string {
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(code))
	mac.Write([]byte{0})
	mac.Write([]byte(playerID))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *authenticator) IssueCookie(w http.ResponseWriter, code, playerID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName(code),
		Value:    playerID + "." + a.sign(code, playerID),
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *authenticator) Verify(r *http.Request, code string) (string, bool) {
	c, err := r.Cookie(cookieName(code))
	if err != nil {
		return "", false // no cookie for this session
	}
	// playerID is hex, sig is base64url -- neither contains a dot, so Cut on the
	// first dot cleanly splits the two halves.
	playerID, sig, ok := strings.Cut(c.Value, ".")
	if !ok {
		return "", false
	}
	if !hmac.Equal([]byte(sig), []byte(a.sign(code, playerID))) {
		return "", false // forged or tampered
	}
	return playerID, true
}
