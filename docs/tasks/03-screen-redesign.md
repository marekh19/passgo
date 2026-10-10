# 03 Screen redesign

**Outcome:** The create, join, lobby, and game screens use the approved design
system and support the full existing game flow on a phone.

**Scope:** Redesign the working screens and their loading, empty, success, and
error states. Use [the visual mockup](../../pass-go-design-reference.png) for the
lobby, game screen, and pay-player flow; extend the same design language to
create, join, and other payment flows. Preserve the existing game behavior and
live updates. History and host-only controls remain separate tasks. The app is mobile-only;
do not add desktop-specific layouts or desktop dialog variants. Use the shared
payment-sheet/custom-keypad interaction delivered in task 02, with an editable
amount field and ordinary server-form fallback when JavaScript is unavailable.
The development preview is not a production flow: this task wires the primitives
to real actions and feedback. After all screens migrate, remove the legacy dark
layout and compatibility color tokens; retain one shared styling system.
Remove `cmd/design-preview/` and the `design:preview` task after the production
screens provide the real flows. Remove the preview-only composition, demo
submission hooks, and tests/scripts that depend on that server. Keep shared
primitives and payment regression coverage, adapting tests to real flows.
Update maintained docs so they no longer instruct users to run the removed
preview; retain screenshots and verification records as historical evidence.

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
  are checked. The flow is inspected at 320x568 and 390x844 phone viewports,
  including larger text, safe areas, and amount entry with the native keyboard
  when shown. No desktop-specific layout or inspection is required.
- `cmd/design-preview/`, the `design:preview` task, and their preview-only
  markup/demo hooks/tests/scripts are removed. Shared primitives and relevant
  payment regression tests remain. Maintained docs point to the production app
  for inspection, not the removed preview.
- `task build` and `task test` pass; generated templ and CSS assets are current.
