package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/marekh19/passgo/internal/views/pages"
)

// handleLanding shows the create form. Joining is just navigating to a game URL.
func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	render(w, r, pages.Landing())
}

func (s *Server) handleJoinLookup(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("code")))
	if len(code) != 4 {
		http.Error(w, "game code must have four characters", http.StatusBadRequest)
		return
	}
	for _, ch := range code {
		if ch < 'A' || ch > 'Z' {
			if ch < '2' || ch > '9' {
				http.Error(w, "game code must contain letters or digits", http.StatusBadRequest)
				return
			}
		}
	}
	http.Redirect(w, r, gameURL(code), http.StatusSeeOther)
}

// handleGamePage shows the lobby/player view to members, else a join form.
// Reads come straight from store -- pure display data, no business rule to
// protect, so no service layer sits in between.
func (s *Server) handleGamePage(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	m, status, err := s.authenticatedMember(r.Context(), r, code)
	if err != nil {
		if status == http.StatusUnauthorized {
			// Unknown/non-member page visits see the join form; live endpoints are stricter.
			ses, getErr := s.store.GetSession(r.Context(), code)
			if getErr != nil {
				fail(w, getErr)
				return
			}
			render(w, r, pages.Join(ses.Code, ses.Started))
			return
		}
		fail(w, err) // sql.ErrNoRows -> 404
		return
	}
	if !m.session.Started {
		render(w, r, pages.Lobby(lobbyView(m)))
		return
	}
	view, err := s.playerView(r.Context(), m)
	if err != nil {
		fail(w, err)
		return
	}
	render(w, r, pages.Player(view))
}

// handleCreate makes a game, signs the current creator in as admin, redirects to it.
func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	ses, admin, err := s.sessions.Create(r.Context(), name)
	if err != nil {
		fail(w, err)
		return
	}
	s.auth.IssueCookie(w, ses.Code, admin.ID)
	redirect(w, r, gameURL(ses.Code))
}

// handleJoin enrolls a new player in a not-yet-started game and signs them in.
func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	p, err := s.sessions.Join(r.Context(), code, name)
	if err != nil {
		fail(w, err)
		return
	}
	s.live.publish(code, liveChanged)
	s.auth.IssueCookie(w, code, p.ID)
	redirect(w, r, gameURL(code))
}

// handleStart deals the opening cash. Admin-only -- the manager checks the
// caller's ID against the session admin and returns ErrNotAdmin otherwise.
func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	playerID, ok := s.auth.Verify(r, code)
	if !ok {
		http.Error(w, "not a member of this game", http.StatusUnauthorized)
		return
	}
	if err := s.sessions.Start(r.Context(), code, playerID); err != nil {
		fail(w, err)
		return
	}
	s.live.publish(code, liveStarted)
	redirect(w, r, gameURL(code))
}

// handleTransfer posts one money movement. Requires a valid session cookie
// (you're at the table); from/to come from the form. Stopping a player from
// moving someone else's money is a later anti-cheat refinement -- v1 trusts the
// table, like paper Monopoly.
func (s *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if _, ok := s.auth.Verify(r, code); !ok {
		http.Error(w, "not a member of this game", http.StatusUnauthorized)
		return
	}

	amount, err := strconv.ParseUint(r.FormValue("amount"), 10, 64)
	if err != nil {
		http.Error(w, "amount must be a whole number", http.StatusBadRequest)
		return
	}
	tcode, err := strconv.ParseUint(r.FormValue("code"), 10, 16)
	if err != nil {
		http.Error(w, "code must be a number", http.StatusBadRequest)
		return
	}

	if _, err := s.transfers.Execute(r.Context(), code, r.FormValue("from"), r.FormValue("to"), amount, uint16(tcode)); err != nil {
		fail(w, err)
		return
	}
	s.live.publish(code, liveChanged)
	redirect(w, r, gameURL(code))
}
