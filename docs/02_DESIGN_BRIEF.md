# Monopoly Cash Ledger — Mobile UI Design Brief

## Product

A mobile-first web app that replaces Monopoly's paper money. Players join a session via code, see live balances, and tap buttons to execute cash transfers. The physical board, dice, deeds, and cards stay on the table.

## Users & context

- 3–6 players, ages ~10 to adult, sitting around a board game.
- Each player on their own phone, holding it in one hand while the other reaches for dice/deeds.
- Glanceable use: 90% of interactions are <5 seconds (tap a button, confirm, back to the game).
- Lighting varies (kitchen table, dim living room). Phones may be passed around.
- One player is the session admin (creator), with extra powers.

## Design principles

1. **Tap-count is the metric.** Common actions in ≤2 taps. "Collect $200" should be one tap from the main screen.
2. **Glanceable, not pretty.** Big numbers, big buttons, high contrast. Looks like a calculator, not a fintech app.
3. **Thumb-zone first.** Primary actions in the bottom half of the screen. Top is for read-only info.
4. **No modals that hide the balance.** The player's own balance is always visible.
5. **Optimistic UI.** Tap registers instantly; SSE confirms. Never show a spinner on a transfer.
6. **Trust the players.** No confirmation dialogs for routine transfers under, say, $500. Big transfers get a single "Confirm $X" tap.

## Roles

- **Player:** every participant, including the session creator. Can initiate any standard transfer involving themselves or the Bank (collect $200, pay tax, pay another player, etc.). The Bank is not a "role" anyone plays as — it's just an account that transfers route through.
- **Admin (session creator):** also a regular player, plus a hidden admin panel with referee powers (undo last transfer, adjust balances, end game). Accessed via a small badge on their balance card, not a separate identity.

## Screens

### 1. Landing

- App name + tagline.
- Two buttons: **Create Game** and **Join Game**.
- Join: enter 4–6 character code + your name.
- Create: enter your name, get a code to share, become admin.

### 2. Lobby (pre-game)

- Big shareable code at top (tap to copy).
- List of joined players as they arrive (live via SSE).
- Admin sees a **Start Game** button; others see "Waiting for host…".
- Starting the game credits everyone $1,500 from the Bank.

### 3. Main player view (the screen they live on)

**Top third — Your status:**

- Your name + big balance ($1,500). Balance is the largest text on the screen.
- Subtle "Game ABC123 · 5 players" line.

**Middle third — Other players:**

- Compact list: name + balance, one row each.
- Tapping a row pre-fills "Pay [name]" on the action sheet.

**Bottom third — Action buttons** (the thumb zone):

- **Collect $200** (GO) — one tap, done.
- **Pay Player** — opens amount + player picker.
- **Pay Bank** — opens amount entry.
- **Collect from Bank** — opens amount entry (for mortgages, Chance, etc.).

Floating bottom-right: small **history icon** → opens transaction feed.

### 4. Amount entry sheet

- Slides up from bottom. Doesn't cover the balance.
- Big number pad (calculator-style, custom — not the OS keyboard).
- Quick-amount chips above pad: $50, $100, $200, $500.
- For "Pay Player": player picker as a horizontal scrollable row of name pills above the pad.
- Single primary button: **Pay $X** or **Collect $X**.
- Cancel = swipe down or tap outside.

### 5. Transaction feed

- Reverse chronological list of every transfer in the session.
- Each row: "Alice → Bob · $200 · 2 min ago" with a subtle category icon (player/bank/GO).
- Admin sees an **Undo** button on the most recent transaction only.

### 6. Admin panel (session creator only)

- Accessible via a small badge on their own balance card.
- **Undo last transfer**, **Adjust player balance** (with reason note), **End game**.
- Visually distinct (different accent color) so it's clear when you're in admin mode.

## Visual style

- **Type:** system font stack. One bold display weight for balances, regular for everything else. Balance ~48px, buttons ~18px, body ~16px.
- **Color:** dark mode default (game nights happen in dim rooms, and OLED phones look great). Monopoly-ish accent — a muted green (~#1F7A4D) for "collect," a warm red (~#C0392B) for "pay." Bank actions in neutral gray. Restrained homage only, no full Monopoly board kitsch.
- **Layout:** single column, max-width 480px, centered on larger screens. No horizontal scroll anywhere except the player picker.
- **Feedback:** balance number animates briefly (count-up/down) when it changes. Subtle haptic on tap if available. No celebration animations — those get old by turn 3.

## Out of scope (v1)

- Property ownership, deeds, rent lookup
- Auctions, trades with linked transfers (stretch goal)
- Dice rolling, board state
- Auth, accounts, persistence beyond the session
- Spectator mode, replay, stats

## Accessibility

- All buttons ≥44px tap target.
- Contrast ratio ≥4.5:1 on all text.
- Numbers read correctly to screen readers ("one thousand five hundred dollars," not "1500").
- Don't rely on color alone to distinguish pay/collect — use icons or +/− signs too.

## Success criteria

- A player who's never seen the app can collect their GO salary within 10 seconds of joining.
- Paying rent to another player takes ≤4 taps from anywhere in the app.
- During a real game, the app is invisible — people argue about hotel rent, not the UI.
