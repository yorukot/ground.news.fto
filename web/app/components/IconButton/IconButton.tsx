// M3 icon button (standard): https://m3.material.io/components/icon-buttons/overview
import { Button as BaseButton } from "@base-ui/react/button";
import type { ComponentProps } from "react";
import { Icon, type IconName } from "../Icon/Icon";
import styles from "./IconButton.module.css";

type Props = Omit<ComponentProps<typeof BaseButton>, "className" | "children" | "aria-label"> & {
  icon: IconName;
  /** Required: an icon alone has no accessible name. */
  label: string;
};

export function IconButton({ icon, label, ...rest }: Props) {
  return (
    <BaseButton
      className={`${styles.iconButton} md-state-layer md-focus-ring`}
      aria-label={label}
      {...rest}
    >
      <Icon name={icon} />
    </BaseButton>
  );
}
