// M3 small top app bar: https://m3.material.io/components/top-app-bar/overview
import type { ReactNode } from "react";
import { Link, NavLink } from "react-router";
import { useI18n } from "~/i18n/i18n";
import styles from "./TopAppBar.module.css";

export function TopAppBar({ actions }: { actions?: ReactNode }) {
  const { m } = useI18n();
  const links = [
    { to: "/outlets", label: m.nav.outlets },
    { to: "/about", label: m.nav.about },
  ];
  return (
    <header className={styles.bar}>
      <div className={styles.inner}>
        <Link to="/" className={`${styles.title} md-title-large md-focus-ring`}>
          {m.site.name}
        </Link>
        <nav className={styles.nav} aria-label={m.nav.label}>
          {links.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              className={`${styles.link} md-label-large md-state-layer md-focus-ring`}
            >
              {link.label}
            </NavLink>
          ))}
        </nav>
        <div className={styles.actions}>{actions}</div>
      </div>
    </header>
  );
}
