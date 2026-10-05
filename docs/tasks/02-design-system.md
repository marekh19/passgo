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

**Done when:**

- A short maintained design reference states typography, color, spacing,
  controls, feedback, and responsive behavior with examples an implementer can
  follow. It names any intentional departures from the mockup; the user reviews
  the resulting direction before a full-screen rebuild.
- The shared layout and primitives needed by task 03 exist in the repo; they do
  not create a second styling system or unused component catalog.
- Contrast, focus states, tap targets, and meaning without color are covered.
- A representative screen or component can be inspected at phone and desktop
  widths. `task build` and `task test` pass.
