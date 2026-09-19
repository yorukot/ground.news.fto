// M3 basic dialog: https://m3.material.io/components/dialogs/overview
// Base UI: https://base-ui.com/react/components/dialog
import { Dialog as BaseDialog } from "@base-ui/react/dialog";
import type { ReactNode } from "react";
import styles from "./Dialog.module.css";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  children?: ReactNode;
  /** Buttons, right-aligned; the confirming action goes last. */
  actions: ReactNode;
};

export function Dialog({ open, onOpenChange, title, description, children, actions }: Props) {
  return (
    <BaseDialog.Root open={open} onOpenChange={onOpenChange}>
      <BaseDialog.Portal>
        <BaseDialog.Backdrop className={styles.backdrop} />
        <BaseDialog.Popup className={styles.popup}>
          <BaseDialog.Title className={`${styles.title} md-headline-small`}>
            {title}
          </BaseDialog.Title>
          {description && (
            <BaseDialog.Description className={`${styles.description} md-body-medium`}>
              {description}
            </BaseDialog.Description>
          )}
          {children && <div className={styles.content}>{children}</div>}
          <div className={styles.actions}>{actions}</div>
        </BaseDialog.Popup>
      </BaseDialog.Portal>
    </BaseDialog.Root>
  );
}
