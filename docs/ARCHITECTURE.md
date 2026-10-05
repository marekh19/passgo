# Architecture

Pass Go is a server-rendered Go app with small feature slices under `internal/`. `cmd/passgo/main.go` composes concrete implementations; callers depend on slice interfaces where a dependency needs substituting, without adding a layer for every read.

## Ownership

- `internal/http` owns routes, request parsing, membership checks, HTTP responses, and view-model assembly. Mutations go through the owning slice; display reads may use storage and balance interfaces directly.
- `internal/session` owns game lifecycle and coordinates account creation/opening credits with session metadata. `internal/transfer` owns money-movement rules, account resolution, and posting. See its `rules.go` for supported player-initiated transfers.
- `internal/store` owns SQLite metadata and hides sqlc bindings; `internal/tb` owns TigerBeetle operations and hides the SDK. SQLite holds sessions and players, not balances or transfers. A game has one TigerBeetle ledger; account IDs cross the two systems via `[16]byte` and `tb.Uint128` helpers.
- `internal/auth` owns signed session cookies; HTTP verifies membership as well as the cookie. `internal/views` owns templ markup and display formatting, with HTTP combining metadata and balances for display.

## Cross-cutting choices

TigerBeetle enforces spending limits on player accounts; the Bank is unconstrained. Amounts are whole dollars. Operations touching SQLite and TigerBeetle have no shared transaction: consider partial failure and retries when changing lifecycle or money flows. Surface meaningful ledger failures to the user.

When adding live updates, use notifications rather than another data store: clients fetch authenticated server-rendered views after successful changes. Full-page reads remain usable without a stream.

When adding behavior, place rules with the slice that owns them, keep transport/presentation in HTTP and views, and test at the relevant seam with small fakes or real SQLite/HTTP flows. Follow `AGENTS.md` for project workflow and tooling.
