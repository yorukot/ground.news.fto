# Components and pages

Each UI component is an M3 component by spec, built on a Base UI primitive where one exists and on plain semantic HTML where none does. Components live in `web/app/components/`, one folder each, with a `.tsx` file and a `.module.css` file that uses only tokens from [design-system.md](design-system.md).

**Before building any component:** read its M3 spec page (anatomy, measurements, states, accessibility) and the Base UI page for the primitive. Record the spec link at the top of the component file. Measurements below are from the M3 specs but are a summary, not a substitute.

## Component map

### Needed for the public site

| M3 component | Built on | Key spec points | Used for |
| --- | --- | --- | --- |
| [Top app bar](https://m3.material.io/components/top-app-bar/overview) (small) | `<header>`, plain HTML | 64px tall; `surface`, becoming `surface-container` on scroll; title in title-large | Site header |
| [Card](https://m3.material.io/components/cards/overview), outlined and filled | `<article>`, plain HTML | 12px corners; 16px padding; outlined uses an `outline-variant` border; whole-card click targets get a state layer | Event cards, article cards |
| [Button](https://m3.material.io/components/buttons/overview): filled, tonal, outlined, text | Base UI `Button`; links that look like buttons are plain anchors (`ButtonLink`, `ExternalButtonLink`) | 40px tall; full corners; label-large; 24px side padding (text button 12px) | "Read original", pagination, admin actions |
| [Icon button](https://m3.material.io/components/icon-buttons/overview) | Base UI `Button` | 40px visible, 48px target; 24px icon; needs `aria-label` | Theme toggle, close, overflow |
| [Chip](https://m3.material.io/components/chips/overview): assist | A plain anchor; on this site an assist chip always navigates | 32px tall; 8px corners; label-large; outlined | Outlet chips on a timeline step, linking to that outlet's article |
| [Chip](https://m3.material.io/components/chips/overview): filter | Base UI `Toggle` and `Toggle Group` | Same, with a selected state using `secondary-container` and a leading check icon | Filtering coverage by outlet (after MVP) |
| [Divider](https://m3.material.io/components/divider/overview) | Base UI `Separator` | 1px, `outline-variant` | Between sections |
| [List](https://m3.material.io/components/lists/overview) | `<ul>`/`<ol>`, plain HTML | One-, two- and three-line items; 56, 72 and 88px minimum heights | Outlet list, reprint list |
| [Tooltip](https://m3.material.io/components/tooltips/overview), plain | Base UI `Tooltip` | `inverse-surface`; body-small; 4px corners | Icon button labels, full timestamps |
| [Menu](https://m3.material.io/components/menus/overview) | Base UI `Menu` | `surface-container`; level 2; 4px corners; 48px items | Settings menu: appearance and language |
| [Snackbar](https://m3.material.io/components/snackbar/overview) | Base UI `Toast` | `inverse-surface`; 4px corners; level 3; one at a time; bottom of the window | "Link copied", load errors |
| [Progress indicator](https://m3.material.io/components/progress-indicators/overview), linear | Base UI `Progress` | 4px track; `primary` on `secondary-container` | Route navigation pending state, under the app bar |
| [Tabs](https://m3.material.io/components/tabs/overview), primary | Base UI `Tabs` | 48px tall; `primary` active indicator; title-small labels | Compact event page: switching between Timeline and Coverage, if stacking proves too long |
| Skeleton placeholders | Plain HTML | Not an M3 component; use `surface-container-highest` blocks with card shapes | Loading states |

### Needed for the admin tool

| M3 component | Built on | Used for |
| --- | --- | --- |
| [Dialog](https://m3.material.io/components/dialogs/overview), basic | Base UI `Dialog` and `Alert Dialog` (280–560px wide; 28px corners; `surface-container-high`; 24px padding; scrim behind) | Confirming a merge or a move |
| [Text field](https://m3.material.io/components/text-fields/overview), outlined | Base UI `Field` + `Input` (56px tall; 4px corners; floating label; supporting and error text through `Field`) | Login, search by event title |
| Event picker | The outlined text field plus a native radio group of matches, inside the dialog | Picking the target event for a merge or move. A Base UI `Combobox` would be the fuller M3 search pattern; the radio list was simpler, keyboard-accessible as is, and shows the article count beside each match |
| Form handling | Base UI `Form` + React Router actions | Every admin mutation |
| Data table | `<table>`, plain HTML, styled with list tokens | Recent events and low-confidence links |

### Not in the MVP

Navigation bar, navigation rail, navigation drawer, FAB, bottom sheet, carousel, date picker, slider, badges. None has a job yet. See the navigation note in [design-system.md](design-system.md#navigation).

## Shared building blocks

Three pieces are written once and reused by every interactive component:

1. **`StateLayer`** (a CSS class): the `::before` overlay that implements hover, focus and pressed opacity from the state tokens.
2. **`FocusRing`** (a CSS class): the `:focus-visible` indicator.
3. **`Typography`** (CSS classes `.md-body-large` and so on, in `theme/typography.css`): one class per type style, applying all five properties, including the zh-TW adjustments.

Pattern for wrapping a Base UI primitive:

- The wrapper exposes M3 vocabulary (`variant="filled" | "tonal" | "outlined" | "text"`), not styling props.
- State styling uses Base UI's data attributes in the CSS Module (`[data-disabled]`, `[data-pressed]`, `[data-open]`, `[data-checked]`), not JavaScript.
- Enter and exit animations use `[data-starting-style]` and `[data-ending-style]` with motion tokens.
- Popups are positioned by Base UI's `Positioner` and sized with the CSS variables it provides.
- Use the `render` prop to compose primitives, for example an `IconButton` as a `Menu.Trigger`. Do not use it to turn a `Button` into a link; Base UI says links keep their own element.
- No user-visible string is written in a component. Text comes from `useI18n()`; see `web/app/i18n/`.

## The timeline (custom component)

M3 has no timeline component, so this is composed from M3 parts and tokens rather than invented from scratch.

- Structure: an `<ol>`, one `<li>` per step, oldest first.
- Each step: a node on a vertical rail, the date in label-large (`on-surface-variant`), the step text in body-large, then a row of assist chips, one per outlet that reported the development. Each chip jumps to that outlet's article card.
- Rail: 1px `outline-variant`. Node: 12px circle, `primary` fill. A step with an approximate date uses a hollow node with an `outline` border **and** a text marker before the date ("c." in English, "約" in Chinese), so the meaning never depends on the visual alone.
- The first article to report a step is not visually marked as "first" or "exclusive". Chips are ordered by publish time, and that is the only signal.
- A long dormant gap between steps is shown as a plain text line on the rail, such as "3 months later", in label-medium. It helps readers understand a reactivated event.
- More than about 8 steps: show the latest 5, with a text button to expand the earlier ones (Base UI `Collapsible`).

## Pages

### Home: event list

- Top app bar, then a single column of event cards (filled cards, `surface-container-low`), most recently updated first.
- Event card: title (title-large), the latest timeline step as one line (body-large), then metadata in label-medium: number of outlets covering it, number of articles, and last update time. The whole card is one link.
- Layout: one column through Medium, capped at about 720px; two columns of cards from Expanded up.
- Pagination: a "load more" tonal button backed by the API cursor. No infinite scroll, so the footer stays reachable and position is predictable.
- A reactivated old event looks like any other card; its metadata line includes when it started, such as "Since Mar 2026".

### Event page

- Header: event title as `<h1>` (headline-medium), then metadata: first seen, last updated, outlet count.
- **Timeline** section, then **Coverage** section. Compact and Medium stack them. Expanded and up uses two panes: coverage as the primary pane, the timeline as a sticky supporting pane about 360px wide.
- Coverage is a list of article cards (outlined), ordered by publish time, newest first, with a text toggle for oldest first. Never ordered or grouped by outlet.
- Article card, in this order, matching the plan:
  1. Outlet name (label-large, `on-surface-variant`) and published time (`<time>`, label-medium)
  2. Original headline, unchanged (title-medium)
  3. Summary, 2–4 sentences (body-large)
  4. A text button, "Read the original", opening the original in a new tab with `rel="noopener"`
- Wire reprints: one card for the story. Below the headline, a line such as "Also run by 5 other outlets" expands (Base UI `Collapsible`) into a list of the other outlets, each linking to its own copy.
- Articles with no timeline step look identical to those with one. The timeline chips link into the cards; the cards themselves carry no "step" badge.
- Meta tags from the route: title, a description built from the event title and latest step, and Open Graph tags for link previews.

### Outlets and About

- Outlets: a plain two-line list with name and domain, alphabetical by name. No descriptions or categories of our own; describing an outlet would be labeling it.
- About: what the site does, the no-labeling principle, how summaries and timelines are produced (including that a model writes them), and how to report an error.

### Admin

Behind login, not linked from the public site, `noindex`. Same components and theme.

- A table of recent link decisions, sorted so low-confidence ones come first.
- Event view with "merge another event into this one" and per-article "move" actions. Each opens a dialog with the event picker (a move can also name a new event); the dialog is the confirmation. The result is shown as a status notice at the top of the page after the redirect. A snackbar (Base UI `Toast`) is the M3 pattern for this and is still to build, along with the "link copied" one on the public site.

## Definition of done for a component

- Matches the M3 spec for anatomy, measurements and all states (enabled, hover, focus, pressed, disabled, plus selected or open where relevant).
- Uses tokens only.
- Works in light, dark and high-contrast schemes, and with reduced motion.
- Keyboard-operable, with a visible focus ring, a 48px target and correct names and roles; passes `axe-core`.
- Renders correctly on the server with no hydration warnings.
- Every string comes from the message catalogue, and the component reads correctly in both locales.
- Checked with Traditional Chinese text, including long headlines that wrap.
