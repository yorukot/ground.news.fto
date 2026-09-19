// An event on the homepage: a filled card that is one link to the event page.
import { Link } from "react-router";
import { useI18n } from "~/i18n/i18n";
import type { EventSummary } from "~/lib/api.server";
import { isLongRunning } from "~/lib/format";
import styles from "./EventCard.module.css";

export function EventCard({ event }: { event: EventSummary }) {
  const { m, fmt } = useI18n();
  return (
    <article className={styles.card}>
      <Link
        to={`/events/${event.id}`}
        className={`${styles.link} md-state-layer md-focus-ring`}
        prefetch="intent"
      >
        <h2 className="md-title-large">{event.title}</h2>
        {event.latestDevelopment && (
          <p className={`${styles.latest} md-body-large`}>{event.latestDevelopment}</p>
        )}
        <p className={`${styles.meta} md-label-medium`}>
          <span>{m.eventCard.outlets(event.outletCount)}</span>
          <span>{m.eventCard.articles(event.articleCount)}</span>
          <span>
            {m.eventCard.updated}{" "}
            <time dateTime={event.updatedAt}>{fmt.dateTime(event.updatedAt)}</time>
          </span>
          {isLongRunning(event.firstSeenAt, event.updatedAt) && (
            <span>
              {m.eventCard.since}{" "}
              <time dateTime={event.firstSeenAt}>{fmt.yearMonth(event.firstSeenAt)}</time>
            </span>
          )}
        </p>
      </Link>
    </article>
  );
}
