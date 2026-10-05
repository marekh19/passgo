# Pass Go

> Pass Go. Written in Go. Skip the paper money.

<center>
  <img alt="Pass Go Illustration" src="https://cdn.marekhonzal.com/passgo/passgo-illustration.webp" style="width: 100%; max-width: 500px;"/>
</center>

Mobile-first web app that replaces Monopoly's paper money. Players join a session by code, see live balances, and tap to move cash. The board, dice, and deeds stay on the table.

## Stack

Go · TigerBeetle (money) · SQLite (session metadata) · templ + Tailwind + HTMX/SSE · Docker + Coolify

## Status

In development. The core create, join, start, and transfer flow works with manual page refreshes. The current UI is temporary. See [`docs/`](docs/) for the original briefs and implementation plan.

## Run locally

Requires Go, Docker, Task, and the Tailwind standalone CLI.

```sh
task tb:up
task app:run
```

Open <http://localhost:8080>. Create a game, share its code, and have other players join through the home page. The host starts the game. Refresh the game page to see transfers made by other players. The local SQLite database is stored in `dev.db`.

---

Not affiliated with or endorsed by Hasbro. *Monopoly* is a trademark of Hasbro, Inc.
