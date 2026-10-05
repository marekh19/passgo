# 03 Screen redesign

**Outcome:** The create, join, lobby, and game screens use the approved design
system and support the full existing game flow on a phone.

**Scope:** Redesign the working screens and their loading, empty, success, and
error states. Use [the visual mockup](../../pass-go-design-reference.png) for the
lobby, game screen, and pay-player flow; extend the same design language to
create, join, and other payment flows. Preserve the existing game behavior and
live updates. History and host-only controls remain separate tasks.

**Done when:**

- A player can create, join, start, pay another player, pay the Bank, and collect
  from the Bank through the redesigned screens.
- The lobby, game screen, and pay-player flow are recognizably aligned with the
  mockup, with any necessary accessibility or interaction changes documented in
  the design reference.
- Balances, participants, and action results remain clear while live updates
  arrive. Routine money actions do not depend on manual refresh.
- Forms show clear validation and failure messages, including insufficient funds.
- Keyboard navigation, visible focus, labels, contrast, and phone-size targets
  are checked. The flow is inspected on a narrow phone viewport and a desktop
  viewport.
- `task build` and `task test` pass; generated templ and CSS assets are current.
