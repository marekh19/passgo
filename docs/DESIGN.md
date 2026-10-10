# Pass Go design system

The opt-in design system follows the playful print mockup. Production pages
still use the legacy layout. Run `task design:preview` and open
`http://<server-lan-ip>:7119` from a phone on the same network to inspect the
real shared components. Local access at <http://127.0.0.1:7119> also works.
The preview listens on all IPv4 interfaces and has no authentication; use it
only on trusted networks, do not forward its port, and stop it after review.
The preview requires no database or ledger and cannot move money. Task 03 adopts the system after
preview approval; it also adds real forms and payment failure handling.

## Tokens and typography

Tokens live in `input.css`, within the existing Tailwind v4 stylesheet. New
components use the `pg-` patterns or semantic utilities, never legacy `pay` and
`collect` utilities. Legacy colors remain unchanged until task 03 removes them.

| Semantic token | Value | Meaning |
| --- | --- | --- |
| paper | `#FFF9ED` | Background and surfaces |
| ink | `#171714` | Text, outlines, hard shadows |
| muted | `#5F5B52` | Secondary text on paper only |
| action-pay | `#FF4938` | Outgoing payments and primary start action |
| action-collect | `#FFE33D` | Collections and Host badge |
| ticket | `#A8DFFF` | Game code ticket |
| subdued | `#E9E3D7` | Disabled surfaces and neutral badges |

Self-hosted Anton 400 serves the brand, headings, large balances, codes, and
main action labels. Never synthesize bold or italic Anton. Barlow 400/600/700
serves body text, inputs, names, badges, and supporting labels. Fonts use swap;
source commit, download URLs, and licenses are in `static/fonts/`.

The type scale in rem is .875 small, 1 body, 1.125 control, 1.25 section,
1.5 action, 2.5 title, 3.5 code/amount, and 4.5 balance. Body line height is 1.5,
display 1.05, control 1.2. Use sentence case except actual game codes.
Money uses `$` and comma separators, with whole dollars only. Never round or
truncate a balance. Use `.pg-money-long` for long amounts so all digits wrap
in readable body type. The enhanced sheet switches automatically above
14 formatted characters.

## Space, surfaces, and controls

Use the 4/8/12/16/24/32/48px spacing scale, expressed in rem. Phone side padding
is 1rem, section gap 1.5rem, and control gap .5rem. Safe-area insets can increase
padding. The phone shell is at most 30rem; there is no desktop breakpoint.

Controls, cards, and containers have 2px ink outlines; action tiles use 3px.
Dividers use 1px. Radius is .5rem for fields/presets, .75rem for cards/buttons,
and 1.5rem for sheet top corners. Badges are pills. Primary buttons and tiles
use a 4px hard ink shadow; pressed state moves 2px and reduces the shadow.
Reduced motion removes the transform. Focus uses a 3px ink outline with 3px
offset. Keep surrounding space for that outline.

Use at least 48x48px interactive targets. Disabled controls have native disabled
semantics and subdued fill, not opacity. Add a reason when not obvious. All
bright fills use ink text. The red brand text is large display text only; red
on paper is not suitable for ordinary small text.

## Shared markup

`internal/views/design.templ` owns `AppLayout`, `Brand`, `Button`, `ActionTile`,
`Panel`, `Badge`, `Notice`, `PaymentSheet`, and `AmountEntry`. These do not contain
ledger rules. Native attributes are forwarded; always specify button `type`.
Do not override the primitive's class, sheet ID, or accessible heading reference.
Use native form attributes to keep server submission possible when integrating.

```templ
@Button(Pay, templ.Attributes{"type": "submit", "name": "intent", "value": "pay"}) {
    Pay $200
}
@Badge(Host) {
    Host
}
@Notice(Error, templ.Attributes{"role": "alert"}) {
    Payment failed. Your amount is saved. Try again.
}
```

