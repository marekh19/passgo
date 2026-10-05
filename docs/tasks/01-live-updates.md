# 01 Live game updates

**Outcome:** People in the same game see joins, game start, and balance changes
without refreshing.

**Scope:** Add SSE to the current lobby and game pages. Keep the temporary visual
design and existing transfer forms. History, host controls, and redesign are
separate tasks.

**Starting points:** `internal/http/http.go` registers routes;
`internal/http/handlers.go` renders pages and handles actions;
`internal/views/pages/` contains templates;
`internal/http/flow_test.go` exercises the HTTP flow. Follow the session and
transfer services when deciding where to publish successful changes.

**Done when:**

- A player joining appears in other open lobby pages.
- Starting the game moves every connected player to the game page.
- A transfer updates balances in two open browser sessions.
- An unauthenticated visitor or a member of another game cannot subscribe to
  this game's events.
- A player who reconnects sees the current roster and balances without relying
  on missed events.
- Automated tests cover updates and access control. `task test` and `task build`
  pass, and the flow works in two browser sessions with TigerBeetle running.
