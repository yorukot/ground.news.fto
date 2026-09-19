import { useState } from "react";
import { ArticleCard } from "~/components/ArticleCard/ArticleCard";
import { Button } from "~/components/Button/Button";
import { Timeline } from "~/components/Timeline/Timeline";
import { useI18n } from "~/i18n/i18n";
import { getEvent } from "~/lib/api.server";
import { messagesFor, pageMeta } from "~/lib/meta";
import type { Route } from "./+types/event";
import styles from "./event.module.css";

export async function loader({ params, request }: Route.LoaderArgs) {
  return getEvent(params.id, request.signal);
}

export function meta({ loaderData, matches }: Route.MetaArgs) {
  const m = messagesFor(matches);
  if (!loaderData) return pageMeta(m, m.error.notFoundTitle, m.error.notFoundBody);
  const latest = loaderData.timeline.at(-1)?.development ?? "";
  return pageMeta(m, loaderData.title, m.event.description(loaderData.title, latest));
}

export default function EventPage({ loaderData: event }: Route.ComponentProps) {
  const { m, fmt } = useI18n();
  // Coverage is ordered by time only, never by outlet.
  const [oldestFirst, setOldestFirst] = useState(false);
  const articles = oldestFirst ? [...event.articles].reverse() : event.articles;

  return (
    <>
      <header className={styles.header}>
        <h1 className={`${styles.title} md-headline-medium`}>{event.title}</h1>
        <p className={`${styles.meta} md-label-medium`}>
          <span>
            {m.event.firstSeen}{" "}
            <time dateTime={event.firstSeenAt}>{fmt.date(event.firstSeenAt)}</time>
          </span>
          <span>
            {m.event.updated}{" "}
            <time dateTime={event.updatedAt}>{fmt.dateTime(event.updatedAt)}</time>
          </span>
        </p>
      </header>

      <div className={styles.panes}>
        <section className={styles.timeline} aria-labelledby="timeline-heading">
          <h2 id="timeline-heading" className={`${styles.sectionHeading} md-title-medium`}>
            {m.event.timeline}
          </h2>
          <Timeline steps={event.timeline} />
        </section>

        <section className={styles.coverage} aria-labelledby="coverage-heading">
          <div className={styles.coverageHeader}>
            <div>
              <h2 id="coverage-heading" className="md-title-medium">
                {m.event.coverage}
              </h2>
              <p className={`${styles.count} md-label-medium`}>
                {m.event.coverageCount(event.articleCount, event.outletCount)}
              </p>
            </div>
            <Button
              variant="text"
              aria-label={`${m.event.sortLabel}: ${oldestFirst ? m.event.oldestFirst : m.event.newestFirst}`}
              onClick={() => setOldestFirst((value) => !value)}
            >
              {oldestFirst ? m.event.oldestFirst : m.event.newestFirst}
            </Button>
          </div>
          <ul className={styles.articles}>
            {articles.map((article) => (
              <li key={article.id}>
                <ArticleCard article={article} />
              </li>
            ))}
          </ul>
        </section>
      </div>
    </>
  );
}
