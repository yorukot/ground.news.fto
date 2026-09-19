# Outlets and crawling policy

The site summarizes articles with an AI model. Many publishers now state a position on that use, so deciding how to cover an outlet takes more than checking whether `robots.txt` lets a crawler in.

## Two kinds of coverage

| | Full | Headline-only |
| --- | --- | --- |
| For | Outlets with no objection to AI use | Outlets that refuse or restrict AI use of their content |
| What is fetched | The article page | Only the feed or news sitemap. **The article page is never fetched** |
| What the model sees | The article | **Nothing** |
| What the site shows | Original headline, our summary, link, place on the timeline | Original headline, link, and a note that the outlet doesn't allow AI summaries |
| How it joins an event | The model decides, from candidates found by entities and full-text search | String matching only: known entity names found in the headline (see below) |

Headline-only coverage is what a search engine or a link aggregator does: it lists a page by its title and sends the reader there. It keeps those outlets in the side-by-side comparison, which is the point of the site, without using their text in any way they have objected to.

The code enforces the boundary rather than trusting configuration:

- `internal/crawl` takes a purpose with every fetch. A site whose `robots.txt` carries `Content-Signal: ai-input=no` is refused for `ForAI` and allowed only for `ForIndex`, so an outlet that adds the signal later is respected automatically.
- The fetch job cancels itself for a headline-only outlet, even if it was enqueued before the outlet's coverage changed.
- Headline-only articles are stored with an empty body, and the summarize and link jobs are never enqueued for them.

We also never republish article text (the API has no field for it), always show the original headline unchanged with a link to the original, identify the crawler honestly in its user agent, and wait at least 3 seconds between requests to one site.

## Deciding an outlet's coverage

1. `robots.txt` must allow our user agent to fetch what we fetch (feeds and sitemaps for both kinds; article pages for full coverage).
2. The outlet gets **headline-only** coverage if any of these is true:
   - a `Content-Signal` with `ai-input=no` (the [Content Signals](https://contentsignals.org) vocabulary: `ai-input` covers retrieval, grounding and summarization; `ai-train` covers training, which we never do);
   - a statement in `robots.txt` or its terms that content may not be used for AI, LLM or machine-learning purposes;
   - it blocks the named AI crawlers (GPTBot, ClaudeBot, CCBot and the like) even though it allows `*`. That is ambiguous about a site like this one, and we resolve the doubt in the outlet's favor.
3. For full coverage, `app crawl-check <slug>` must show clean extraction: the right headline, the whole body, and nothing but the body.

If an outlet grants permission, it moves to full coverage by changing one field in `internal/seed/outlets.go`.

## Status

Checked 2026-09-19. Re-read an outlet's `robots.txt` before changing its coverage.

| Outlet | Coverage | Basis |
| --- | --- | --- |
| 中央社 CNA | Full | `Content-signal: search=yes, ai-input=yes, ai-train=no`: explicit permission for this use |
| 公視新聞網 PTS | Full | No AI restrictions |
| ETtoday新聞雲 | Full | No AI restrictions |
| 三立新聞網 SETN | Full | No AI restrictions |
| TVBS新聞網 | Full | No AI restrictions (blocks only Amazonbot) |
| 東森新聞 EBC | Full | No AI restrictions |
| 民視新聞網 FTV | Full | No AI restrictions |
| 中天新聞網 CTi | Full | No AI restrictions |
| 風傳媒 Storm | Full | No AI restrictions; publishes an `llms.txt` |
| NOWnews今日新聞 | Full | No AI restrictions |
| 鏡週刊 Mirror Media | Full | No AI restrictions |
| 聯合新聞網 UDN | Headline-only | `robots.txt` states its content may not be used for LLM, machine-learning or AI purposes |
| Newtalk新聞 | Headline-only | `Content-Signal: ai-input=no` |
| 報導者 The Reporter | Headline-only | Reserves rights against AI, LLM and text-and-data-mining use; says licensing may be available |
| 自由時報 LTN | Headline-only | Allows `*` but blocks every named AI crawler |
| 中時新聞網 China Times | Headline-only | Allows `*` but blocks every named AI crawler |
| 關鍵評論網 The News Lens | Headline-only | Allows `*` but blocks named AI crawlers, including `ChatGPT-User` |
| 上報 Up Media | Headline-only | Allows `*` but blocks named AI crawlers |
| 天下雜誌 CommonWealth | Not crawled | No restrictions, but its RSS feed is dead (latest item 2021) and it lists no sitemap |
| Yahoo奇摩新聞 | Not crawled | Aggregator, and it blocks AI crawlers. The code can credit an aggregator's articles to the original outlet, but it isn't configured |

Eleven outlets have full coverage and seven are headline-only. Between them they span the spectrum the site needs: for example CTi and TVBS alongside SETN and FTV in full, with China Times, UDN and LTN present by headline.

## How a headline joins an event without a model

`link.MatchHeadline` looks for known entity names (people, organizations, places the model has already extracted from full-coverage articles) inside the headline, and attaches the article to the recent event that shares the most of them. It is deliberately conservative, because no model checks its work:

- a name must be at least three characters; two-character strings such as 台海 or 王男 are everywhere;
- the headline must share at least **two** entities with the event. One shared name never joins events;
- only events updated in the last 72 hours are considered;
- a tie between two events is not guessed at;
- it never creates an event, never adds a timeline step, and never moves an event up the homepage;
- the match is stored with a confidence of 0.5 or 0.7, so it appears near the top of the admin review list.

A headline usually arrives before any full-coverage outlet has reported the same story, so an unmatched headline is retried every 30 minutes for 24 hours, then left unlisted.

The cost of this caution is recall: many headline-only articles never join an event, and those outlets will look quieter on the site than they are. Permission from an outlet fixes that for the outlet; nothing else should.

## Known limits

- SETN, CTi, NOWnews, FTV and Storm don't put the section in the URL, so their sports, entertainment and lifestyle pieces are crawled too. They cost model calls and create events nobody needs. A topic filter (in the summarize step, or by sitemap keywords) is the fix, and what counts as in scope is a product decision.
- Asking for permission is still worth doing: for the outlets that refuse, and to be on firm ground with the ones that merely say nothing.

## Adding an outlet

1. Read its `robots.txt` and terms and decide the coverage by the rules above.
2. Add `Crawl: &crawl.Config{...}` to its entry in `internal/seed/outlets.go`: feeds or sitemaps (a news sitemap with `<news:title>` is ideal, and required in practice for headline-only), an `ArticleURLPattern` that keeps to news sections where the URL allows it, and selectors if readability alone isn't clean. Set `Coverage: CoverageHeadline` and the reason in `NotCrawled` where that applies.
3. Run `go run ./cmd/app crawl-check <slug>`. For full coverage, read the headline and the first and last lines of the body. For headline-only it lists titles and fetches no article page.
4. `make seed` applies the config to a database where the outlet has none, and always applies the coverage. To change an existing crawl config, update the `outlets` row.
