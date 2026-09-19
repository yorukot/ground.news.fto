# Docs

Design and architecture docs for Ground News for Taiwan. The code follows these docs; when the two disagree, fix one of them.

| Doc | What it covers |
| --- | --- |
| [`../plan.md`](../plan.md) | The MVP plan: what the product is and how the pipeline works |
| [tech-stack.md](tech-stack.md) | Backend (Go), frontend (React + Vite + Base UI), infrastructure, repo layout |
| [design-system.md](design-system.md) | Material Design 3 foundations: color, type, shape, elevation, motion, layout, states, accessibility |
| [components.md](components.md) | Each M3 component we need, the Base UI primitive behind it, and the page designs |
| [outlets.md](outlets.md) | Full and headline-only coverage, the rule that decides between them, and every outlet's status |

## Decisions in one table

| Area | Decision |
| --- | --- |
| Backend language | Go, one binary with `serve`, `worker` and `migrate` subcommands |
| Database | Postgres, with full-text search for event linking and pg\_trgm for entity alias lookup. No vector search in the MVP |
| Job pipeline | River (Postgres-backed queue); no Redis, no separate broker |
| AI provider | OpenAI only, via the official `openai-go` SDK, behind one small interface |
| Frontend | React + TypeScript + Vite, using React Router framework mode for SSR |
| UI primitives | Base UI (`@base-ui/react`), unstyled; all visuals are ours |
| Design language | Material Design 3, implemented as `--md-sys-*` CSS custom properties |
| Styling | CSS Modules + design tokens; no Tailwind, no CSS-in-JS |
| Theme | Generated from one seed color with `@material/material-color-utilities`; light, dark, and higher-contrast schemes |
| Language | English UI by default, with a typed i18n layer (`web/app/i18n/`); Traditional Chinese is the second locale. Original headlines and outlet names stay in Chinese |

## Notes from reviewing the plan

1. **"SSR website" vs "React + Vite".** Plain Vite builds a client-rendered SPA. A news site needs server-rendered HTML for search engines and for link previews on LINE, Facebook and Threads, which do not run JavaScript. React Router's framework mode is a Vite plugin that adds SSR, so it satisfies both the plan and the stack choice. It does mean one small Node process in production next to the Go API. See [tech-stack.md](tech-stack.md#rendering-ssr-with-react-router-framework-mode). **This is the one decision to confirm before scaffolding.**
2. **Brand color must be politically neutral.** In Taiwan, blue and green read as KMT and DPP, turquoise as TPP, yellow as NPP, and red carries its own weight. The site's core principle is that it never labels anyone, so the theme color should not look like a party color either. See [design-system.md](design-system.md#color).
3. **Outlets get no colors.** Per-outlet color coding would be a silent label. Outlets are identified by name (and favicon at most), all rendered in the same neutral roles.
4. **No embeddings in the MVP.** The plan first used embeddings to find candidate events for a new article. Since titles, timeline lines and recaps are all our own English text, Postgres full-text search plus shared entities does that job with no extra model, cost or extension. The model still makes the link decision. `events.timeline_text` must be rewritten whenever an event gains a step, so the search index stays current.
5. **Places are entities too.** Typhoons, earthquakes and local crime stories often name no person or organization but always name a place, so `place` is an entity kind alongside `person` and `organization`.
6. **Chinese text and MinHash.** Shingle on character n-grams (e.g. 5-character windows) rather than words, so no word segmenter is needed.
7. **Language.** Decided after the plan was written: the site is in English, with i18n so other languages can be added. Headlines are shown unchanged, so they stay in Chinese and are marked with `lang="zh-Hant-TW"`. Summaries and timeline lines are the site's own text and are written in English. The summary rule "keep the article's own terms" then means translating terms faithfully ("the mainland" stays "the mainland", never "China").

## Open questions

- **Outlet permissions.** Seven outlets refuse or restrict AI use, so they get headline-only coverage: headline and link, no page fetch, nothing given to the model; see [outlets.md](outlets.md). Asking them for permission would give them full coverage.
- **Topic scope.** Several outlets' sitemaps mix sports, entertainment and lifestyle with news. What the site should cover is a product decision; a filter is easy to add once it's made.

- The seed color is the M3 baseline violet (`#6750A4`) for now; try alternatives in Material Theme Builder before launch. Constraints are in [design-system.md](design-system.md#color).
- The site name ("Ground News Taiwan" / "新聞並陳") is a placeholder, defined only in `web/app/i18n/messages/`.
- Should summaries and timeline lines also be produced in Chinese for the zh-TW locale? Today they exist in one language (English); the UI chrome is what gets translated.
- Hosting target (single VPS with Docker Compose is assumed).
- The summary model is `rlongAI/gpt-5.5` through a LiteLLM proxy, which exposes aliases rather than dated snapshots; revisit if a pinned snapshot becomes available.

## Working rule: read the docs first

Before designing or building any component, read the current spec rather than working from memory. Material Design 3 and Base UI both change.

- Material Design 3: <https://m3.material.io> (components, styles, foundations). The site is JavaScript-rendered; if a tool can't read it, the same token values live in the official token sources at <https://github.com/material-components/material-web/tree/main/tokens>.
- Base UI: <https://base-ui.com/react/overview/quick-start>, and the machine-readable index at <https://base-ui.com/llms.txt> (every component page has a `.md` version).
- React Router: <https://reactrouter.com>
- Material Color Utilities: <https://github.com/material-foundation/material-color-utilities>
- River: <https://riverqueue.com/docs> · pgx: <https://github.com/jackc/pgx> · sqlc: <https://docs.sqlc.dev> · openai-go: <https://github.com/openai/openai-go>
