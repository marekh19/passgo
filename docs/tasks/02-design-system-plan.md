# Design-system implementation plan

This is the implementation contract for [task 02](02-design-system.md).
The user authorized implementation of this plan. The foundation and isolated
preview now exist; see [the design reference](../DESIGN.md) and
[verification evidence](../design-preview/README.md). The user approved the rendered visual direction and confirmed the phone checks,
including the background-scroll fix. Manual screen-reader verification was
removed at the user's request and was not performed. Task 02 is complete.

## Agreed boundaries

- Follow the [mockup](../../pass-go-design-reference.png)'s playful print direction.
- Mobile-only product: no desktop layout, desktop dialog variant, or desktop
  inspection requirement. Still accommodate different phone widths, text zoom,
  safe areas, and short viewports without clipping essential controls.
- Self-host Anton for display type and Barlow for readable text.
- Use cream, ink, red payments, yellow collections, and blue code tickets.
- Use names and textual You/Host badges; do not add player shapes or identities.
- Build a working reusable payment sheet and custom keypad in a development
  preview. Amount remains an editable field, not a keypad-only widget.
- Do not redesign production screens or connect the preview to real transfers.
  Task 03 integrates the approved foundation and payment interactions.
- User approval of the rendered preview is required before a full-screen rebuild.

## Deliverables and ownership

1. `docs/DESIGN.md`: short maintained visual/interaction reference with examples,
   final token table, usage rules, accessibility checks, and mockup departures.
2. `input.css`: tokens and reusable pattern classes in the existing Tailwind v4
   CSS-first setup. No second stylesheet, framework, or component dependency.
3. `static/fonts/`: font files plus licenses and a source/version manifest.
4. `internal/views/`: new opt-in `AppLayout` and shared templ primitives; retain
   the current `Layout` for legacy pages until task 03.
5. `static/js/payment.js` and `static/js/payment_test.mjs`: small progressive
   enhancement for the sheet and amount entry; no frontend framework.
6. `cmd/design-preview/`: development-only server rendering real shared templ
   components with fixed sample content. No production preview route.
7. `Taskfile.yaml`: `design:preview` command and payment-client tests included in
   `task test`. Commit generated templ and CSS alongside their source.

The preview binds to `0.0.0.0:7119` for phone review from the headless server,
serves its page and `/static/`, and
requires no database, authentication, TigerBeetle, or application API. Run it
through `task design:preview`, which depends on `generate`. It is not compiled
into `cmd/passgo`. Unsupported paths/methods return 404/405; no mutation routes.
It has no authentication. Use a trusted local network, do not forward port 7119,
and stop the preview after review.

## Visual contract

Use semantic tokens, not per-screen arbitrary colors. All text on the bright
fills uses ink. No gradients, textures, fake device bezel, or decorative motion.

| Token | Value | Use |
| --- | --- | --- |
| paper | `#FFF9ED` | App background and default surfaces |
| ink | `#171714` | Text, outlines, shadows |
| muted | `#5F5B52` | Secondary text on paper |
| pay | `#FF4938` | Outgoing money actions; primary start action |
| collect | `#FFE33D` | Incoming money actions and Host badges |
| ticket | `#A8DFFF` | Shared game code ticket |
| subdued | `#E9E3D7` | Disabled surface and neutral badges |

Expose the new semantic colors through `@theme` (e.g. `--color-action-pay`,
`--color-action-collect`). Preserve existing `--color-pay` / `--color-collect`
compatibility utilities unchanged until task 03 migrates the old pages; they
are legacy only, not another selectable theme. Task 03 removes those legacy
values and the old layout after all screens migrate. New components must not
use legacy tokens or hard-coded copies of token values.

- Font: Anton regular/400 for wordmark, page titles, balance, large code, and
  main action labels. Do not synthesize bold or italic Anton.
- Font: Barlow 400/600/700 for body, names, inputs, badges, field labels, and
  explanatory copy. Fallbacks: Anton -> Impact -> sans-serif; Barlow ->
  system-ui -> sans-serif. Use `font-display: swap`.
