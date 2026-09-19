# Design system: Material Design 3

The site follows Material Design 3 (M3). Base UI gives us behavior and accessibility with no styles, so every visual decision comes from M3 tokens defined here. Token values below are taken from Google's official token sources (`material-components/material-web`, token set v0.192), which mirror <https://m3.material.io>.

**Rule:** components never use raw values. No hex colors, pixel radii, font sizes or durations in component CSS, only `var(--md-sys-*)`. If a value is missing, add a token here first.

**Scope:** we implement baseline M3. M3 Expressive (the 2025 update: spring motion, shape morphing, emphasized type) is optional polish for later; it has no official web implementation, and a calm, plain look suits a site whose job is to stay out of the way.

## How M3 shapes this product

The product's principle is that the site never labels or judges. The design has to keep that promise too:

- One neutral theme for everything. No color, badge, position or size difference between outlets.
- Every article card uses the same component, the same roles and the same type styles. Order is by time, never by outlet.
- The `error` color role is only for actual errors, never for content.
- Emphasis (primary color, elevation, larger type) goes to the site's own structure, such as the event title and timeline, not to any outlet's content.

## Tokens

M3 has three token tiers, and we keep all three as CSS custom properties:

| Tier | Example | Used by |
| --- | --- | --- |
| Reference (`--md-ref-*`) | `--md-ref-typeface-plain` | Only system tokens |
| System (`--md-sys-*`) | `--md-sys-color-primary` | Component CSS |
| Component (`--md-comp-*`) | `--md-comp-filled-button-container-height` | Added only when a component needs a named, reusable value |

## Color

### Scheme generation

All colors come from one seed color, run through `@material/material-color-utilities` by `web/scripts/generate-theme.ts`, which writes `web/app/theme/colors.css` (run `pnpm theme` after changing the seed). The script outputs:

- Light and dark schemes, selected by `prefers-color-scheme`, with a manual override stored in a cookie (a cookie, not `localStorage`, so the server renders the right theme and there is no flash).
- High-contrast versions of both, selected by `prefers-contrast: more`. M3's dynamic color defines three contrast levels (standard, medium, high); browsers expose only one "more contrast" signal, so we map it to high and skip medium.
- The script fails if any text role pair falls below 4.5:1 in any scheme.

### Choosing the seed

Constraint: the seed must not read as a Taiwanese party color. That rules out blue (KMT), green (DPP), turquoise (TPP), yellow (NPP), orange (PFP) and strong red.

Recommendation: a muted violet near the M3 baseline (`#6750A4`), with the **Neutral** or **Tonal Spot** scheme variant so surfaces stay close to gray and the hue appears only in small accents. Try two or three seeds in Material Theme Builder before committing. The seed is an open decision.

### Color roles

Component CSS uses roles, never palette tones. The full set:

| Group | Tokens |
| --- | --- |
| Primary | `--md-sys-color-primary`, `-on-primary`, `-primary-container`, `-on-primary-container` |
| Secondary | `--md-sys-color-secondary`, `-on-secondary`, `-secondary-container`, `-on-secondary-container` |
| Tertiary | `--md-sys-color-tertiary`, `-on-tertiary`, `-tertiary-container`, `-on-tertiary-container` |
| Error | `--md-sys-color-error`, `-on-error`, `-error-container`, `-on-error-container` |
| Surface | `--md-sys-color-surface`, `-surface-dim`, `-surface-bright`, `-surface-container-lowest`, `-surface-container-low`, `-surface-container`, `-surface-container-high`, `-surface-container-highest` |
| On surface | `--md-sys-color-on-surface`, `-on-surface-variant` |
| Outline | `--md-sys-color-outline`, `-outline-variant` |
| Inverse | `--md-sys-color-inverse-surface`, `-inverse-on-surface`, `-inverse-primary` |
| Other | `--md-sys-color-scrim`, `-shadow`, `-background`, `-on-background` |

