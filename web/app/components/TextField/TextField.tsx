// M3 outlined text field: https://m3.material.io/components/text-fields/overview
// Base UI: https://base-ui.com/react/components/field
import { Field } from "@base-ui/react/field";
import type { ComponentProps } from "react";
import styles from "./TextField.module.css";

type Props = Omit<ComponentProps<typeof Field.Control>, "className"> & {
  label: string;
  name: string;
  supportingText?: string;
  /** Shown instead of the supporting text, and marks the field invalid. */
  error?: string;
};

export function TextField({ label, name, supportingText, error, ...input }: Props) {
  return (
    <Field.Root name={name} invalid={Boolean(error)} className={styles.root}>
      <div className={styles.container}>
        {/* A space placeholder lets CSS tell an empty field from a filled one. */}
        <Field.Control className={`${styles.input} md-body-large`} placeholder=" " {...input} />
        <Field.Label className={styles.label}>{label}</Field.Label>
      </div>
      {error ? (
        <p className={`${styles.error} md-body-small`} role="alert">
          {error}
        </p>
      ) : (
        supportingText && (
          <Field.Description className={`${styles.supporting} md-body-small`}>
            {supportingText}
          </Field.Description>
        )
      )}
    </Field.Root>
  );
}
