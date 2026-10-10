# Preview verification

Task 02 implementation evidence. These are development-preview screenshots,
not migrated production screens. The user approved the visual direction and
confirmed phone scroll-lock, larger-text/focus, and keyboard checks. Task 02 is
complete under the revised testing scope.

Run `task design:preview` on the headless server, then open
`http://<server-lan-ip>:7119` on a phone on the same network. The preview now
binds to all IPv4 interfaces. It has no authentication; use a trusted network,
do not forward port 7119, and stop it after review. The browser script below
still uses loopback for its automated checks. After the LAN binding change,
`task build`, `task test`, and `git diff --check` passed. HTTP access via the
server's LAN address returned 200 with the preview page. Access from a separate
phone still needs checking; server-local access does not prove firewall reachability.

## Automated checks

- `task build`: passed.
- `task test`: passed, including Go preview route/markup tests and eight Node tests
  covering live updates, amount interactions, scroll locking/restoration, and
  cleanup after failed dialog opening.
  The TigerBeetle package test
  passed with the available local server.
- `go mod tidy`: completed without dependency changes.
- `git diff --check`: passed.
- `task design:preview`: generated assets and served successfully on loopback.
  The browser script stops it after checking; Task's exit 143 is intentional
  cleanup, not a failed render.
- Chromium 153.0.8010.12, Playwright 1.62.1: 84 assertions passed. See
  `browser-checks.txt` and the repeatable `check.cjs` script.

The script uses an existing Playwright installation and browser, not a new
project dependency. Run from the repository root with port 7119 free:

```sh
PLAYWRIGHT_MODULE=/absolute/path/to/playwright \
CHROMIUM_PATH=/absolute/path/to/chrome \
node docs/design-preview/check.cjs
```

To check an already running preview without starting or stopping its server,
set `PREVIEW_URL=http://127.0.0.1:7119` as well.

Checked both 320x568 and 390x844: no page overflow, 48px targets, initial field
focus, presets, pointer selection insertion, Backspace, physical keyboard input,
real clipboard paste, invalid-value retention, exact uint64 maximum, native modal
keyboard containment, Escape focus restoration, reopen/reset, and local demo
submission. Regression checks also prove the background stays fixed under
backdrop and sheet-boundary scrolling, closing restores its position, and the
sheet itself remains scrollable. Before the fix, backdrop scrolling moved the
background by 200px; after the fix these checks pass at both phone sizes.
At native modal focus wrap Chromium briefly focuses the document
body, but never a background control. No custom focus trap was added.

Also checked 200% root text size without horizontal overflow in the page or
sheet, submit scrolling at a reduced 350px viewport height, and reduced-motion
media activation. Screenshot inspection exposed enlarged sheet-header/preset/
keypad overflow; flexible wrapping and fixed minimum keypad targets resolved it.
The amount field scrolls its editable contents normally; the formatted mirror
wraps and retains every digit.

## Screenshots

| Viewport | Page | Sheet | Enlarged page | Enlarged sheet |
| --- | --- | --- | --- | --- |
| 320x568 | [Page](320-page.png) | [Sheet](320-sheet.png) | [200%](320-large-text.png) | [200% sheet](320-large-sheet.png) |
| 390x844 | [Page](390-page.png) | [Sheet](390-sheet.png) | [200%](390-large-text.png) | [200% sheet](390-large-sheet.png) |

Full-page screenshots include long names, six players, four-character code,
feedback examples, and the maximum amount. Sheet screenshots show the visible
portion of a scrollable dialog, not a requirement to fit every control at once.

## Manual verification and testing scope

- User confirmed phone keyboard use, larger-text/focus behavior, and the
  background-scroll fix passed. These confirmations supplement the automated
  viewport checks; no phone model/browser version was recorded.
- User explicitly removed manual screen-reader verification from testing.
  It was not performed. Accessible markup and automated keyboard tests remain.

Task 02 is complete. Task 03 applies the approved foundation to production screens.
