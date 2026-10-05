# 04 Transaction history

**Outcome:** Players can inspect the money movements in their game and understand
who paid whom, how much, and when.

**Scope:** Record and display successful transfers for one game, including the
opening credits and later corrections when those features exist. TigerBeetle
remains the authority for money; any additional metadata must stay consistent
with it. This task does not add host actions.

**Done when:**

- A member can view a reverse-chronological history for their game, with names,
  amount, direction, and action type. Empty and failed loads are understandable.
- A member cannot read another game's history; unauthenticated visitors cannot
  read it either.
- Reloading or restarting the app does not lose recorded movements. Retried
  requests do not create duplicate visible movements.
- Tests cover persistence, ordering, access control, and the display.
  `task build` and `task test` pass.
