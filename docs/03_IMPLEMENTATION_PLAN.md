# Pass Go — Implementation Plan

> High-level, step-by-step build order. Each step has a clear "done when" so you know to move on. Aligned to the 3-evening schedule but phase-based so you can flex.

---

## Phase 0 — Prep (30 min, before Evening 1)

**Goal:** environment ready, repo initialized, you've read enough TigerBeetle docs that the data model clicks.

1. Install toolchain: Go 1.23+, Docker, Taskfile (`brew install go-task` or equiv), templ CLI, sqlc CLI, Tailwind standalone binary.
2. Skim TigerBeetle docs: "Data Modeling" and "Deployment" sections only. ~20 min.
3. `git init passgo`, push empty repo to GitHub. Add `.gitignore` for Go + `bin/` + `*.db` + `static/css/output.css`.
4. Buy `passgo.quest` (Porkbun or Cloudflare). Park it for now.

**Done when:** `go version`, `task --version`, `templ version`, `sqlc version`, `docker ps` all work.

---

## Phase 1 — Skeleton (Evening 1, hour 1)

**Goal:** project layout exists, `task dev` runs an empty Go server that responds with "hello."

1. Create the directory tree from the technical brief (`cmd/passgo`, `internal/{tb,store,session,transfer,broadcast,admin,auth,http,views}`, `static`, etc.).
2. Initialize `go.mod`: `go mod init github.com/<you>/passgo`.
3. Write `Taskfile.yml` with the `dev`, `app:run`, `build`, `test`, `templ:gen`, `tailwind:build` tasks.
4. Write `cmd/passgo/main.go` as a minimal `http.ListenAndServe` on `:8080` that returns "Pass Go" at `/`.
5. Add `Dockerfile` and `compose.yml` (just the app service for now; TigerBeetle in next phase).

**Done when:** `task app:run` serves "Pass Go" at `http://localhost:8080`.

---

## Phase 2 — TigerBeetle slice (Evening 1, hours 2–3)

**Goal:** `internal/tb` exposes a working `Client` interface; you can create accounts and execute transfers from Go.

