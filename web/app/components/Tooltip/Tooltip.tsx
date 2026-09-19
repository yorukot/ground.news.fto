// M3 plain tooltip: https://m3.material.io/components/tooltips/overview
// Base UI: https://base-ui.com/react/components/tooltip
//
// Tooltips are supplementary only and don't appear on touch devices, so the
// trigger must already carry its own accessible name (aria-label).
import { Tooltip as BaseTooltip } from "@base-ui/react/tooltip";
import type { ReactElement } from "react";
import styles from "./Tooltip.module.css";

export const TooltipProvider = BaseTooltip.Provider;

type Props = {
  label: string;
  /** The trigger element; it receives the tooltip's event handlers. */
  children: ReactElement<Record<string, unknown>>;
};

export function Tooltip({ label, children }: Props) {
  return (
    <BaseTooltip.Root>
      <BaseTooltip.Trigger render={children} />
      <BaseTooltip.Portal>
        <BaseTooltip.Positioner side="bottom" sideOffset={4}>
          <BaseTooltip.Popup className={`${styles.popup} md-body-small`}>{label}</BaseTooltip.Popup>
        </BaseTooltip.Positioner>
      </BaseTooltip.Portal>
    </BaseTooltip.Root>
  );
}
