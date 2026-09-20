// One article in an event's coverage. Every article uses this same card, in
// the order set by the plan: outlet and time, original headline, summary,
// link to the original. Nothing here varies by outlet.
import { Collapsible } from "@base-ui/react/collapsible";
import { useI18n } from "~/i18n/i18n";
import { CONTENT_LANG } from "~/i18n/locales";
import type { Article } from "~/lib/api.server";
import { ExternalButtonLink } from "../Button/Button";
import { Card } from "../Card/Card";
import { Icon } from "../Icon/Icon";
import { articleAnchor } from "../Timeline/Timeline";
import styles from "./ArticleCard.module.css";

export function ArticleCard({ article }: { article: Article }) {
  const { m, fmt } = useI18n();
  const headingId = `${articleAnchor(article.id)}-headline`;
  return (
    <Card
      as="article"
      variant="outlined"
      id={articleAnchor(article.id)}
      className={styles.card}
      aria-labelledby={headingId}
    >
      {article.imageUrl && (
        <img
          className={styles.image}
          src={article.imageUrl}
          alt=""
          loading="lazy"
          referrerPolicy="no-referrer"
        />
      )}
      <p className={styles.meta}>
        <span className="md-label-large" lang={CONTENT_LANG}>
          {article.outlet.name}
        </span>
        <time className="md-label-medium" dateTime={article.publishedAt ?? undefined}>
          {fmt.dateTime(article.publishedAt)}
        </time>
      </p>
      <h3 id={headingId} className="md-title-medium" lang={CONTENT_LANG}>
        {article.headline}
      </h3>
      {article.reprints.length > 0 && <Reprints article={article} />}
      {article.summary && <p className={`${styles.summary} md-body-large`}>{article.summary}</p>}
      {article.headlineOnly && (
        <p className={`${styles.note} md-body-medium`}>{m.article.headlineOnly}</p>
      )}
      <div className={styles.actions}>
        <ExternalButtonLink href={article.url}>{m.article.readOriginal}</ExternalButtonLink>
      </div>
    </Card>
  );
}

function Reprints({ article }: { article: Article }) {
  const { m, fmt } = useI18n();
  return (
    <Collapsible.Root className={styles.reprints}>
      <Collapsible.Trigger
        className={`${styles.reprintsTrigger} md-label-large md-state-layer md-focus-ring`}
      >
        {m.article.alsoRunBy(article.reprints.length)}
        <Icon name="expand_more" size={18} className={styles.reprintsIcon} />
      </Collapsible.Trigger>
      <Collapsible.Panel className={styles.reprintsPanel}>
        <ul className={styles.reprintsList}>
          {article.reprints.map((reprint) => (
            <li key={reprint.url} className="md-body-medium">
              <a href={reprint.url} target="_blank" rel="noopener noreferrer">
                <span lang={CONTENT_LANG}>{reprint.outlet.name}</span>
                <span className="visually-hidden"> {m.common.opensInNewTab}</span>
              </a>
              <time className={styles.reprintTime} dateTime={reprint.publishedAt ?? undefined}>
                {fmt.dateTime(reprint.publishedAt)}
              </time>
            </li>
          ))}
        </ul>
      </Collapsible.Panel>
    </Collapsible.Root>
  );
}
