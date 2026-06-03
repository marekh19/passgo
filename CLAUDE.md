# Pass Go — project guide

Mobile-first web app that replaces Monopoly's paper money. Players join a session
by code, see live balances, and tap buttons to move cash between each other and
the Bank. Real-money correctness via TigerBeetle; everything else is plumbing.

Full specs live in `docs/`:
- `docs/01_TECHNICAL_BRIEF.md` — stack, layout, data model, routes, deploy
- `docs/02_DESIGN_BRIEF.md` — mobile UI, screens, visual style
- `docs/03_IMPLEMENTATION_PLAN.md` — phase-by-phase build order

When the work and these docs disagree, the **docs describe intent**, this file
records **decisions we've actually made**. This file wins on tooling/versions.

## Stack

- **Go 1.26** (docs say 1.23+; we run 1.26.3)
- **TigerBeetle** — the money ledger (Docker for dev). One ledger per session.
- **SQLite** via `modernc.org/sqlite` (pure Go, no CGO) — session + player metadata
- **sqlc** — type-safe Go from SQL
- **templ** — type-safe Go HTML components
- **Tailwind** standalone CLI (no Node) + **templUI** components
- **HTMX + SSE** — server pushes balance/history updates; clients POST actions
- **Taskfile** (`Taskfile.yaml`) — task runner, not Make
- **Docker** multi-stage + **Coolify** on a VPS for deploy

Module path: `github.com/marekh19/passgo`. No JS framework, no Node at runtime.

## Architecture — vertical slices

Each feature is a folder under `internal/`. The entrypoint file shares the folder
name and defines a **public interface + `New(...)` constructor**. Other slices
depend on the *interface*, never the concrete type.

- `tb` (TigerBeetle), `store` (SQLite), `session`, `transfer`, `admin`,
  `broadcast` (SSE hub), `auth` (signed cookie), `http` (routes/handlers),
  `views` (`.templ` files).
- `cmd/passgo/main.go` is the **only** place concrete impls get wired together.
  Construction order: `tb → store → auth → session → transfer → admin → broadcast → http`.
- **Tests use small hand-written fakes** for the interfaces a slice depends on.
  No mocking framework.

## Data model essentials

- **Whole dollars only.** No cents. `$200` = the integer `200`.
- TigerBeetle account codes: `1`=Player, `2`=Bank, `3`=Free Parking (v1.1).
  Players have `debits_must_not_exceed_credits`; Bank has no flags.
- Transfer codes: `10` P→P, `20-23` Bank→Player, `30-32` Player→Bank, `99` admin.
- **Undo** = post a reversing transfer, code `99`, original ID in `user_data_128`.
- Game start: Bank transfers `$1500` to each player.
- u128 IDs are awkward in Go — there's a `tb.Uint128` helper; use it everywhere.

## Tooling decisions (already done)

Rule: pin a tool as a **`go tool`** only if it's light *and* its version must match
generated code. Heavy or non-Go tools stay **global** so they don't pollute the
app's module graph (Go 1.24+ `go tool` deps join MVS).

| Tool | How | Invoke as |
|---|---|---|
| **templ** | `go tool` (in `go.mod`) | `go tool templ ...` |
| **sqlc** | global (brew) | `sqlc` |
| **tailwind** | global (brew, **v4**) | `tailwindcss` |
| **air** | global (`~/go/bin`) | `air` |
| **task** | global (brew) | `task` |

- Generated `*_templ.go` files **are committed** (simpler Docker build).
- `internal/store/generated/` is sqlc output — **never hand-edit**.
- Run `go mod tidy` before committing after any import/tool change.

## Watch out for (docs are stale here)

- **Tailwind is v4, not v3.** Ignore the `tailwind.config.js` in the brief —
  v4 is CSS-first (`@import "tailwindcss"` + `@theme {}` in the CSS file).
- **templ is a `go tool`, not on PATH.** Taskfile/Dockerfile must call
  `go tool templ generate`, not bare `templ generate`. (Brief's Taskfile uses
  the bare form — update it when writing the real Taskfile.)
- Dockerfile in the brief pins `golang:1.23` and `go install templ@latest`;
  bump to 1.26 and prefer `go tool templ` for version match.
- Shell is **fish**; `~/go/bin` added via `fish_add_path`.

## Conventions

- **Caveman writing** for PR/commit bodies and summaries: terse, noun-verb,
  bullets over prose, bold the key term. Normal clarity in code comments.
- Match surrounding code style. Small interfaces, constructor injection.
- TigerBeetle errors have specific codes (e.g. insufficient funds) — surface
  them as user-facing toasts, don't swallow.
