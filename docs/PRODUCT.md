# Product intent

Pass Go replaces paper money during an in-person Monopoly game. The board, dice,
deeds, and cards stay on the table. This document records product intent, not
implemented behavior or an approved screen design. See [project status](STATUS.md)
for what works now and what to build next.

## People and roles

- About 3–6 people play around one table, each using a phone for quick money actions.
- The person who creates the game is both a player and its host. The Bank is an
  account, not another person.
- A short game code lets people join. Players do not need persistent user accounts.

## Intended game flow

1. A host creates a game, shares its code, and starts it after others join.
2. Starting credits each player with $1,500 from the Bank.
3. Players see balances and move whole-dollar amounts between themselves and the
   Bank. Collecting $200 for passing GO should be a quick action.
4. Everyone's open game view stays current as other players act.
5. Players can review money movements. The host can correct mistakes and end the
   game. These are release goals, not claims about current implementation.

Money movements must be accurate. A player cannot pay more than their balance;
the Bank has no spending limit. Failed actions need a clear explanation.

## Experience goals

- Keep frequent actions short and usable with one hand on a phone. Collecting
  $200 should be a single action.
- Make balances and action results easy to read. Use accessible target sizes and
  contrast; do not rely on color alone to explain money movement.
- The current UI is temporary. [The visual mockup](../pass-go-design-reference.png)
  is the intended direction for the redesign. It shows the lobby, game screen,
  and pay-player flow; adapt its details for real content and accessibility.

## Outside the first release

Pass Go does not track the board, dice, property ownership, deeds, or cards. It
does not automate game rules beyond money movement. Spectator mode, replay, and
persistent user accounts are outside the first release.
