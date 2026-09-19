// M3 assist chip: https://m3.material.io/components/chips/overview
// Rendered as a link: on this site an assist chip always navigates.
import type { ReactNode } from "react";
import styles from "./Chip.module.css";

type Props = {
  href: string;
  /** Set when the label is in a different language from the page, e.g. an outlet name. */
  lang?: string;
  children: ReactNode;
};

export function AssistChipLink({ href, lang, children }: Props) {
  return (
    <a
      className={`${styles.chip} md-label-large md-state-layer md-focus-ring`}
      href={href}
      lang={lang}
    >
      {children}
    </a>
  );
}
