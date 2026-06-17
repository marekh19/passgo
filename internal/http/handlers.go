package http

import (
	"net/http"
	"strconv"
	"strings"
)

// handleLanding shows the create form. Joining is just navigating to a game URL.
func (s *Server) handleLanding(w http.ResponseWriter, r *http.Request) {
	renderLanding(w)
}

// handleGamePage shows the lobby/player view to members, else a join form.
// Reads come straight from store -- pure display data, no business rule to
// protect, so no service layer sits in between.
func (s *Server) handleGamePage(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	ses, err := s.store.GetSession(r.Context(), code)
	if err != nil {
		fail(w, err) // sql.ErrNoRows -> 404
		return
	}
	players, err := s.store.ListPlayers(r.Context(), code)
	if err != nil {
		fail(w, err)
		return
	}

	// Member iff the cookie verifies AND its player id is in this session.
	// Codes are reused after a sweep, so a valid-looking old cookie might not
	// belong to the current game -- the membership scan covers that.
	me := ""
	if id, ok := s.auth.Verify(r, code); ok {
		for _, p := range players {
			if p.ID == id {
				me = id
				break
			}
		}
	}
	if me == "" {
		renderJoin(w, ses)
		return
	}
	renderGame(w, ses, players, me)
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
	redirect(w, r, gameURL(code))
}