- Vendor upstream TTF files for those exact faces; do not add a font-conversion
  build dependency or fetch fonts at runtime. Include OFL files, the upstream
  commit and download URLs in the manifest. No external font/CDN requests.
  Sources: [Anton](https://github.com/google/fonts/tree/main/ofl/anton),
  [Barlow](https://github.com/google/fonts/tree/main/ofl/barlow), both SIL OFL.
- Type scale in rem: small `.875`, body `1`, control `1.125`, section `1.25`,
  action `1.5`, title `2.5`, code `3.5`, balance `4.5`. Body line height 1.5;
  display 1.05; control 1.2. Sentence case, except actual game codes.
- Money has separators and a `$` prefix; whole dollars only. Never round a
  balance to fit. Ordinary balances use display type; long values switch to
  body type and may wrap within their container without hiding digits.
- Spacing scale: 4, 8, 12, 16, 24, 32, 48px expressed in rem. Phone side padding
  16px; section gap 24px; control gap 8px. Respect safe-area insets.
- Outlines: 2px ink on controls, cards, and list containers; 3px on major action
  tiles. Dividers 1px ink. Radius: 8px inputs/presets, 12px cards/buttons,
  24px sheet top corners. Badges are pills.
- Hard shadow: 4px 4px 0 ink on primary buttons/action tiles. Hover is not
  required. Pressed state translates 2px on both axes and reduces shadow to
  2px; remove transition/transform motion under reduced-motion preference.
- Focus: 3px ink outline, 3px offset, with enough surrounding space not to clip
  it. Sheet backdrop is ink at 45%; no blur.
- The wordmark is text: “Pass” in ink and “Go” in red. Decorative icons are
  inline SVG, `aria-hidden`; icon-only controls require accessible names.
- Ticket treatment: blue outlined panel, large real four-character code,
  dashed separation before a labeled Copy control. Implement notches as purely
  decorative CSS; the code remains selectable text. Copy behavior is task 03.

## Bounded component inventory

Use templ components in `internal/views` and a small shared stylesheet vocabulary.
Do not build an extensible component framework. Components accept semantic
variants, children, and `templ.Attributes` where native attributes are needed;
forward ID, type, disabled, name, value, and ARIA/data attributes unchanged.
Native form submission must remain possible. Do not accept unrestricted color
or size props or embed game/business rules in shared components.

| Primitive/pattern | Contract / consumer |
| --- | --- |
| `AppLayout(title)` | Shared head/assets, paper/ink body, safe-area phone shell; children supply content. No desktop breakpoint or column expansion. |
| `Brand()` | Text wordmark; used in preview then screen headers. |
| `Button(variant, attrs)` | Native button; pay, collect, neutral variants; explicit button type; optional child icon/label. |
| `ActionTile(variant, attrs)` | Large native button, icon/title/supporting text; pay/collect only; 2-column phone action grid. |
| `Panel(attrs)` | Outlined surface with children; no extra interaction semantics. |
| `Badge(kind)` | Text children; host or neutral kind. No color-only roles. |
| `Notice(kind, attrs)` | Info, success, warning, error with visible textual lead-in; caller selects live-region behavior. |
| Field pattern | Native label/input/error markup and shared classes; do not wrap all input types in a generic form abstraction. |
| Player row pattern | Name, optional You/Host badges, optional right-aligned money; names wrap, money is not truncated. |
| Code ticket pattern | Four-character code and Copy button, as described above. |
| `PaymentSheet(id, title, attrs)` | Native dialog with header, named close button, form-content slot; always phone bottom sheet. |
| `AmountEntry(id, attrs)` | Labeled editable amount field, presets, digit buttons and Backspace; hidden enhancement hooks, not ledger logic. |

Shared row/field/ticket patterns may stay CSS plus explicit templ markup where
an extraction would only obscure native HTML. Document copyable examples in
`docs/DESIGN.md`. Do not add history tables, host tools, menu contents, selection
controls, or speculative variants just for future tasks. Their typography,
buttons, forms, notices, and surfaces already have suitable rules.

## Sheet and amount-entry contract

- Use native `<dialog>` and `showModal()`; no hand-written focus trap. Sheet
  sits at the viewport bottom, fills available width, and has a maximum height
  of 90dvh. Its interior scrolls; no drag-to-dismiss or decorative drag handle.
  Respect bottom safe area. The dialog does not change style on desktop.
- Trigger records its element and opens the dialog. Accessible name comes from
  the heading. Initial focus goes to the amount field. Close button and Escape
  close without submitting; restore focus to the trigger. Do not close on
  backdrop taps, to avoid losing an in-progress amount accidentally.
- Form uses a labeled text input, `inputmode="none"`, `autocomplete="off"`,
  `spellcheck="false"`, and decimal-digit guidance. This requests no phone
  keyboard where supported; never make the input readonly or block ordinary
  typing, selection, paste, or assistive editing. A browser showing its native
  keyboard is acceptable; the sheet must still scroll to its controls.
- Field contains raw dollar digits (no `$` or separators). A separate display
  shows the formatted amount; do not announce both the field and display on
  every digit. Initial field is empty, display is `$0`, submit is disabled.
- Presets `$50`, `$100`, `$200`, `$500` replace the whole amount. These are
  suggestions, not mandatory amounts. A preset is visibly and programmatically
  selected (`aria-pressed`) only when its value equals a valid field value.
- Keypad layout: 1/2/3, 4/5/6, 7/8/9, blank/0/Backspace. Every button has
  `type="button"`. Empty cell is not focusable. Backspace has a visible icon
  and accessible name. Targets at least 48px on each axis; gap 8px.
- Digit buttons insert at the saved field selection; Backspace removes the
  selection or preceding digit. Preserve selection on pointer interaction,
  keep keyboard focus usable, and update state through one amount-entry path.
  Presets replace the value irrespective of selection. Do not add global
  keyboard listeners; native editing keys work when the field has focus.
- Accept ASCII digits only. Empty/zero disables submission. Leading zeroes are
  permitted and normalize on blur/submission. Invalid pasted/typed values
  remain visible with an inline explanation; do not silently strip, round,
  clamp, or convert fractional/negative input into a different amount.
- Validate the existing uint64 transport range, 1 through
  `18446744073709551615`, using exact strings/BigInt, never JS Number for money.
  Do not impose a new arbitrary payment limit. Server remains authoritative.
- Submit label includes action and valid formatted amount, e.g. `Pay $200`.
  Pending state is a caller-controlled state: prevent repeat activation,
  preserve label context, expose `aria-busy`, and show visible “Paying…” text.
  A server failure retains the entered amount and explains how to recover.
- Preview submit is intercepted locally and shows “Demo only — no money moved.”
  No fetch, real IDs, or POST to the application. Reopening clears demo state.
- Task 03 uses normal server forms as the no-JS fallback, enhanced into sheets;
  this task only proves the shared interaction. Do not replace existing page
  forms, SSE fragments, or HTTP error handling in task 02.

## Preview and accessibility proof

Render one representative phone composition with the shared brand, sample balance,
player rows, four action tiles, code ticket, badges, and notices. A Pay Sam tile
opens the functional payment demo. Include clearly labeled static examples for
empty, error, warning, success, pending, and disabled states. This is a bounded
proof of the actual consumers, not a browsable component catalog.

Check at 320x568 and 390x844 CSS pixels. No desktop-specific check is required.
Also check 200% text enlargement, short usable height with the native keyboard,
reduced motion and keyboard navigation. Manual screen-reader verification is
excluded at the user's request. No horizontal
page scrolling, clipped focus, hidden controls, or truncated money. Sample data
must include a four-character code, 20-character player names, six players,
`$1,500`, and an exact uint64-max amount. Names wrap; long amounts may wrap or
use the smaller body treatment; essential digits never disappear.

- All interactive targets at least 48x48px, including Close and Copy.
- Normal text contrast at least 4.5:1, large text at least 3:1, essential control
  boundaries/focus indicators at least 3:1. Measure actual final combinations
  and record the ratios, not merely an assertion that the palette is accessible.
- Label every field, associate errors with `aria-describedby` and `aria-invalid`,
  use text/icons alongside color, and retain native button/dialog semantics.
- Error feedback is persistent; success uses a polite status region when shown
  dynamically. Do not repeatedly announce balance or formatted amount mirrors.
- Disabled states use subdued fill plus native disabled semantics and explanatory
  text when the reason is not obvious. Do not achieve disabled styling with opacity.
- Verify sheet focus entry/containment/restoration, Escape, Backspace/selection,
  presets, physical typing/paste, and validity/submit transitions in a browser.

## Implementation sequence

1. Plan authorized by the user's request to implement task 02. Do not substitute
   fonts, change scope, or begin screen redesign without a new explicit decision.
2. Vendor fonts/licenses; add tokens/pattern CSS and opt-in `AppLayout`. Keep
   legacy screen output unchanged. Start `docs/DESIGN.md` from these exact rules.
3. Build the bounded primitives and preview server/Taskfile command. Render
   real shared components, not duplicate mock HTML.
4. Build sheet and keypad enhancement. Add Node tests for empty/zero, presets,
   normalization, insertion/selection/backspace, invalid text/paste, exact
   uint64 boundary/overflow, formatted labels, and no repeated pending submit.
   Test init/close/reopen behavior with small hand-written DOM fakes following
   the existing live-client test approach. Native dialog/focus behavior also
   requires browser checks; fake DOM tests cannot establish it.
5. Add Go render/preview tests covering accessible markup, preview-only routes,
   and absence of mutation endpoints. Run the existing flow/live tests unchanged.
6. Run `task build` and `task test` (including new client tests). Run
   `task design:preview`; inspect the phone cases above and capture preview
   screenshots for user review. Record commands/results and manual checks.
7. Finalize `docs/DESIGN.md`, explicitly list mockup departures, and ask the user
   to approve the rendered direction. Approval of this written plan alone is
   not approval of the resulting preview.
8. Complete task 02 only after every task-level check passes and user approval
   is recorded. Follow `docs/STATUS.md` completion steps; only then begin task 03.

## Expected mockup departures

Four-character real codes; no fake phone border; flat rather than textured fills;
no invented menu/history functionality; no colored player tokens; accessible
text field alongside the custom keypad; no drag affordance; mobile-only layout;
wrapping/smaller type for real long content; measured accessible contrast/focus.

## Completion evidence

Record in task 02: implementation commit or file references, preview command,
phone screenshots, contrast ratios, keyboard/text-zoom results,
`task build`/`task test` outcomes, and explicit user preview approval. Any check
that cannot run remains unchecked and keeps task 02 active. Do not claim desktop
support, backend error integration, or completed screen redesign from this task.
