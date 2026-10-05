# 05 Host controls

**Outcome:** The host can correct a money mistake and end a game without editing
data manually.

**Scope:** Let the host reverse the latest eligible ordinary transfer, adjust a
player's balance with a reason, and end a game. Show these actions in history.
The creator remains a normal player. No property or board administration.

**Done when:**

- Only the host can use host actions, including through direct HTTP requests.
- Undo reverses the latest eligible ordinary transfer once. Opening credits,
  adjustments, and previous reversals are not eligible; a retry cannot undo a
  different transfer.
- An adjustment requires a reason and records the amount, affected player, and
  host. Invalid amounts and insufficient funds fail clearly without changing
  balances.
- Ending a game prevents new joins, starts, and money movements while preserving
  its balances and history for members to read.
- Other open game views reflect each successful action. Tests cover permissions,
  retries, and failure paths; `task build` and `task test` pass.
