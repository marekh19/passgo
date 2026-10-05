# Pass Go

> Pass Go. Written in Go. Skip the paper money.

<center>
  <img alt="Pass Go Illustration" src="https://cdn.marekhonzal.com/passgo/passgo-illustration.webp" style="width: 100%; max-width: 500px;"/>
</center>

Mobile-first web app that replaces Monopoly's paper money. Players join a session by code, see live balances, and tap to move cash. The board, dice, and deeds stay on the table.

## Stack

Go · TigerBeetle (money) · SQLite (session metadata) · templ · Tailwind

## Status

In development. See [project status](docs/STATUS.md) for current behavior and the
active task, and [product intent](docs/PRODUCT.md) for the release goal. The
current UI is temporary.

## Run locally

Requires Go, Docker, Task, and the Tailwind standalone CLI.

```sh
task tb:up
task app:run
```

Open <http://localhost:8080>. Create a game, share its code, and have other players join through the home page. The host starts the game. Refresh the game page to see transfers made by other players. The local SQLite database is stored in `dev.db`.

Run `task test` to check the Go code. For live reload, use `task dev` after starting TigerBeetle and open <http://localhost:8090>.

---

Not affiliated with or endorsed by Hasbro. *Monopoly* is a trademark of Hasbro, Inc.
