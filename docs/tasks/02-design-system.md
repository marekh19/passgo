# 02 Design system

**Outcome:** The redesign has an agreed visual and interaction foundation that
can be applied consistently across the app.

**Scope:** Use [the visual mockup](../../pass-go-design-reference.png) as the
primary visual direction. Translate its playful print feel, bold type, thick
outlines, light background, vivid action colors, and tactile controls into the
small set of tokens and reusable patterns needed by the existing screens and
planned history/host controls. It is a quick mockup, so adapt details that do
not work with real content or accessibility. Task 03 applies the system to the
screens.

**Implementation plan:** [Design-system plan](02-design-system-plan.md).
The user authorized implementation: mobile-only, an isolated development
preview, self-hosted Anton/Barlow, print-style colors, names and role badges
instead of player shapes, and a working payment sheet/custom keypad with an
editable amount field. These now exist. Production screen redesign and real
payment wiring stay in task 03. See [the maintained design reference](../DESIGN.md)
and [preview screenshots/checks](../design-preview/README.md).

**Done when:**

- A short maintained design reference states typography, color, spacing,
  controls, feedback, and phone-width/text-enlargement behavior with examples an
  implementer can follow. It names any intentional departures from the mockup; the user reviews
  the resulting direction before a full-screen rebuild.
- The shared layout and primitives needed by task 03 exist in the repo; they do
  not create a second styling system or unused component catalog.
- Contrast, focus states, tap targets, and meaning without color are covered.
- The development-only preview uses the actual shared primitives and a working
  payment-sheet/keypad demo without connecting to real money transfers. Existing
  production screens remain unchanged until task 03.
- The preview is inspected at 320x568 and 390x844 phone viewports, with larger
  text and long real-world content. No desktop layout or desktop verification is
  required. Sheet/keypad keyboard behavior is checked. Manual screen-reader
  verification is excluded at the user's request.
- `task build` and `task test` pass; generated templ and CSS assets are current.

## Planning / verification record

- User authorized the implementation plan. Added opt-in `AppLayout` and shared
  primitives in `internal/views/design.templ`, semantic CSS tokens/patterns,
  licensed self-hosted fonts, and the isolated `cmd/design-preview` server.
  Legacy layouts, production screens, SSE fragments, and payment handlers are
  unchanged. Generated templ/CSS assets are current and included in the change.
- `static/js/payment.js` implements the preview sheet/keypad with editable input,
  selection/paste handling, exact uint64 validation, and local-only submission.
  Go route/markup tests and Node behavior tests cover the preview boundary.
- `task build` and `task test` passed. A fresh `GOFLAGS='-count=1 -v' task test`
  also passed, including the real TigerBeetle round-trip without skips.
  `go mod tidy` and `git diff --check` passed.
- `task design:preview` initially served on loopback; at the user's request it
  now binds to all IPv4 interfaces on port 7119 for phone review from a headless
  server. No authentication: use trusted networks only and stop after review.
  Chromium 153 automated checks passed
  at 320x568 and 390x844, with 200% text, long content, native dialog navigation,
  focus restoration, physical typing/clipboard paste, presets, selection/
  Backspace, and amount/submit transitions. Short viewport scrolling passed.
  Inspected captured screenshots and fixed enlarged-text sheet overflow.
  Final measured contrast ratios and mockup departures are in `docs/DESIGN.md`.
- User approved the rendered visual direction: “It is beatiful, I love it”.
  Phone review then identified background scrolling while the sheet was open.
  Reproduced a 200px backdrop-scroll movement before fixing it. The sheet now
  fixes the background at its saved position, contains scroll chaining, and
  restores position on Close/Escape without blocking sheet scrolling. Node
  regression tests and 84 browser assertions passed; `task build`/`task test`
  passed. Screenshots/check results are refreshed.
- User confirmed phone checks passed for scroll lock, larger text/focus, and
  keyboard use. The user explicitly excluded screen-reader testing from this
  confirmation.
- User explicitly removed manual screen-reader verification from testing.
  It was not performed; accessible markup and automated keyboard tests remain.
  All revised completion checks are met. Task 02 is complete; task 03 is next.
