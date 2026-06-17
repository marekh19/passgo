package http

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"

	"github.com/marekh19/passgo/internal/store"
)

// --- Phase 5 placeholder HTML. Replaced wholesale in Phase 6 (templ + Tailwind);
// kept dead-simple, just enough to drive the flow by browser or curl. ---

func writeHTML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, body)
}

func renderLanding(w http.ResponseWriter) {
	writeHTML(w, `<!doctype html>
<title>Pass Go</title>
<h1>Pass Go</h1>
<form method="post" action="/api/sessions">
  <input name="name" placeholder="your name" required>
  <button>Create game</button>
</form>
<p>To join a game, open <code>/sessions/&lt;code&gt;</code>.</p>`)
}

func renderJoin(w http.ResponseWriter, ses store.Session) {
	if ses.Started {
		writeHTML(w, fmt.Sprintf(`<!doctype html>
<title>%[1]s</title>
<h1>Game %[1]s already started</h1>
<p>You can't join a game in progress.</p>`, ses.Code))
		return
	}
	writeHTML(w, fmt.Sprintf(`<!doctype html>
<title>Join %[1]s</title>
<h1>Join game %[1]s</h1>
<form method="post" action="/api/sessions/%[1]s/join">
  <input name="name" placeholder="your name" required>
  <button>Join</button>
</form>`, ses.Code))
}

func renderGame(w http.ResponseWriter, ses store.Session, players []store.Player, me string) {
	var b strings.Builder
	status := "lobby"
	if ses.Started {
		status = "live"
	}
	fmt.Fprintf(&b, "<!doctype html>\n<title>Game %[1]s</title>\n<h1>Game %[1]s</h1>\n<p>Status: <b>%[2]s</b></p>\n", ses.Code, status)

	// Player IDs are shown so manual/curl testing can paste them into the
	// transfer form's from/to fields.
	b.WriteString("<h2>Players</h2>\n<ul>\n")
	for _, p := range players {
		tag := ""
		if p.ID == me {
			tag += " (you)"
		}
		if p.ID == ses.AdminID {
			tag += " [admin]"
		}
		fmt.Fprintf(&b, "  <li>%s%s &mdash; <code>%s</code></li>\n", html.EscapeString(p.Name), tag, p.ID)
	}
	b.WriteString("</ul>\n")

	switch {
	case !ses.Started && me == ses.AdminID:
		fmt.Fprintf(&b, "<form method=\"post\" action=\"/api/sessions/%s/start\"><button>Start game</button></form>\n", ses.Code)
	case !ses.Started:
		b.WriteString("<p>Waiting for the admin to start the game&hellip;</p>\n")
	default:
		fmt.Fprintf(&b, `<h2>Transfer</h2>
<form method="post" action="/api/sessions/%s/transfers">
  <input name="from" placeholder="from: player id or 'bank'" required>
  <input name="to" placeholder="to: player id or 'bank'" required>
  <input name="amount" type="number" placeholder="amount" required>
  <input name="code" type="number" placeholder="code: 10 / 20-23 / 30-32" required>
  <button>Send</button>
</form>
`, ses.Code)
	}
	writeHTML(w, b.String())
}
