// M3 common buttons: https://m3.material.io/components/buttons/overview
// Base UI: https://base-ui.com/react/components/button
import { Button as BaseButton } from "@base-ui/react/button";
import type { ComponentProps, ReactNode } from "react";
import { Link, type LinkProps } from "react-router";
import { useI18n } from "~/i18n/i18n";
import { Icon, type IconName } from "../Icon/Icon";
import styles from "./Button.module.css";

export type ButtonVariant = "filled" | "tonal" | "outlined" | "text";

type Common = {
  variant?: ButtonVariant;
  /** Leading icon. */
  icon?: IconName;
  /** Trailing icon, e.g. open_in_new on links that leave the site. */
  trailingIcon?: IconName;
  children: ReactNode;
};

function classes(variant: ButtonVariant, hasIcon: boolean, hasTrailing: boolean, extra?: string) {
  return [
    styles.button,
    styles[variant],
    hasIcon && styles.withIcon,
    hasTrailing && styles.withTrailingIcon,
    "md-label-large md-state-layer md-focus-ring",
    extra,
  ]
    .filter(Boolean)
    .join(" ");
}

function Content({ icon, trailingIcon, children }: Common) {
  return (
    <>
      {icon && <Icon name={icon} size={18} />}
      <span>{children}</span>
      {trailingIcon && <Icon name={trailingIcon} size={18} />}
    </>
  );
}

type ButtonProps = Common & Omit<ComponentProps<typeof BaseButton>, "className" | "children">;

export function Button({ variant = "filled", icon, trailingIcon, children, ...rest }: ButtonProps) {
  return (
    <BaseButton className={classes(variant, !!icon, !!trailingIcon)} {...rest}>
      <Content icon={icon} trailingIcon={trailingIcon}>
        {children}
      </Content>
    </BaseButton>
  );
}

// Base UI: links have their own semantics and must not be rendered through
// Button, so link buttons are anchors sharing the same styles.

type ButtonLinkProps = Common & Omit<LinkProps, "className" | "children">;

/** An in-app link that looks like a button. */
export function ButtonLink({
  variant = "filled",
  icon,
  trailingIcon,
  children,
  ...rest
}: ButtonLinkProps) {
  return (
    <Link className={classes(variant, !!icon, !!trailingIcon)} {...rest}>
      <Content icon={icon} trailingIcon={trailingIcon}>
        {children}
      </Content>
    </Link>
  );
}

type ExternalButtonLinkProps = Common & { href: string };

/** A link to another site, opened in a new tab. */
export function ExternalButtonLink({
  variant = "text",
  icon,
  trailingIcon = "open_in_new",
  href,
  children,
}: ExternalButtonLinkProps) {
  const { m } = useI18n();
  return (
    <a
      className={classes(variant, !!icon, !!trailingIcon)}
      href={href}
      target="_blank"
      rel="noopener noreferrer"
    >
      <Content icon={icon} trailingIcon={trailingIcon}>
        {children}
      </Content>
      <span className="visually-hidden"> {m.common.opensInNewTab}</span>
    </a>
  );
}
