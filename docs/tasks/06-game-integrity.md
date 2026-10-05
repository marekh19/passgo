# 06 Game integrity

**Outcome:** Failed or repeated requests cannot corrupt a game or let one player
spend another player's money.

**Scope:** Harden the existing create, join, start, and transfer paths before
release. Fix the underlying state transitions and trust boundaries. Keep the
single-instance deployment model unless a real requirement changes it.

**Done when:**

- Ordinary transfers can debit only the signed-in player's account and can
  credit only that player, another member, or the Bank as the action requires.
  A member cannot act in another game by sending a forged form or URL.
- Repeated or concurrent start requests deal each player's $1,500 exactly once.
  A failure partway through start leaves a recoverable state rather than an
  apparently started game with partial money.
- Create, join, and transfer failures do not leave misleading session/player
  records or duplicate money movements; retries have defined behavior.
- Tests exercise these failures and authorization boundaries with realistic
  handlers and fakes. `task build` and `task test` pass.
