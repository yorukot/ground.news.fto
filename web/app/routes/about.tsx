import { useI18n } from "~/i18n/i18n";
import { messagesFor, pageMeta } from "~/lib/meta";
import type { Route } from "./+types/about";
import styles from "./about.module.css";

export function meta({ matches }: Route.MetaArgs) {
  const m = messagesFor(matches);
  return pageMeta(m, m.about.title, m.site.description);
}

export default function About() {
  const { m } = useI18n();
  return (
    <article className={styles.page}>
      <h1 className="md-headline-small">{m.about.title}</h1>
      {m.about.sections.map((section) => (
        <section key={section.heading} className={styles.section}>
          <h2 className="md-title-large">{section.heading}</h2>
          {section.body.map((paragraph) => (
            <p key={paragraph} className="md-body-large">
              {paragraph}
            </p>
          ))}
        </section>
      ))}
    </article>
  );
}
