// M3 linear progress indicator: https://m3.material.io/components/progress-indicators/overview
// Base UI: https://base-ui.com/react/components/progress
import { Progress } from "@base-ui/react/progress";
import styles from "./LinearProgress.module.css";

/** Indeterminate progress, shown while a navigation is pending. */
export function LinearProgress({ label }: { label: string }) {
  return (
    <Progress.Root className={styles.root} value={null} aria-label={label}>
      <Progress.Track className={styles.track}>
        <Progress.Indicator className={styles.indicator} />
      </Progress.Track>
    </Progress.Root>
  );
}