Pairing rule: content on a role uses that role's `on-` color, and nothing else. Text on `primary-container` is `on-primary-container`; text on any surface is `on-surface` (high emphasis) or `on-surface-variant` (lower emphasis). Following the pairs is what guarantees contrast in every scheme.

### Where roles go on this site

| Element | Role |
| --- | --- |
| Page background | `surface` |
| Top app bar | `surface`, changing to `surface-container` once content scrolls under it |
| Event cards on the homepage | `surface-container-low` |
| Article cards on the event page | `surface` with an `outline-variant` border (outlined card) |
| Timeline rail and nodes | `outline-variant` line; `primary` node for a step, `outline` for an approximate date |
| Headlines, summaries | `on-surface` |
| Outlet name, timestamps, metadata | `on-surface-variant` |
| Links and text buttons | `primary` |
| Dividers | `outline-variant` |
| Snackbar | `inverse-surface` with `inverse-on-surface` |
| Dialog backdrop | `scrim` at 32% opacity |

## Typography

### Typefaces

| Token | Value | Notes |
| --- | --- | --- |
| `--md-ref-typeface-plain` | `"Roboto", "PingFang TC", "Noto Sans TC", "Noto Sans CJK TC", "Microsoft JhengHei", system-ui, sans-serif` | Roboto handles Latin and digits; the platform's own font handles Chinese |
| `--md-ref-typeface-brand` | Same as plain for the MVP | M3 allows a distinct brand face for display, headline and title-large; we don't need one yet |

Roboto is self-hosted (Latin subset; weights 400, 500, 700). Chinese is not a webfont: even sliced by `unicode-range`, Noto Sans TC's `@font-face` rules alone added about 430 KB of CSS, and macOS, iOS, Windows and Android all ship a good Traditional Chinese face. The UI is in English, so Chinese appears only in headlines and outlet names.

### Type scale

The M3 scale is five roles in three sizes. Tokens follow `--md-sys-typescale-<role>-<size>-<property>` with properties `font`, `size`, `line-height`, `weight`, `tracking`.

| Style | Size | Line height | Weight | Tracking |
| --- | --- | --- | --- | --- |
| display-large | 3.5625rem (57px) | 4rem | 400 | -0.015625rem |
| display-medium | 2.8125rem (45px) | 3.25rem | 400 | 0 |
| display-small | 2.25rem (36px) | 2.75rem | 400 | 0 |
| headline-large | 2rem (32px) | 2.5rem | 400 | 0 |
| headline-medium | 1.75rem (28px) | 2.25rem | 400 | 0 |
| headline-small | 1.5rem (24px) | 2rem | 400 | 0 |
| title-large | 1.375rem (22px) | 1.75rem | 400 | 0 |
| title-medium | 1rem (16px) | 1.5rem | 500 | 0.009375rem |
| title-small | 0.875rem (14px) | 1.25rem | 500 | 0.00625rem |
| body-large | 1rem (16px) | 1.5rem | 400 | 0.03125rem |
| body-medium | 0.875rem (14px) | 1.25rem | 400 | 0.015625rem |
| body-small | 0.75rem (12px) | 1rem | 400 | 0.025rem |
| label-large | 0.875rem (14px) | 1.25rem | 500 | 0.00625rem |
| label-medium | 0.75rem (12px) | 1rem | 500 | 0.03125rem |
| label-small | 0.6875rem (11px) | 1rem | 500 | 0.03125rem |

Sizes are in `rem` so they respect the reader's browser font size.

### Adjustments for Traditional Chinese

The UI is English and uses the M3 values as they are. Chinese text (headlines, outlet names, and the whole UI in the zh-TW locale) is marked with `lang`, and these deliberate deviations apply under `:lang(zh)`:

- **Tracking is 0** for all styles. Positive letter-spacing between Chinese characters looks wrong.
- **Body line height is looser:** body-large 1.75rem, body-medium 1.5rem. Dense square glyphs need more leading than Latin.
- **Summaries use body-large**, not body-medium. 14px Chinese body text is hard to read for longer passages.
- **Avoid label-small (11px) for Chinese text.** Use label-medium as the floor.
- **No italics** (Chinese has none) and no synthetic bold; use weight 500 or 700 from the loaded files.