1. Add TigerBeetle to `compose.yml`. Verify `task tb:up` starts it and exposes port 3000.
2. Write `internal/tb/tb.go` with the `Client` interface (see brief).
3. Write the `Uint128` helper (constructor from random, stringify, parse).
4. Implement `New(addr)` returning the concrete client wrapping the official `tigerbeetle-go` SDK.
5. Implement `CreateAccount`, `Transfer`, `Balances`. Skip `LinkedTransfer` and `History` for now — add when needed.
6. Write a smoke test in `tb_test.go` that spins up against a live TB (or skip if it's annoying — manual `curl`-equivalent via a temporary `main` is fine for now).

**Done when:** a small Go script (or temporary endpoint) creates two accounts, transfers $200 between them, and reads back correct balances.

**Stop condition:** if the TigerBeetle Go SDK fights you for >1h, read the official examples in the SDK repo. Don't move on until this slice works — everything else depends on it.

---

## Phase 3 — SQLite store slice (Evening 1, hour 4)

**Goal:** `internal/store` persists sessions and players.

1. Write `internal/store/schema.sql` (sessions + players tables from the brief). Enable WAL mode in the schema file.
2. Write `internal/store/queries.sql` with the queries you'll need: `CreateSession`, `GetSession`, `UpdateSessionLastActive`, `CreatePlayer`, `ListPlayersBySession`, `GetMaxLedgerID`, `SweepInactiveSessions`.
3. Configure `sqlc.yaml` to generate into `internal/store/generated/`.
4. Run `task sqlc` to generate query bindings.
5. Write `internal/store/store.go` with the `Store` interface wrapping the sqlc-generated code. Add a `New(path string)` constructor that opens the DB and applies the schema.
6. Add `task db:migrate` to apply the schema idempotently.

**Done when:** a quick test creates a session row, reads it back, creates a player, lists players for the session.

---

## Phase 4 — Session + transfer slices (Evening 2, hour 1)

**Goal:** business logic layer above tb + store. No HTTP yet.

1. Write `internal/session/session.go` with `Manager` interface: `Create`, `Join`, `Start`, `End`.
   - `Create`: short code generation, allocate next ledger ID from store, create Bank account in TigerBeetle, persist session row.
   - `Join`: create player row + player account in TigerBeetle (with `debits_must_not_exceed_credits`).
   - `Start`: mark session started, execute $1,500 transfer from Bank → each player.
2. Write `internal/transfer/transfer.go` with `Service` interface: `Execute(ctx, sessionCode, from, to, amount, code)`. Validates and posts to TigerBeetle.
3. Write `internal/admin/admin.go` with `Service` interface: `UndoLast`, `Adjust`. Undo = post reversing transfer; Adjust = code 99 transfer.
4. Hand-write tiny fakes for `tb.Client` and `store.Store` in each slice's test file. Test each slice in isolation.

**Done when:** unit tests pass for session creation, joining, starting (initial transfers), and a simple transfer execution.

---

## Phase 5 — HTTP layer + auth (Evening 2, hour 2)

**Goal:** routes exist, cookie-based identity works, but pages are minimal HTML.

1. Write `internal/auth/auth.go` with `Service`: `IssueCookie(playerID, sessionCode)`, `Verify(r *http.Request, sessionCode)`. HMAC-signed cookie.
2. Write `internal/http/http.go` with router setup. Use `net/http` ServeMux (Go 1.22+ supports path patterns natively).
3. Mount routes from the technical brief:
   - UI: `/`, `/sessions/{code}`, `/sessions/{code}/history`, `/sessions/{code}/events`
   - API: `/api/sessions`, `/api/sessions/{code}/join`, `/api/sessions/{code}/start`, `/api/sessions/{code}/transfers`, `/api/sessions/{code}/admin/{undo,adjust}`
4. Wire everything in `cmd/passgo/main.go`: construct slices in order (`tb.New` → `store.New` → `auth.New` → `session.New(tb, store)` → `transfer.New(...)` → `admin.New(...)` → `broadcast.New()` → `http.New(deps...)`).
5. Return placeholder HTML strings (no templ yet). Verify each route works with `curl`.

**Done when:** you can `curl -X POST /api/sessions` to create a session, then `curl /sessions/{code}` to see it, all with proper redirects + cookies.

---

## Phase 6 — Templ + Tailwind + templUI (Evening 2, hour 3)

**Goal:** real UI replaces the placeholder HTML.

1. Set up Tailwind: write `tailwind.config.js` scanning `**/*.templ`, write `input.css` with `@tailwind` directives + templUI imports.
2. Install templUI: follow their setup, vendor or `go get` the components you need (Button, Sheet, Input, Toast).
3. Write `internal/views/layout.templ` — base HTML, dark mode, mobile viewport meta, HTMX script tag, SSE extension.
4. Write the three pages:
   - `pages/landing.templ` — name + Create/Join buttons.
   - `pages/lobby.templ` — share code, players list (SSE), Start button (admin only).
   - `pages/player.templ` — balance card + players list + action buttons + history link.
5. Write the components: `balance.templ`, `players_list.templ`, `action_sheet.templ`, `history.templ`.
6. Run `task templ:gen` and `task tailwind:build` until everything renders.

**Done when:** all three pages look like the design brief on a phone-sized viewport. No SSE yet — manual refresh to see balance changes.

---

## Phase 7 — Broadcast + SSE (Evening 2, hour 4)

**Goal:** balances and history update live across all connected players.

1. Write `internal/broadcast/broadcast.go` with `Hub` interface: `Subscribe(sessionCode, playerID) (<-chan Event, cancel)`, `Publish(sessionCode, event)`.
2. Implement subscriber registry: `map[sessionCode]map[playerID]chan Event` guarded by `sync.RWMutex`. Buffered channels, non-blocking sends.
3. Write the SSE handler in `internal/http/sse.go`: long-lived response, `text/event-stream`, flush on each event, clean up subscriber on disconnect.
4. In the transfer service, after successful TigerBeetle write: re-fetch balances, render fragments via templ, publish `balances` + `history` events to the hub.
5. Add `hx-ext="sse"` + `sse-connect` + `sse-swap` attributes to the player view.

**Done when:** two browser tabs on the same session see each other's transfers update within ~100ms with no manual refresh.

---

## Phase 8 — Action sheets + the full transfer UX (Evening 3, hour 1)

**Goal:** all four action buttons work end to end with the amount-entry sheet.

1. Build the amount entry sheet component (templUI's Sheet, custom number pad inside).
2. Wire **Collect $200** as a one-tap action (no sheet, posts directly).
3. Wire **Pay Bank** and **Collect from Bank**: open sheet with amount pad, post to `/api/sessions/{code}/transfers` with the right transfer code.
4. Wire **Pay Player**: sheet includes player picker (horizontal pill row) + amount pad.
5. Add quick-amount chips ($50, $100, $200, $500) above the pad.
6. Optimistic UI: on tap, immediately render the updated balance from the POST response.
7. Test on an actual phone over local WiFi (`http://<laptop-ip>:8080`).

**Done when:** you can run a 5-minute fake game between two phones using only the app.

---

## Phase 9 — Admin panel + history (Evening 3, hour 2)

**Goal:** session creator can undo and adjust; everyone can see history.

1. Add admin badge to the balance card (only visible if `playerID == session.adminID`).
2. Admin sheet: **Undo last transfer** + **Adjust balance** (with reason note) + **End game**.
3. Wire to `/api/sessions/{code}/admin/{undo,adjust}` endpoints.
4. Transaction feed page (`/sessions/{code}/history`): reverse-chronological list with category icons. Admin sees Undo button on most recent only.
5. SSE `history` event prepends new rows to the feed if it's open.

**Done when:** you can undo a wrong transfer and see the reversal reflected everywhere within ~100ms.

---

## Phase 10 — Polish (Evening 3, hour 3)

**Goal:** the app feels finished.

1. Tap target audit: every interactive element ≥44px.
2. Contrast audit: text/background ratios ≥4.5:1 in dark mode.
3. Balance count-up animation on change (CSS transition or tiny JS via templUI).
4. Toast notification on incoming transfers ("Alice paid you $200") — SSE `toast` event.
5. Empty states: lobby with no players, history with no transfers.
6. Error states: invalid amount, insufficient funds (TigerBeetle returns specific error codes — surface them as toasts).
7. Background goroutine: sweep sessions where `last_active` is >8h old (delete row, archive ledger).

**Done when:** you'd show this to a friend without preemptive apologies.

---

## Phase 11 — Deploy (Evening 3, hour 4)

**Goal:** running at <https://passgo.quest>.

1. Spin up Hetzner CX11 (or smallest available), point `passgo.quest` A record at it.
2. Install Coolify via their one-liner script. Wait ~5 min.
3. In Coolify UI: New Resource → Docker Compose → point at GitHub repo + branch.
4. Add environment variables: `TB_ADDR`, `DB_PATH`, `PORT`, `COOKIE_SECRET` (generate a random 32-byte hex).
5. Configure persistent volumes for TigerBeetle data file and SQLite file.
6. Set domain to `passgo.quest`, enable Let's Encrypt.
7. First deploy. Watch logs in Coolify.
8. Test from a phone on cellular (real-world latency check).

**Done when:** typing `passgo.quest` on your phone over cellular loads the landing page, lets you create a game, and works smoothly across two devices.

---

## Phase 12 — README + ship (after Evening 3, optional)

1. Write the README: tagline (*"Pass Go. Written in Go. Skip the paper money."*), Hasbro disclaimer, screenshot from phone, 30s GIF of a live transfer across two devices.
2. Push to main, share the link.

---

## Stretch goals (post-v1)

- Linked transfers for atomic trades (Boardwalk + $200 for Park Place).
- Free Parking pot account (house rule).
- Spectator link.
- Property ownership tracking.
- Game replay from TigerBeetle's immutable transfer log.
- Multi-game suite under the `passgo.quest` domain (Catan, Risk, etc.).

---

## Risk register (revisit at each phase boundary)

| Risk | Mitigation | Phase to watch |
|---|---|---|
| TigerBeetle SDK friction | Pin version, isolate behind `tb.Client` interface | Phase 2 |
| u128 ergonomics | Helper module written once, used everywhere | Phase 2 |
| SSE buffered by Coolify proxy | Set `X-Accel-Buffering: no` header | Phase 7, 11 |
| templUI API churn | Pin commit/tag in go.mod | Phase 6 |
| Mobile Safari quirks on SSE | Test on real iOS device before Phase 11 | Phase 8 |
| Cookie HMAC secret leakage | Use `COOKIE_SECRET` env var, never commit | Phase 5 |

---

## Daily checkpoints

- **End of Evening 1:** Phases 0–3 complete. TigerBeetle + SQLite working through Go, no UI.
- **End of Evening 2:** Phases 4–7 complete. Two phones can play a fake game, balances update live.
- **End of Evening 3:** Phases 8–11 complete. Deployed to passgo.quest, ready for real game night.

If you fall behind, the cut order is: deployment (Phase 11), admin panel polish (Phase 9), toast/animation polish (Phase 10). Keep the core transfer flow + SSE no matter what.
