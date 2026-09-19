import { useEffect, useState } from "react";
import { useFetcher } from "react-router";
import { Button } from "~/components/Button/Button";
import { EventCard } from "~/components/EventCard/EventCard";
import { useI18n } from "~/i18n/i18n";
import { type EventSummary, listEvents } from "~/lib/api.server";
import { messagesFor, pageMeta } from "~/lib/meta";
import type { Route } from "./+types/home";
import styles from "./home.module.css";

export async function loader({ request }: Route.LoaderArgs) {
  const cursor = new URL(request.url).searchParams.get("cursor");
  return listEvents(cursor, request.signal);
}

export function meta({ matches }: Route.MetaArgs) {
  const m = messagesFor(matches);
  return pageMeta(m, null, m.site.description);
}

export default function Home({ loaderData }: Route.ComponentProps) {
  const { m, locale } = useI18n();
  const more = useFetcher<typeof loader>();
  const [extra, setExtra] = useState<EventSummary[]>([]);
  const [nextCursor, setNextCursor] = useState(loaderData.nextCursor);

  // A fresh first page (navigation or revalidation) resets what was appended.
  useEffect(() => {
    setExtra([]);
    setNextCursor(loaderData.nextCursor);
  }, [loaderData]);

  useEffect(() => {
    const page = more.data;
    if (!page) return;
    setExtra((current) => {
      const seen = new Set(current.map((event) => event.id));
      return [...current, ...page.events.filter((event) => !seen.has(event.id))];
    });
    setNextCursor(page.nextCursor);
  }, [more.data]);

  const firstIds = new Set(loaderData.events.map((event) => event.id));
  const events = [...loaderData.events, ...extra.filter((event) => !firstIds.has(event.id))];
  const [featured, ...stories] = events;
  const loading = more.state !== "idle";

  return (
    <div className={styles.page}>
      <section className={styles.hero} aria-labelledby="home-heading">
        <section className={styles.intro}>
          {locale === "zh-TW" && (
            <p className={`${styles.kicker} md-label-large`}>{m.home.kicker}</p>
          )}
          <h1 id="home-heading" className={styles.heading} aria-label={m.home.brandLabel}>
            {locale === "zh-TW" ? (
              <span className={styles.zhWordmark} aria-hidden="true">
                <span>{m.home.title}</span>
                <span className={styles.zhWindow}>
                  <span className={styles.wordTrack}>
                    {[...m.home.rotatingWords, m.home.rotatingWords[0]].map((word, index) => (
                      <span key={`${word}-${index}`}>{word}</span>
                    ))}
                  </span>
                </span>
              </span>
            ) : (
              <span className={styles.enWordmark} aria-hidden="true">
                <span>
                  <strong>W</strong>hat
                </span>
                <span>
                  <strong>T</strong>he
                </span>
                <span className={styles.enChanging}>
                  <strong>F</strong>
                  <span className={styles.enWindow}>
                    <span className={styles.wordTrack}>
                      {[...m.home.rotatingWords, m.home.rotatingWords[0]].map((word, index) => (
                        <span key={`${word}-${index}`}>{word}</span>
                      ))}
                    </span>
                  </span>
                </span>
              </span>
            )}
          </h1>
          <p className={`${styles.description} md-body-large`}>{m.home.intro}</p>
          {events.length === 0 && <p className={`${styles.empty} md-body-large`}>{m.home.empty}</p>}
          <div className={styles.promise} aria-hidden="true">
            <span />
            <span />
            <span />
          </div>
        </section>
        {featured && <EventCard event={featured} variant="featured" />}
      </section>

      {stories.length > 0 && (
        <section className={styles.feed} aria-labelledby="latest-heading">
          <header className={styles.sectionHeader}>
            <div>
              <p className={`${styles.sectionKicker} md-label-medium`}>{m.home.kicker}</p>
              <h2 id="latest-heading" className={styles.sectionTitle}>
                {m.home.latest}
              </h2>
            </div>
            <p className={`${styles.sectionIntro} md-body-medium`}>{m.home.latestIntro}</p>
          </header>
          <ul className={styles.grid}>
            {stories.map((event) => (
              <li key={event.id}>
                <EventCard event={event} />
              </li>
            ))}
          </ul>
        </section>
      )}
      {nextCursor && (
        // A plain GET form: without JavaScript it navigates to the next page.
        <more.Form method="get" action="/?index" className={styles.more}>
          <input type="hidden" name="cursor" value={nextCursor} />
          <Button type="submit" variant="tonal" disabled={loading} focusableWhenDisabled>
            {loading ? m.home.loadingMore : m.home.loadMore}
          </Button>
        </more.Form>
      )}
    </div>
  );
}