Notice kinds are Info, Success, Warning, Error. Each has a visible textual
lead-in. Callers choose live-region behavior; use a polite status for dynamic
success and persistent errors. Do not announce balance or formatted mirrors
on each update. Names and You/Host text badges replace decorative identities.

Keep field, player row, and ticket patterns as explicit HTML:

```html
<label for="player-name">Your name</label>
<input id="player-name" name="name" class="pg-input"
       aria-describedby="name-error" aria-invalid="true">
<p id="name-error" class="pg-field-error">Enter your name.</p>

<div class="pg-player-row">
  <div>Christopher Williams <span class="pg-badge pg-badge-neutral">You</span></div>
  <span class="pg-money">$1,500</span>
</div>

<section class="pg-ticket" aria-label="Game code">
  <div><p>Share this code to join</p><p class="pg-code">ABCD</p></div>
  <button type="button" class="pg-button pg-neutral">Copy</button>
</section>
```

Code remains selectable text. Ticket notches are decorative CSS; the Copy
control has dashed separation. Task 03 supplies copying. Decorative SVGs are
hidden from assistive technology; icon-only controls have accessible names.

## Payment interaction

`PaymentSheet` is a native modal dialog, full-width at the viewport bottom,
with a 90dvh maximum height and scrollable interior. Opening locks the background
page at its current position; Close and Escape unlock it and restore that position.
The sheet remains scrollable, with scroll chaining contained at its edges.
The header can wrap at large text sizes; presets can reflow. No drag handle or
backdrop dismissal.
Native Escape closes; closing restores focus to the recorded trigger. Initial
focus goes to the labeled amount field. Native dialog provides containment.

`AmountEntry` keeps an editable text field with `inputmode="none"`. Typing,
selection, paste, presets, and keypad share validation. Digit buttons insert at
the saved selection; Backspace deletes the selection or preceding character.
Pointer keypad activation preserves the field selection and focus. Keyboard
users can focus each button. Presets replace the value and expose aria-pressed.

Accept ASCII digits only. Preserve invalid input with an associated inline
explanation. Empty and zero cannot submit. Leading zeroes normalize on blur or
submission. Validate exactly 1 through 18446744073709551615 using strings/BigInt,
not Number. The formatted display is hidden from assistive technology to avoid
announcing the amount twice.

The preview's `data-payment-demo` hook initializes `payment.js`; it intercepts
submission locally and never fetches or posts. Pending state disables repeat
submission, marks aria-busy, preserves amount context, and temporarily locks
editing. Close and reopen clears demo state. Task 03 must add normal server
forms as the no-JS fallback and wire recoverable server failures while retaining
the entered amount. The demo initializer is deliberately not real payment wiring.

## Contrast and evidence

WCAG relative-luminance contrast, computed from the final opaque sRGB tokens:

| Pair | Ratio |
| --- | --- |
| Ink / paper | 17.13:1 |
| Ink / pay | 5.36:1 |
| Ink / collect | 13.95:1 |
| Ink / ticket | 12.54:1 |
| Ink / subdued | 14.06:1 |
| Muted / paper | 6.45:1 |
| Red brand / paper | 3.20:1 (large text only) |

Ink outlines and focus rings meet 3:1 against every component surface. Normal
text meets 4.5:1; large brand text meets 3:1. Do not add other pairings without
checking their contrast.

[Preview evidence](design-preview/README.md) records Chromium checks and phone
screenshots. The user approved the rendered direction and confirmed the phone
keyboard, larger-text/focus, and scroll-lock checks. Manual screen-reader
verification was removed at the user's request and was not performed. Accessible
markup and keyboard tests remain. Task 02 is complete.

## Intentional mockup departures

Four-character real codes; no fake phone bezel, textures, decorative motion,
player shapes, invented menu/history controls, or drag affordance. Use names and
role badges, an editable field alongside the keypad, flat fills, measured
contrast, and wrapping/smaller type for long content. Disabled preview actions
use subdued fill and explain why they are disabled. No desktop adaptation.
