# 01 Live game updates

**Outcome:** People in the same game see joins, game start, and balance changes
without refreshing.

**Scope:** Add live updates to the current lobby and game pages. Keep the
existing visual design and POST/redirect/GET transfer forms. History, host
controls, redesign, persistent event replay, and cross-process fan-out are
separate work. This task targets **one app process** (the current deployment
model); SQLite and TigerBeetle remain the sources of truth.

## Behavior and protocol

- Use native browser `EventSource` and an authenticated `GET /sessions/{code}/events`
  SSE endpoint. Events are **invalidation signals**, not balance or player data:
  send `event: changed` after a join or transfer and `event: started` after a
  successful start. Emit valid SSE frames with a `data:` line (for example,
  `event: changed\ndata: 1\n\n`), so native EventSource dispatches them.
  Key subscriptions by session code; only that session's subscribers receive
  its notifications. An initial connection need not replay events or send an
  initial change event.
- Before opening a stream, check that the session exists, the signed cookie
  verifies **for that code**, and its player ID occurs in the session's current
  player list (the same membership rule used by `handleGamePage`). Return 404
  for an unknown session and 401 for any nonmember (missing/invalid, another
  game's cookie, or a valid but stale cookie after code reuse). Do not open a
  stream or reveal any game state to them. Apply the same checks to the HTML
  refresh endpoints below; neither a guessed code nor a signed cookie alone
  grants access.
- Publish only after the relevant service call has completed successfully:
  `Join` after the player is persisted, `Start` after starting cash is dealt
  **and** the session is marked started, `Execute` after the transfer is posted.
  The existing handlers are the mutation entry points; a shared in-process
  notifier owned by the HTTP server is sufficient. Do not publish for failed
  actions, and do not change their redirects or error responses. Creation has
  no other connected members to notify.
- `GET /sessions/{code}/lobby/roster` returns a server-rendered **roster
  fragment** for an authenticated member while the session is unstarted. It
  keeps the current names, `(you)` and host labels in a replaceable `#roster`
  container. If the session has started,
  return 409 (the lobby client navigates to `/sessions/{code}`); do not render
  stale lobby data.
- `GET /sessions/{code}/balances` returns server-rendered **balance and player
  list fragments** for an authenticated member of a started session, using the
  same SQLite player data and TigerBeetle balances as the existing game page.
  Include both the viewer's balance/player count (`#balance`) and all other
  players' names and balances (`#players`) in one response, replacing those two
  containers together. If still in the lobby, return 409. Reuse existing templ
  components/view-model assembly where practical, rather than maintaining two
  versions of balance calculation or escaping. A read failure must not replace
  good content with an error page.
- Lobby pages listen for `changed` and fetch/replace the roster fragment;
  `started` navigates to the canonical game URL. Game pages listen for
  `changed` and fetch/replace the balance and player-list fragments. Replace
  only these display regions, preserving in-progress transfer form values and
  focus. Fetch fresh state when the EventSource **opens or reopens**, even if
  no event arrived; this also detects a game that started while the lobby was
  disconnected (409 -> navigate). Coordinate overlapping refreshes so an older
  response cannot overwrite a newer snapshot. A normal GET of the page must
  always show current state without SSE.
- Show a small, readable connection warning when the stream disconnects;
  retain usable forms and the existing manual balance-refresh link. Hide the
  warning and fetch a new snapshot when connected again. Browser/EventSource
  automatic retry is sufficient; no persisted event IDs or replay log. On
  refresh returning 401/404 (membership/session no longer valid), navigate to
  the canonical session URL so the existing join/404 handling can take over.
  Avoid exposing balance/player HTML in event payloads. Use no-cache headers
  on snapshots and SSE, flush the stream, send periodic SSE comments as
  keepalives, and release subscribers when their request ends. Slow readers
  must not block successful game actions; coalesce redundant `changed` events,
  but never silently lose `started`: deliver it or close that stream so the
  reconnect snapshot detects the new phase.

## Starting points

- `internal/http/http.go` registers routes; `internal/http/handlers.go` renders
  the current page and handles the three POST actions.
- `internal/views/pages/lobby.templ`, `player.templ`, `balance.templ`, and
  `players_list.templ` contain the display regions; `internal/views/layout.templ`
  loads existing client assets. Generate changed templates with
  `go tool templ generate` and commit generated `*_templ.go` files.
- `internal/http/flow_test.go` covers the flow using real HTTP handlers and
  SQLite with a fake ledger. `internal/http/http_test.go` has handler/auth fakes;
  use real signed cookies and two independent browser clients in the live tests.
- `internal/session/manager.go` and `internal/transfer/transfer.go` identify
  when each mutation becomes successful. `internal/auth/auth.go` verifies
  per-game cookies; membership must also be checked against `ListPlayers`.
- Run project commands via `Taskfile.yaml` (`task test`, `task build`); after
  changing Go imports, run `go mod tidy`. Avoid editing generated store bindings.

## Done when

- Joining appears on another member's open lobby without reloading; `(you)`
  and host labels are correct for each viewer.
- Starting navigates every connected lobby member to the game page, where
  opening balances appear. Reconnecting a lobby after start also navigates.
- A successful player-to-player or Bank transfer updates the balances seen in
  two open browser sessions without wiping an unfinished form. Failed actions
  do not emit a change; another session's actions do not update this session.
- Nonmembers cannot open the SSE stream or fetch fragments, including no
  cookie, a forged/other-game cookie, and a valid cookie for a player no longer
  present (e.g. after code reuse). Missing sessions return 404.
- Reloading or reconnecting retrieves the current roster/balances without
  relying on missed events; a disconnected page shows the warning and remains
  usable, then resynchronizes and clears it on reconnect.
- Automated tests exercise real streaming (including cancellation/cleanup),
  publish-on-success and isolation, authenticated snapshots, start navigation
  behavior, and reconnection. Tests use bounded waits, not arbitrary sleeps.
  `task test` and `task build` pass.
- With `task tb:up` and `task app:run`, manually open two independent browser
  sessions: create/join, observe roster change, start, transfer in each
  direction, disconnect/reconnect one browser and confirm latest balances.
  Record checks that could not run in `docs/STATUS.md` and leave this task
  active until all completion checks pass.
