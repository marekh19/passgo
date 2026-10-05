# 07 Release and deployment

**Outcome:** Pass Go runs at a user-approved public location and supports a full
game on real phones without developer intervention.

**Scope:** Package and deploy the app and TigerBeetle with persistent data,
secrets, TLS, and a documented recovery path. Verify the product intent across
the finished screens. Choose the hosting target with the user before spending
money, buying a domain, or changing external services.

**Done when:**

- A fresh deployment can be configured from documented steps. Secrets are not
  committed; cookies are secure under HTTPS; app and ledger data survive restart.
  Production does not use the development-only unrestricted seccomp setting.
- A documented backup and restore test recovers both SQLite and TigerBeetle
  state together without changing balances or history.
- Two real phones complete a game: create, join, start, transfers, live updates,
  history, correction, and end. Check reconnection and failure messages.
- Accessibility and small-screen checks pass on the finished UI. `task build`
  and `task test` pass, including the TigerBeetle integration test.
- The README describes how to use and operate the released app. `docs/STATUS.md`
  reflects the verified release; no unfinished first-release task remains.
