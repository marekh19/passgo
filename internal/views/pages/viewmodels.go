package pages

import (
	"strconv"
	"strings"
)

// PlayerBalance is one player's name + current whole-dollar balance for the live
// player view. The balance is the TigerBeetle net (credits - debits).
type PlayerBalance struct {
	ID      string
	Name    string
	Balance int64
}

// PlayerView is everything the live player screen renders; your own balance card
// plus the other players. Assembled by the http layer from store (names) + tb
// (balances) -- the one place those two sources are combined for display.
type PlayerView struct {
	Code        string
	PlayerCount int
	IsAdmin     bool
	Me          PlayerBalance
	Others      []PlayerBalance
}

// LobbyMember is one joined player in the pre-game lobby (no balance yet).
type LobbyMember struct {
	ID   string
	Name string
}

// LobbyView is the pre-game lobby: the share code, who's joined so far, and
// whether the viewer is the admin (who gets the Start button).
type LobbyView struct {
	Code    string
	Members []LobbyMember
	Me      string // viewer's player id, to tag "(you)"
	AdminID string // session admin's id, to tag the host
	IsAdmin bool   // viewer is the admin -> show "Start", else "Waiting..."
}

// dollars formats whole dollars with thousands separators: 1500 -> "$1,500",
// -200 -> "-$200", 0 -> "$0". Used by the balance components.
func dollars(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	digits := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, c := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	out := "$" + b.String()
	if neg {
		out = "-" + out
	}
	return out
}

func playerCountLabel(n int) string {
	if n == 1 {
		return "1 player"
	}
	return strconv.Itoa(n) + " players"
}
