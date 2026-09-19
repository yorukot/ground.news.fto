import { Form, redirect, useNavigation } from "react-router";
import { Button } from "~/components/Button/Button";
import { TextField } from "~/components/TextField/TextField";
import { useI18n } from "~/i18n/i18n";
import { sessionCookie, tokenIsValid } from "~/lib/admin.server";
import { messagesFor } from "~/lib/meta";
import type { Route } from "./+types/login";
import styles from "./admin.module.css";

export function meta({ matches }: Route.MetaArgs) {
  return [{ title: messagesFor(matches).admin.loginTitle }, { name: "robots", content: "noindex" }];
}

export async function action({ request }: Route.ActionArgs) {
  const token = String((await request.formData()).get("token") ?? "").trim();
  if (!token || !(await tokenIsValid(token))) {
    return { failed: true };
  }
  return redirect("/admin", { headers: { "Set-Cookie": sessionCookie(token) } });
}

export default function AdminLogin({ actionData }: Route.ComponentProps) {
  const { m } = useI18n();
  const busy = useNavigation().state !== "idle";
  return (
    <div className={styles.narrow}>
      <h1 className="md-headline-small">{m.admin.loginTitle}</h1>
      <Form method="post" className={styles.form}>
        <TextField
          label={m.admin.token}
          name="token"
          type="password"
          autoComplete="current-password"
          required
          supportingText={m.admin.tokenHelp}
          error={actionData?.failed ? m.admin.badToken : undefined}
        />
        <div>
          <Button type="submit" disabled={busy} focusableWhenDisabled>
            {m.admin.signIn}
          </Button>
        </div>
      </Form>
    </div>
  );
}