### Where styles go

| Element | Style |
| --- | --- |
| Event title on the event page | headline-medium (headline-small on compact) |
| Event title on a homepage card | title-large |
| Section headings ("Timeline", "Coverage") | title-medium |
| Article headline | title-medium |
| Article summary | body-large |
| Timeline step text | body-large |
| Outlet name | label-large |
| Timestamps, counts | label-medium |
| Buttons, chips | label-large |

## Shape

| Token | Value | Used for |
| --- | --- | --- |
| `--md-sys-shape-corner-none` | 0 | Full-width bars |
| `--md-sys-shape-corner-extra-small` | 4px | Snackbars, tooltips, text fields, menus |
| `--md-sys-shape-corner-small` | 8px | Chips |
| `--md-sys-shape-corner-medium` | 12px | Cards |
| `--md-sys-shape-corner-large` | 16px | Navigation drawer, large sheets |
| `--md-sys-shape-corner-extra-large` | 28px | Dialogs, bottom sheets (top corners) |
| `--md-sys-shape-corner-full` | 9999px | Buttons, badges, search bar |

## Elevation

M3 has six levels, 0 to 5, corresponding to 0, 1, 3, 6, 8 and 12dp. In M3, depth is shown mainly by surface color, using the `surface-container-*` roles, with shadows as a secondary cue for things that float.

| Level | Surface role to use | Shadow | Used for |
| --- | --- | --- | --- |
| 0 | `surface` | None | Page, outlined cards |
| 1 | `surface-container-low` | Subtle | Elevated cards |
| 2 | `surface-container` | Light | Scrolled top app bar, menus |
| 3 | `surface-container-high` | Medium | Dialogs, snackbars, FAB |
| 4–5 | `surface-container-highest` | Strong | Dragged items; unlikely in the MVP |

Shadows are defined once as `--md-sys-elevation-level1` … `level5` box-shadow tokens using `--md-sys-color-shadow`. This is a flat, reading-focused site: almost everything sits at level 0, and elevation is reserved for overlays.

## States

Interactive elements show state with a **state layer**: an overlay in the element's content color (its `on-` color) at a fixed opacity. It is implemented once as a shared CSS class using a `::before` pseudo-element, and driven by `:hover`, `:focus-visible`, `:active` and Base UI's data attributes.

| State | Opacity token | Value |
| --- | --- | --- |
| Hover | `--md-sys-state-hover-state-layer-opacity` | 0.08 |
| Focus | `--md-sys-state-focus-state-layer-opacity` | 0.12 |
| Pressed | `--md-sys-state-pressed-state-layer-opacity` | 0.12 |
| Dragged | `--md-sys-state-dragged-state-layer-opacity` | 0.16 |

- **Disabled:** content at 38% opacity of `on-surface`, container at 12%. No state layer.
- **Focus indicator:** a visible ring on `:focus-visible` only, 3px, `secondary` color, 2px offset, following the element's shape.
- **Ripple:** M3's pressed state includes a ripple. The MVP ships the pressed state layer without the ripple; add a ripple later as one shared hook if wanted, not per component.

## Motion

Baseline M3 easing and duration tokens, used for all CSS transitions.

| Easing token | Value | Use |
| --- | --- | --- |
| `--md-sys-motion-easing-standard` | `cubic-bezier(0.2, 0, 0, 1)` | Small utility transitions: state layers, color changes |
| `--md-sys-motion-easing-standard-decelerate` | `cubic-bezier(0, 0, 0, 1)` | Entering |
| `--md-sys-motion-easing-standard-accelerate` | `cubic-bezier(0.3, 0, 1, 1)` | Exiting |
| `--md-sys-motion-easing-emphasized-decelerate` | `cubic-bezier(0.05, 0.7, 0.1, 1)` | Dialogs, sheets and menus entering |
| `--md-sys-motion-easing-emphasized-accelerate` | `cubic-bezier(0.3, 0, 0.8, 0.15)` | The same, exiting |
| `--md-sys-motion-easing-linear` | `cubic-bezier(0, 0, 1, 1)` | Opacity fades, progress |

