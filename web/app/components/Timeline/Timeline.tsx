// The event timeline. M3 has no timeline component, so this is composed from
// M3 parts (list, assist chips, text button) and tokens; see docs/components.md.
//
// Neutrality rules: the first outlet to report a step is not marked as such
// (chips are simply in publish order), and an approximate date is marked in
// text ("c."), never by the node's look alone.
import { Collapsible } from "@base-ui/react/collapsible";
import { useI18n } from "~/i18n/i18n";
import { CONTENT_LANG } from "~/i18n/locales";
import type { TimelineStep } from "~/lib/api.server";
import { AssistChipLink } from "../Chip/Chip";
import { Icon } from "../Icon/Icon";
import styles from "./Timeline.module.css";

const COLLAPSE_ABOVE = 8;
const KEEP_VISIBLE = 5;

export function articleAnchor(articleId: number): string {
  return `article-${articleId}`;
}

export function Timeline({ steps }: { steps: TimelineStep[] }) {
  const { m } = useI18n();
  if (steps.length === 0) {
    return <p className={`${styles.empty} md-body-large`}>{m.timeline.empty}</p>;
  }
  if (steps.length <= COLLAPSE_ABOVE) {
    return <StepList steps={steps} previous={null} />;
  }

  const earlier = steps.slice(0, -KEEP_VISIBLE);
  const latest = steps.slice(-KEEP_VISIBLE);
  return (
    <Collapsible.Root>
      <Collapsible.Trigger
        className={`${styles.expand} md-label-large md-state-layer md-focus-ring`}
      >
        <span className={styles.expandClosed}>{m.timeline.showEarlier(earlier.length)}</span>
        <span className={styles.expandOpen}>{m.timeline.hideEarlier}</span>
        <Icon name="expand_more" size={18} className={styles.expandIcon} />
      </Collapsible.Trigger>
      <Collapsible.Panel className={styles.panel}>
        <StepList steps={earlier} previous={null} />
      </Collapsible.Panel>
      <StepList steps={latest} previous={earlier.at(-1) ?? null} />
    </Collapsible.Root>
  );
}

function StepList({ steps, previous }: { steps: TimelineStep[]; previous: TimelineStep | null }) {
  const { m, fmt } = useI18n();
  return (
    <ol className={styles.list}>
      {steps.map((step, i) => {
        const before = i === 0 ? previous : (steps[i - 1] ?? null);
        const gap = before ? fmt.gap(before.happenedOn, step.happenedOn) : null;
        return (
          <li key={step.articleId} className={styles.step}>
            {gap && <p className={`${styles.gap} md-label-medium`}>{gap}</p>}
            <div className={styles.row}>
              <span
                className={step.dateIsApproximate ? styles.nodeApproximate : styles.node}
                aria-hidden="true"
              />
              <div className={styles.body}>
                <p className={`${styles.date} md-label-large`}>
                  <time dateTime={step.happenedOn}>
                    {step.dateIsApproximate && `${m.timeline.approximate} `}
                    {fmt.calendarDate(step.happenedOn)}
                  </time>
                </p>
                <p className="md-body-large">{step.development}</p>
                <ul className={styles.reports} aria-label={m.timeline.reportedBy}>
                  {step.reports.map((report) => (
                    <li key={`${report.outlet.id}-${report.articleId}`}>
                      <AssistChipLink
                        href={`#${articleAnchor(report.articleId)}`}
                        lang={CONTENT_LANG}
                      >
                        {report.outlet.name}
                      </AssistChipLink>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </li>
        );
      })}
    </ol>
  );
}
