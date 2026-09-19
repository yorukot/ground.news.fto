// M3 cards: https://m3.material.io/components/cards/overview
import type { ComponentProps, ElementType } from "react";
import styles from "./Card.module.css";

type Props<T extends ElementType> = {
  /** Outlined sits on the page surface with a border; filled uses a container color. */
  variant: "outlined" | "filled";
  as?: T;
} & Omit<ComponentProps<T>, "as">;

export function Card<T extends ElementType = "div">({ variant, as, className, ...rest }: Props<T>) {
  const Component: ElementType = as ?? "div";
  return (
    <Component
      className={[styles.card, styles[variant], className].filter(Boolean).join(" ")}
      {...rest}
    />
  );
}
