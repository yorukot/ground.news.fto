# Ground News for Taiwan

Every outlet's coverage of the same event, side by side, with a short summary of each article and a timeline of the event. The site never labels or scores an outlet or an article; readers compare for themselves.

- [plan.md](plan.md): what the product is and how the pipeline works
- [docs/](docs/README.md): tech stack, Material Design 3 design system, components and pages

## Status

| Part | State |
| --- | --- |
| Database schema, migrations, sqlc queries | Done |
| Read API (`/api/v1/events`, `/events/{id}`, `/outlets`) | Done, with handler tests |
| Seed data (outlets, fictional sample events) | Done |
| Web app: design system, components, home, event, outlets, about; English and Traditional Chinese | Done, with component tests |
| Ingestion pipeline (crawl, dedupe, summarize, link) | Done; run end to end on three outlets with the real model |
| Admin tool (review link decisions, merge events, move articles) | Done, with end-to-end tests |
| Docker images, production compose, CI workflow | Done; images build. The CI workflow has not run on GitHub yet |
| Outlet coverage | 18 of 20 outlets: 11 in full coverage, 7 headline-only because they refuse or restrict AI use; see [docs/outlets.md](docs/outlets.md) |

## Run it locally

Needs Go 1.25+, Node 22+, pnpm and Docker.

```sh
cp .env.example .env
make dev-db      # Postgres on 127.0.0.1:5433
make migrate
make seed        # outlets plus fictional sample events
make serve       # API on 127.0.0.1:8080
make worker      # optional: crawl, summarize and link real articles
```

Without `OPENAI_API_KEY` the worker uses a fake model, which is enough to exercise the pipeline. Set `ADMIN_TOKEN` to enable the admin tool at `/admin`.

In another terminal:

```sh
cd web
pnpm install
pnpm dev         # http://localhost:5173
```

## Checks

```sh
make test-db && make test          # Go tests; the database-backed ones need TEST_DATABASE_URL
cd web && pnpm typecheck && pnpm check && pnpm test
cd web && E2E_BASE_URL=http://127.0.0.1:3000 E2E_ADMIN_TOKEN=... pnpm e2e   # against a running, seeded site
```

After changing `api/openapi.yaml` run `pnpm api-types` in `web/`. After changing SQL run `make sqlc`. After changing the theme seed run `pnpm theme` in `web/`.
