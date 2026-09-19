import { Fragment } from "react";
import { Divider } from "~/components/Divider/Divider";
import { useI18n } from "~/i18n/i18n";
import { CONTENT_LANG } from "~/i18n/locales";
import { listOutlets } from "~/lib/api.server";
import { messagesFor, pageMeta } from "~/lib/meta";
import type { Route } from "./+types/outlets";
import styles from "./outlets.module.css";

export async function loader({ request }: Route.LoaderArgs) {
  return { outlets: await listOutlets(request.signal) };
}

export function meta({ matches }: Route.MetaArgs) {
  const m = messagesFor(matches);
  return pageMeta(m, m.outlets.title, m.outlets.intro);
}

// A plain two-line list. No descriptions or categories of our own:
// describing an outlet would be labeling it.
export default function Outlets({ loaderData }: Route.ComponentProps) {
  const { m } = useI18n();
  return (
    <div className={styles.page}>
      <h1 className="md-headline-small">{m.outlets.title}</h1>
      <p className={`${styles.intro} md-body-large`}>{m.outlets.intro}</p>
      <ul className={styles.list}>
        {loaderData.outlets.map((outlet, i) => (
          <Fragment key={outlet.id}>
            {i > 0 && (
              <li aria-hidden="true">
                <Divider />
              </li>
            )}
            <li className={styles.item}>
              <span className="md-body-large" lang={CONTENT_LANG}>
                {outlet.name}
              </span>
              <span className={`${styles.domain} md-body-medium`}>{outlet.domain}</span>
            </li>
          </Fragment>
        ))}
      </ul>
    </div>
  );
}
