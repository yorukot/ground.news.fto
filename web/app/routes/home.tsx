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
  const { m } = useI18n();
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
  const loading = more.state !== "idle";

  return (
    <>
      <h1 className={`${styles.heading} md-headline-small`}>{m.home.title}</h1>
      {events.length === 0 ? (
        <p className={`${styles.empty} md-body-large`}>{m.home.empty}</p>
      ) : (
        <ul className={styles.grid}>
          {events.map((event) => (
            <li key={event.id}>
              <EventCard event={event} />
            </li>
          ))}
        </ul>
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
    </>
  );
}
