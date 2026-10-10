# Project status

This is the maintained record of current behavior and work. For the intended
product and release boundary, read [product intent](PRODUCT.md). If the user has
not given a more specific task, work only on the single task under **Now**.
Read its task file for scope and completion checks. **Later** is an ordered
queue, not permission to build those items at the same time.

## Implemented

- Players can create a game, join by code, and start it. The creator is the host, and starting gives each player $1,500.
- Players can pay another player or the Bank, and collect from the Bank. TigerBeetle stores transfers; SQLite stores game and player details.
- Landing, join, lobby, and player pages exist. Started games show balances. Each transfer redirects to an updated page.
- Lobby and game pages open an authenticated in-process SSE stream and refresh server-rendered roster/balance fragments after successful joins, starts, and transfers.
- The core flow and live-update client behavior have automated tests; HTTP flow tests use real handlers and SQLite with a fake ledger. The TigerBeetle round-trip test runs when a local TigerBeetle server is available.

## Now

- [02 Design system](tasks/02-design-system.md)

## Completed

- [01 Live game updates](tasks/01-live-updates.md)
  - 2026-10-05: `task test` passed.
  - 2026-10-05: `task build` passed.
  - 2026-10-05: Manual two-device check with TigerBeetle/app running passed.

## Later

- [03 Screen redesign](tasks/03-screen-redesign.md)
- [04 Transaction history](tasks/04-transaction-history.md)
- [05 Host controls](tasks/05-host-controls.md)
- [06 Game integrity](tasks/06-game-integrity.md)
- [07 Release and deployment](tasks/07-release.md)

## Completing a task

When every **Done when** item passes, update **Implemented**, move its link from
**Now** to a **Completed** list, and promote the first **Later** link to **Now**.
Record any verification that could not run; leave the task active if a completion
check remains unmet. Update a future task file only when its scope or acceptance
checks actually change. The project is ready to call finished when task 07 is
complete and no tasks remain under **Now** or **Later**.
