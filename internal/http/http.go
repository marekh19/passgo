// Package http wires the router and handlers. It depends only on the other
// slices' interfaces -- session/transfer for actions, store for read-only page
// data, auth for identity cookie. It never touches tb or SQLite directly.
package http

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/marekh19/passgo/internal/auth"
	"github.com/marekh19/passgo/internal/session"
	"github.com/marekh19/passgo/internal/store"
	"github.com/marekh19/passgo/internal/transfer"
)

type Server struct {
	store     store.Store
	sessions  session.Manager
	transfers transfer.Service
	auth      auth.Authenticator
}

func New(st store.Store, sessions session.Manager, transfers transfer.Service, authn auth.Authenticator) *Server {
	return &Server{store: st, sessions: sessions, transfers: transfers, auth: authn}
}

// Handler builds the router. net/http's method+path patterns do the routing;
// {$} anchors "/" so the landing route doesn't swallow every path.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// UI -- server-rendered pages
	mux.HandleFunc("GET /{$}", s.handleLanding)
	mux.HandleFunc("GET /sessions/{code}", s.handleGamePage)

	// API -- form POSTs; reply with a 303 to the game page
	mux.HandleFunc("POST /api/sessions", s.handleCreate)
	mux.HandleFunc("POST /api/sessions/{code}/join", s.handleJoin)
	mux.HandleFunc("POST /api/sessions/{code}/start", s.handleStart)
	mux.HandleFunc("POST /api/sessions/{code}/transfers", s.handleTransfer)

	return mux
}

// gameURL is the canonical page for a session.
func gameURL(code string) string { return "/sessions/" + code }

// redirect issues a 303 See Other -- the POST/redirect/GET pattern, so a browser
// refresh re-GETs the page instead of re-posting the action.
func redirect(w http.ResponseWriter, r *http.Request, url string) {
	http.Redirect(w, r, url, http.StatusSeeOther)
}

// fail maps a slice error to an HTTP status + message. Unknown errors (tb
// rejections, infra) become 500 for now.
// TODO: typed tb error -> 422 + toast for insufficient funds etc.
func fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		http.Error(w, "session not found", http.StatusNotFound)
	case errors.Is(err, session.ErrNotAdmin):
		http.Error(w, "only the admin can do that", http.StatusForbidden)
	case errors.Is(err, session.ErrAlreadyStarted), errors.Is(err, transfer.ErrNotStarted):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, session.ErrCodeExhausted):
		http.Error(w, "could not allocate a game code, try again", http.StatusServiceUnavailable)
	case errors.Is(err, transfer.ErrBadAmount),
		errors.Is(err, transfer.ErrBadCode),
		errors.Is(err, transfer.ErrBadDirection),
		errors.Is(err, transfer.ErrSelfTransfer),
		errors.Is(err, transfer.ErrUnknownPlayer):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "something went wrong", http.StatusInternalServerError)
	}
}
