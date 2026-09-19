import { Form } from "react-router";
import { Button, ButtonLink } from "~/components/Button/Button";
import { useI18n } from "~/i18n/i18n";
import { CONTENT_LANG } from "~/i18n/locales";
import { adminFetch, type LinkDecision } from "~/lib/admin.server";
import { messagesFor } from "~/lib/meta";
import type { Route } from "./+types/links";
import styles from "./admin.module.css";

export function meta({ matches }: Route.MetaArgs) {
  return [{ title: messagesFor(matches).admin.linksTitle }, { name: "robots", content: "noindex" }];
}

export async function loader({ request }: Route.LoaderArgs) {
  const res = await adminFetch(request, "/api/v1/admin/links");
  if (!res.ok) throw new Response("Admin API error", { status: 502 });
  return (await res.json()) as { links: LinkDecision[] };
}

export default function AdminLinks({ loaderData }: Route.ComponentProps) {
  const { m, fmt } = useI18n();
  return (
    <div className={styles.page}>
      <div className={styles.heading}>
        <h1 className="md-headline-small">{m.admin.linksTitle}</h1>
        <Form method="post" action="/admin/logout">
          <Button type="submit" variant="text">
            {m.admin.signOut}
          </Button>
        </Form>
      </div>
      <p className={`${styles.intro} md-body-large`}>{m.admin.linksIntro}</p>

      {loaderData.links.length === 0 ? (
        <p className={`${styles.intro} md-body-large`}>{m.admin.noLinks}</p>
      ) : (
        <div className={styles.tableWrap}>
          <table className={`${styles.table} md-body-medium`}>
            <thead className="md-title-small">
              <tr>
                <th scope="col">{m.admin.colConfidence}</th>
                <th scope="col">{m.admin.colArticle}</th>
                <th scope="col">{m.admin.colEvent}</th>
                <th scope="col">{m.admin.colPublished}</th>
                <th scope="col">
                  <span className="visually-hidden">{m.admin.colAction}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {loaderData.links.map((link) => (
                <tr key={link.articleId}>
                  <td className={styles.number}>{link.confidence.toFixed(2)}</td>
                  <td>
                    <span lang={CONTENT_LANG}>{link.headline}</span>
                    <span className={styles.secondary} lang={CONTENT_LANG}>
                      {link.outletName}
                    </span>
                  </td>
                  <td>{link.eventTitle}</td>
                  <td className={styles.nowrap}>{fmt.dateTime(link.publishedAt)}</td>
                  <td>
                    <ButtonLink variant="text" to={`/admin/events/${link.eventId}`}>
                      {m.admin.openEvent}
                    </ButtonLink>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
