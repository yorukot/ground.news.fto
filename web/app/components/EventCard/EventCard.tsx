// An event on the homepage: a filled card that is one link to the event page.
import { Link } from "react-router";
import { useI18n } from "~/i18n/i18n";
import type { EventSummary } from "~/lib/api.server";
import { isLongRunning } from "~/lib/format";
import styles from "./EventCard.module.css";

export function EventCard({
  event,
  variant = "standard",
}: {
  event: EventSummary;
  variant?: "featured" | "standard";
}) {
  const { m, fmt } = useI18n();
  return (
    <article className={styles.card} data-variant={variant}>
      <Link
        to={`/events/${event.id}`}
        className={`${styles.link} md-state-layer md-focus-ring`}
        prefetch="intent"
      >
        {event.imageUrl && (
          <div className={styles.imageFrame}>
            <img
              src={event.imageUrl}
              alt=""
              loading={variant === "featured" ? "eager" : "lazy"}
              referrerPolicy="no-referrer"
            />
          </div>
        )}
        <div className={styles.content}>
          <div className={`${styles.eyebrow} md-label-medium`}>
            <span className={styles.status}>
              <span className={styles.dot} aria-hidden="true" />
              {variant === "featured" ? m.home.featured : m.eventCard.developing}
            </span>
            <time dateTime={event.updatedAt}>{fmt.date(event.updatedAt)}</time>
          </div>
          <h2 className={styles.title}>{event.title}</h2>
          {event.latestDevelopment && (
            <div className={styles.development}>
              <p className={`${styles.latestLabel} md-label-medium`}>{m.eventCard.latest}</p>
              <p className={`${styles.latest} md-body-large`}>{event.latestDevelopment}</p>
            </div>
          )}
          <div className={`${styles.meta} md-label-medium`}>
            <div className={styles.metrics}>
              <span>{m.eventCard.outlets(event.outletCount)}</span>
              <span>{m.eventCard.articles(event.articleCount)}</span>
            </div>
            <span className={styles.updated}>
              {m.eventCard.updated}{" "}
              <time dateTime={event.updatedAt}>{fmt.dateTime(event.updatedAt)}</time>
            </span>
            {isLongRunning(event.firstSeenAt, event.updatedAt) && (
              <span>
                {m.eventCard.since}{" "}
                <time dateTime={event.firstSeenAt}>{fmt.yearMonth(event.firstSeenAt)}</time>
              </span>
            )}
            <span className={styles.arrow} aria-hidden="true">
              →
            </span>
          </div>
        </div>
      </Link>
    </article>
  );
}
