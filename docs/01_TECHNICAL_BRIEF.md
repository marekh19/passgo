# Pass Go — Technical Brief

> Product name: **Pass Go** · technical slug: `passgo` · domain: passgo.quest

## Stack

- **Language:** Go 1.23+
- **Database:** TigerBeetle (single-node, Docker for dev) for money; SQLite for session metadata
- **Templating:** Templ (type-safe Go components)
- **Component library:** [templUI](https://templui.io/) — pre-built templ components styled with Tailwind
- **Styling:** Tailwind CSS (standalone CLI, no Node required)
- **Interactivity:** HTMX + HTMX SSE extension
- **SQLite driver:** `modernc.org/sqlite` (pure Go, no CGO)
- **Query layer:** sqlc (generates type-safe Go from SQL)
- **Task runner:** [Taskfile](https://taskfile.dev/) (`Taskfile.yml`, not Make)
- **Containerization:** Docker (multi-stage build)
- **Deployment:** VPS + [Coolify](https://coolify.io/) (self-hosted PaaS, Git-based deploys)

No JavaScript framework. No Node at runtime. Single Go binary containerized alongside TigerBeetle.

## Why this stack

- **Go:** stdlib HTTP server is enough, great concurrency for SSE fan-out, TigerBeetle's official Go client is the most mature.
- **Templ + templUI:** compile-time-checked templates plus a battery-included component set (buttons, sheets, inputs, toasts) so we don't reinvent UI primitives. templUI components are just templ files — drop them in, customize freely.
- **Tailwind standalone:** one binary watches and rebuilds CSS, no `npm install`. templUI is designed for it.
- **HTMX + SSE:** server pushes balance updates, clients POST actions. Sub-100ms perceived latency, no WebSocket complexity.
- **SQLite for session state:** survives restarts, debuggable with any SQLite viewer, zero ops.
- **Taskfile:** YAML, cross-platform, better dependency handling than Make. Built-in `--watch` mode.
- **Coolify:** push to Git, it builds and deploys. No fly.io lock-in, runs on a $5 Hetzner box, gives you logs/metrics/TLS for free.

## Project layout (vertical slices)

Each feature lives in its own folder under `internal/`. The folder's entrypoint file shares the folder's name and defines the public interface for that slice. This makes dependency injection trivial (constructors take interfaces from other slices) and keeps tests isolated.

```
passgo/
├── cmd/passgo/main.go                  # entry point, wiring
├── internal/
│   ├── tb/                             # TigerBeetle slice
│   │   ├── tb.go                       # interface + constructor
│   │   ├── transfers.go
│   │   ├── accounts.go
│   │   └── tb_test.go
│   ├── store/                          # SQLite slice
│   │   ├── store.go                    # interface + constructor
│   │   ├── schema.sql
│   │   ├── queries.sql                 # sqlc input
│   │   ├── generated/                  # sqlc output (not edited by hand)
│   │   └── store_test.go
│   ├── session/                        # session lifecycle slice
│   │   ├── session.go                  # interface + constructor
│   │   ├── manager.go                  # create/join/start/end
│   │   ├── codes.go                    # short code generation
│   │   └── session_test.go
│   ├── transfer/                       # transfer execution slice
│   │   ├── transfer.go                 # interface + constructor
│   │   ├── rules.go                    # transfer codes, validation
│   │   └── transfer_test.go
│   ├── broadcast/                      # SSE fan-out slice
│   │   ├── broadcast.go                # interface + constructor
│   │   ├── subscribers.go
│   │   └── broadcast_test.go
│   ├── admin/                          # admin actions slice
│   │   ├── admin.go                    # interface + constructor
│   │   ├── undo.go
│   │   └── admin_test.go
│   ├── auth/                           # cookie-based identity slice
│   │   ├── auth.go                     # interface + constructor
│   │   └── auth_test.go
│   ├── http/                           # HTTP layer (routes + handlers)
│   │   ├── http.go                     # router setup, mounts routes
│   │   ├── ui.go                       # UI handlers (HTML responses)
│   │   ├── api.go                      # API handlers (HTMX form POSTs)
│   │   ├── sse.go                      # SSE handler
│   │   ├── middleware.go
│   │   └── http_test.go
│   └── views/                          # templ components (.templ files)
│       ├── layout.templ
│       ├── pages/
│       │   ├── landing.templ
│       │   ├── lobby.templ
│       │   └── player.templ
│       └── components/                 # app-specific; templUI imported as needed
│           ├── balance.templ
│           ├── players_list.templ
│           ├── action_sheet.templ
│           └── history.templ
├── static/
│   ├── css/output.css                  # Tailwind output
│   └── htmx.min.js                     # vendored, no CDN
├── tailwind.config.js
├── input.css                           # Tailwind directives + templUI imports
├── sqlc.yaml
├── Taskfile.yml
├── Dockerfile
├── compose.yml                         # for local dev (TigerBeetle + app)
├── go.mod
└── README.md
```

### The vertical-slice pattern

Every slice under `internal/` follows the same shape. Example for `tb`:

```go
// internal/tb/tb.go
package tb

import "context"

// Client is the public interface for the TigerBeetle slice.
// Other slices depend on this interface, not the concrete impl.
type Client interface {
    CreateAccount(ctx context.Context, ledger uint32, code uint16, flags AccountFlags) (Uint128, error)
    Transfer(ctx context.Context, t TransferReq) (Uint128, error)
    LinkedTransfer(ctx context.Context, ts []TransferReq) ([]Uint128, error)
    Balances(ctx context.Context, ids []Uint128) (map[Uint128]Balance, error)
    History(ctx context.Context, ledger uint32, limit int) ([]Transfer, error)
}

type client struct { /* ... */ }

func New(addr string) (Client, error) {
    // construct and return concrete impl satisfying Client
}
```

Then `internal/session/session.go` takes `tb.Client` and `store.Store` as constructor args:

```go
// internal/session/session.go
package session

type Manager interface {
    Create(ctx context.Context, adminName string) (*Session, error)
    Join(ctx context.Context, code, playerName string) (*Player, error)
    Start(ctx context.Context, code, adminID string) error
}

func New(tbClient tb.Client, db store.Store) Manager { /* ... */ }
```

Tests in each slice get fakes for the interfaces they depend on. No mocking framework needed — small hand-written fakes are fine for this size of project.

`cmd/passgo/main.go` is the only place that wires concrete implementations together.

## TigerBeetle data model

**Ledger per session.** Each game session gets a unique `ledger` ID (uint32). Accounts in different sessions can't transfer between each other.

**Account codes:**

- `1` = Player
- `2` = Bank
- `3` = Free Parking pot (v1.1)

**Account fields:**

- `id` = u128, generated from snowflake-ish scheme
- `ledger` = session's ledger ID
- `code` = account type
- `user_data_128` = player ID (foreign key to SQLite `players.id`)
- `user_data_64` = session code (sanity check)
- `flags`:
  - Players: `debits_must_not_exceed_credits = true`
  - Bank: no flags (unlimited credit)

**Initial state on game start:** Bank transfers $1,500 to each player.

**Transfer codes:**

- `10` = Player → Player (rent, gift)
- `20` = Bank → Player — GO salary
- `21` = Bank → Player — Mortgage payout
- `22` = Bank → Player — Chance/Community Chest payout
- `23` = Bank → Player — Generic collect from bank
- `30` = Player → Bank — Property purchase
- `31` = Player → Bank — Tax
- `32` = Player → Bank — Generic pay bank
- `99` = Admin adjustment

**Amounts:** u128 integers, whole dollars only. $200 = `200`.

**Undo:** post a reversing transfer with original ID in `user_data_128`, code `99`. UI hides reversed pairs from the feed.

## SQLite schema

```sql
CREATE TABLE sessions (
  code         TEXT PRIMARY KEY,
  ledger_id    INTEGER NOT NULL UNIQUE,
  bank_acct_id BLOB NOT NULL,
  admin_id     TEXT NOT NULL,
  started      INTEGER NOT NULL DEFAULT 0,
  created_at   INTEGER NOT NULL,
  last_active  INTEGER NOT NULL
);

CREATE TABLE players (
  id           TEXT PRIMARY KEY,
  session_code TEXT NOT NULL REFERENCES sessions(code) ON DELETE CASCADE,
  name         TEXT NOT NULL,
  acct_id      BLOB NOT NULL,
  joined_at    INTEGER NOT NULL,
  active       INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX idx_players_session ON players(session_code);
CREATE INDEX idx_sessions_last_active ON sessions(last_active);
```

WAL mode, single writer, multiple readers. Background goroutine sweeps sessions inactive >8h.

## In-memory runtime state

Only live SSE channels (cannot be persisted):

```go
// internal/broadcast/broadcast.go
type Hub interface {
    Subscribe(sessionCode, playerID string) (<-chan Event, func())
    Publish(sessionCode string, ev Event)
}
```

Lazily created per session on first subscriber after a restart.

## HTTP routes

UI routes are unprefixed (humans navigate here). API routes are under `/api` (HTMX form POSTs).

```
# UI — HTML responses, server-rendered templ pages
GET   /                                  landing page
GET   /sessions/{code}                   lobby or player view
GET   /sessions/{code}/history           transaction feed (HTMX fragment)
GET   /sessions/{code}/events            SSE stream

# API — HTMX-driven, return HTML fragments or 303 redirects
POST  /api/sessions                              create session
POST  /api/sessions/{code}/join                  join session
POST  /api/sessions/{code}/start                 admin: start game
POST  /api/sessions/{code}/transfers             execute transfer
POST  /api/sessions/{code}/admin/undo            admin: undo last transfer
POST  /api/sessions/{code}/admin/adjust          admin: manual balance adjustment

# Static
GET   /static/*                          CSS, htmx.min.js
```

Player identity via signed cookie (`playerID + sessionCode`, HMAC). No real auth.

## SSE design

`GET /sessions/{code}/events` keeps a connection open per player. On any transfer:

1. Persist to TigerBeetle.
2. Re-fetch all session balances (one TB call).
3. Render updated HTML fragments via templ.
4. Publish to broadcast hub → fan out to subscribers.

Events:

- `balances` — replaces balance card + players list
- `history` — prepends new row to transaction feed
- `toast` — ephemeral notification

Client-side:

```html
<div hx-ext="sse" sse-connect="/sessions/ABC123/events">
 <div sse-swap="balances" hx-swap="innerHTML">@balanceCard(...)</div>
 <div sse-swap="history" hx-swap="afterbegin"></div>
</div>
```

The acting player's POST response returns the updated fragment directly — they don't wait on SSE roundtrip.

## Taskfile

`Taskfile.yml` at repo root:

```yaml
version: '3'

vars:
  BIN: bin/passgo

tasks:
  dev:
    desc: Run everything in watch mode (templ, tailwind, app)
    deps: [tb:up]
    cmds:
      - task: templ:watch &
      - task: tailwind:watch &
      - task: app:run

  templ:gen:
    desc: Generate Go code from .templ files
    cmds:
      - templ generate

  templ:watch:
    cmds:
      - templ generate --watch

  tailwind:build:
    cmds:
      - tailwindcss -i input.css -o static/css/output.css --minify

  tailwind:watch:
    cmds:
      - tailwindcss -i input.css -o static/css/output.css --watch

  sqlc:
    desc: Regenerate SQLite query bindings
    cmds:
      - sqlc generate

  db:migrate:
    desc: Apply SQLite schema (idempotent)
    cmds:
      - sqlite3 passgo.db < internal/store/schema.sql

  tb:up:
    desc: Start TigerBeetle in Docker
    cmds:
      - docker compose up -d tigerbeetle

  tb:down:
    cmds:
      - docker compose down

  app:run:
    deps: [templ:gen, tailwind:build]
    cmds:
      - go run ./cmd/passgo

  build:
    deps: [templ:gen, tailwind:build]
    cmds:
      - go build -o {{.BIN}} ./cmd/passgo

  test:
    cmds:
      - go test ./...

  lint:
    cmds:
      - go vet ./...
      - gofmt -l -d .

  docker:build:
    cmds:
      - docker build -t passgo:latest .
```

## Docker & Coolify deployment

**Multi-stage Dockerfile:**

```dockerfile
# Build stage
FROM golang:1.23-alpine AS build
WORKDIR /src
RUN wget -O /usr/local/bin/tailwindcss https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 \
    && chmod +x /usr/local/bin/tailwindcss
RUN go install github.com/a-h/templ/cmd/templ@latest
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN templ generate
RUN tailwindcss -i input.css -o static/css/output.css --minify
RUN CGO_ENABLED=0 go build -o /out/passgo ./cmd/passgo

# Runtime stage
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/passgo /passgo
COPY --from=build /src/static /static
COPY --from=build /src/internal/store/schema.sql /schema.sql
EXPOSE 8080
ENTRYPOINT ["/passgo"]
```

**compose.yml** (local dev — TigerBeetle + app):

```yaml
services:
  tigerbeetle:
    image: ghcr.io/tigerbeetle/tigerbeetle:latest
    command: start --addresses=0.0.0.0:3000 /data/passgo.tigerbeetle
    volumes:
      - tb-data:/data
    ports:
      - "3000:3000"

  app:
    build: .
    environment:
      TB_ADDR: tigerbeetle:3000
      DB_PATH: /data/passgo.db
      PORT: 8080
    volumes:
      - app-data:/data
    ports:
      - "8080:8080"
    depends_on:
      - tigerbeetle

volumes:
  tb-data:
  app-data:
```

**Coolify deployment:**

1. Spin up a VPS (Hetzner CX11, ~$4/mo is plenty).
2. Install Coolify via their one-liner.
3. In Coolify: new resource → Docker Compose, point at the GitHub repo.
4. Coolify reads `compose.yml`, builds the image, runs both services, terminates TLS automatically via its built-in reverse proxy.
5. Point `passgo.quest` DNS at the VPS IP. Coolify provisions a Let's Encrypt cert.
6. Git push → auto-deploy on main.

Persistent volumes for `tb-data` (TigerBeetle data file) and `app-data` (SQLite file) survive redeploys. Back up by snapshotting the volume or `sqlite3 .backup` for the SQLite side.

**HTTP/2 note:** Coolify's default reverse proxy supports HTTP/2, which matters for SSE (HTTP/1.1 limits browsers to ~6 concurrent connections per origin).

## Concurrency model

- Broadcast hub: one `sync.RWMutex` per session for subscriber list.
- TigerBeetle handles transfer-level transactional consistency.
- SQLite in WAL mode: single writer, concurrent readers.
- SSE broadcast: non-blocking sends, buffered channels, drop oldest on slow client.

## Risks & mitigations

- **TigerBeetle Go client API churn.** Pin the version. The `tb.Client` interface in the slice means a future upgrade only touches `internal/tb`.
- **SSE through Coolify's proxy.** Should work; if buffering kicks in, set the response header `X-Accel-Buffering: no` and configure the proxy to honor it.
- **u128 ergonomics in Go.** Make a `tb.Uint128` helper for ID gen + stringification. Use it everywhere.
- **templUI version churn.** It's a young project; pin a specific commit/tag.

## Open questions to decide on Evening 1

- Spectator link? (v1: no.)
- Player drop-off mid-game? (v1: admin marks "out," remaining balance → Bank.)
- Cents? (No. Whole dollars.)
