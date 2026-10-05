package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/marekh19/passgo/internal/store"
	"github.com/marekh19/passgo/internal/tb"
	"github.com/marekh19/passgo/internal/views/pages"
)

type memberView struct {
	session store.Session
	players []store.Player
	me      string
}

func (s *Server) authenticatedMember(ctx context.Context, r *http.Request, code string) (memberView, int, error) {
	ses, err := s.store.GetSession(ctx, code)
	if err != nil {
		return memberView{}, http.StatusNotFound, err
	}
	players, err := s.store.ListPlayers(ctx, code)
	if err != nil {
		return memberView{}, http.StatusInternalServerError, err
	}
	id, ok := s.auth.Verify(r, code)
	if !ok {
		return memberView{}, http.StatusUnauthorized, fmt.Errorf("not a member")
	}
	for _, p := range players {
		if p.ID == id {
			return memberView{session: ses, players: players, me: id}, 0, nil
		}
	}
	return memberView{}, http.StatusUnauthorized, fmt.Errorf("not a member")
}

func lobbyView(m memberView) pages.LobbyView {
	members := make([]pages.LobbyMember, 0, len(m.players))
	for _, p := range m.players {
		members = append(members, pages.LobbyMember{ID: p.ID, Name: p.Name})
	}
	return pages.LobbyView{Code: m.session.Code, Members: members, Me: m.me, AdminID: m.session.AdminID, IsAdmin: m.me == m.session.AdminID}
}

func (s *Server) playerView(ctx context.Context, m memberView) (pages.PlayerView, error) {
	ids := make([]tb.Uint128, 0, len(m.players))
	for _, p := range m.players {
		ids = append(ids, tb.IDFromBytes(p.AcctID))
	}
	balances, err := s.balances.Balances(ctx, ids)
	if err != nil {
		return pages.PlayerView{}, err
	}
	view := pages.PlayerView{
		Code:        m.session.Code,
		PlayerCount: len(m.players),
		IsAdmin:     m.me == m.session.AdminID,
		Others:      make([]pages.PlayerBalance, 0, len(m.players)-1),
	}
	for _, p := range m.players {
		balance, ok := balances[tb.IDFromBytes(p.AcctID)]
		if !ok {
			return pages.PlayerView{}, fmt.Errorf("balance missing for player %s", p.ID)
		}
		player := pages.PlayerBalance{ID: p.ID, Name: p.Name, Balance: balance.Net()}
		if p.ID == m.me {
			view.Me = player
		} else {
			view.Others = append(view.Others, player)
		}
	}
	return view, nil
}

func noCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
}

func (s *Server) handleRosterFragment(w http.ResponseWriter, r *http.Request) {
	m, status, err := s.authenticatedMember(r.Context(), r, r.PathValue("code"))
	if err != nil {
		http.Error(w, http.StatusText(status), status)
		return
	}
	if m.session.Started {
		http.Error(w, "game started", http.StatusConflict)
		return
	}
	noCache(w)
	render(w, r, pages.Roster(lobbyView(m)))
}

func (s *Server) handleBalancesFragment(w http.ResponseWriter, r *http.Request) {
	m, status, err := s.authenticatedMember(r.Context(), r, r.PathValue("code"))
	if err != nil {
		http.Error(w, http.StatusText(status), status)
		return
	}
	if !m.session.Started {
		http.Error(w, "game not started", http.StatusConflict)
		return
	}
	pv, err := s.playerView(r.Context(), m)
	if err != nil {
		fail(w, err)
		return
	}
	noCache(w)
	render(w, r, pages.BalanceAndPlayers(pv))
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	m, status, err := s.authenticatedMember(r.Context(), r, r.PathValue("code"))
	if err != nil {
		http.Error(w, http.StatusText(status), status)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	noCache(w)
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Connection", "keep-alive")

	id, ch := s.live.subscribe(m.session.Code)
	defer s.live.unsubscribe(m.session.Code, id)
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "event: %s\ndata: 1\n\n", ev)
			flusher.Flush()
		case <-keepalive.C:
			_, _ = fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