| Duration tokens | Values | Use |
| --- | --- | --- |
| `short1`–`short4` | 50, 100, 150, 200ms | State changes, small elements |
| `medium1`–`medium4` | 250, 300, 350, 400ms | Menus, dialogs, expanding cards |
| `long1`–`long4` | 450, 500, 550, 600ms | Full-screen transitions |
| `extra-long1`–`extra-long4` | 700, 800, 900, 1000ms | Rare; ambient only |

Rules: exits are shorter than entrances; enter with decelerate, exit with accelerate. Under `prefers-reduced-motion: reduce`, all durations collapse to near zero and movement is replaced by a simple fade.

## Layout

### Window size classes

M3 defines five breakpoints. We use them as the only media query values, exposed as custom media or Sass-free constants in `base.css`.

| Class | Width | Layout here |
| --- | --- | --- |
| Compact | under 600px | One column, 16px margins. Timeline above coverage |
| Medium | 600–839px | One column, 24px margins, wider cards |
| Expanded | 840–1199px | Event page becomes two panes: timeline (sticky) and coverage |
| Large | 1200–1599px | Same two panes, content max-width applied |
| Extra-large | 1600px and up | Same, centered; never stretch text lines |

- Spacing uses a 4px base: 4, 8, 12, 16, 24, 32, 48. Gaps between cards are 8px on compact and 16px above.
- Text line length is capped at about 40 Chinese characters (roughly 40em at body-large), whatever the window size.
- The two-pane event page follows M3's supporting-pane canonical layout: coverage is the primary pane, the timeline is the supporting pane.

### Navigation

The MVP has three destinations at most (Events, Outlets, About), and M3 navigation bars and rails are meant for three to five primary destinations of equal weight. Ours aren't equal; nearly all use is the event list. So the MVP uses a top app bar with the site name and a small set of trailing actions, not a navigation bar or rail. Revisit when search, topics or saved events arrive.

## Accessibility

- Contrast: text at least 4.5:1, large text and meaningful non-text elements at least 3:1. Correct role pairing provides this; the theme script asserts it in CI for every generated scheme.
- Touch targets at least 48×48 CSS px, even when the visible element is smaller (icon buttons are 40px visible with a 48px hit area).
- Every page works by keyboard alone; Base UI provides focus management, roving tabindex and ARIA for the primitives it covers.
- Semantic HTML first: `<article>` per article card, `<ol>` for the timeline, `<time datetime>` for every date, one `<h1>` per page.
- Set `lang` on `<html>` from the UI locale, and `lang="zh-Hant-TW"` on embedded Chinese text (headlines, outlet names) so screen readers switch voice and the Chinese type adjustments apply.
- Respect `prefers-reduced-motion`, `prefers-contrast` and `prefers-color-scheme`.
- Approximate dates are marked in text ("c." in English, "約" in Chinese), never by color or icon alone.

## References

- Color roles: <https://m3.material.io/styles/color/roles>
- Type scale: <https://m3.material.io/styles/typography/type-scale-tokens>
- Shape: <https://m3.material.io/styles/shape/corner-radius-scale>
- Elevation: <https://m3.material.io/styles/elevation/overview>
- Motion: <https://m3.material.io/styles/motion/easing-and-duration/tokens-specs>
- States: <https://m3.material.io/foundations/interaction/states/state-layers>
- Breakpoints: <https://m3.material.io/foundations/layout/breakpoints>
- Canonical layouts: <https://m3.material.io/foundations/layout/canonical-layouts/overview>
- Token sources: <https://github.com/material-components/material-web/tree/main/tokens>
- Theme Builder: <https://material-foundation.github.io/material-theme-builder/>
