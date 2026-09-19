import { useEffect, useState } from "react";
import { data, redirect, useFetcher } from "react-router";
import { Button, ButtonLink } from "~/components/Button/Button";
import { Dialog } from "~/components/Dialog/Dialog";
import { TextField } from "~/components/TextField/TextField";
import { useI18n } from "~/i18n/i18n";
import { CONTENT_LANG } from "~/i18n/locales";
import { adminFetch, type EventMatch, errorMessage } from "~/lib/admin.server";
import { getEvent } from "~/lib/api.server";
import { messagesFor } from "~/lib/meta";
import type { Route } from "./+types/event";
import styles from "./admin.module.css";
import type { loader as searchLoader } from "./event-search";

export function meta({ matches }: Route.MetaArgs) {
  return [{ title: messagesFor(matches).admin.title }, { name: "robots", content: "noindex" }];
}

export async function loader({ params, request }: Route.LoaderArgs) {
  // Checks the session before showing anything.
  await adminFetch(request, "/api/v1/admin/session");
  const done = new URL(request.url).searchParams.get("done");
  return { event: await getEvent(params.id, request.signal), done };
}

export async function action({ params, request }: Route.ActionArgs) {
  const form = await request.formData();
  const intent = form.get("intent");

  if (intent === "merge") {
    const res = await adminFetch(request, `/api/v1/admin/events/${params.id}/merge`, {
      method: "POST",
      body: JSON.stringify({ fromEventId: Number(form.get("eventId")) }),
    });
    if (!res.ok) return data({ error: await errorMessage(res) }, { status: 400 });
    return redirect(`/admin/events/${params.id}?done=merged`);
  }

  if (intent === "move") {
    const eventId = Number(form.get("eventId") ?? 0);
    const newEventTitle = String(form.get("newEventTitle") ?? "").trim();
    const res = await adminFetch(request, `/api/v1/admin/articles/${form.get("articleId")}/move`, {
      method: "POST",
      body: JSON.stringify(eventId ? { eventId } : { newEventTitle }),
    });
    if (!res.ok) return data({ error: await errorMessage(res) }, { status: 400 });
    const moved = (await res.json()) as { eventId: number };
    return redirect(`/admin/events/${moved.eventId}?done=moved`);
  }

  return data({ error: "Unknown action" }, { status: 400 });
}

type DialogState = { kind: "merge" } | { kind: "move"; articleId: number; headline: string } | null;

export default function AdminEvent({ loaderData, params }: Route.ComponentProps) {
  const { m, fmt } = useI18n();
  const { event, done } = loaderData;
  const [dialog, setDialog] = useState<DialogState>(null);

  // A successful correction redirects, which delivers fresh loader data: that
  // is the signal to close the dialog. A failed one keeps it open with the error.
  useEffect(() => {
    setDialog(null);
  }, [loaderData]);

  return (
    <div className={styles.page}>
      <div>
        <ButtonLink variant="text" to="/admin">
          {m.admin.backToLinks}
        </ButtonLink>
      </div>
      {done && (
        <p className={`${styles.notice} md-body-medium`} role="status">
          {done === "merged" ? m.admin.merged : m.admin.moved}
        </p>
      )}
      <div className={styles.heading}>
        <h1 className="md-headline-small">{event.title}</h1>
        <ButtonLink variant="text" to={`/events/${event.id}`}>
          {m.admin.publicPage}
        </ButtonLink>
      </div>
      <div>
        <Button variant="tonal" onClick={() => setDialog({ kind: "merge" })}>
          {m.admin.mergeButton}
        </Button>
      </div>

      <div className={styles.tableWrap}>
        <table className={`${styles.table} md-body-medium`}>
          <thead className="md-title-small">
            <tr>
              <th scope="col">{m.admin.colArticle}</th>
              <th scope="col">{m.admin.colPublished}</th>
              <th scope="col">
                <span className="visually-hidden">{m.admin.colAction}</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {event.articles.map((article) => (
              <tr key={article.id}>
                <td>
                  <span lang={CONTENT_LANG}>{article.headline}</span>
                  <span className={styles.secondary}>
                    <span lang={CONTENT_LANG}>{article.outlet.name}</span>
                    {article.reprints.length > 0 &&
                      ` · +${article.reprints.length} ${m.admin.reprintNote}`}
                  </span>
                </td>
                <td className={styles.nowrap}>{fmt.dateTime(article.publishedAt)}</td>
                <td>
                  <Button
                    variant="text"
                    onClick={() =>
                      setDialog({ kind: "move", articleId: article.id, headline: article.headline })
                    }
                  >
                    {m.admin.moveButton}
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <CorrectionDialog
        key={dialog ? `${dialog.kind}-${"articleId" in dialog ? dialog.articleId : ""}` : "closed"}
        state={dialog}
        currentEventId={Number(params.id)}
        onClose={() => setDialog(null)}
      />
    </div>
  );
}

function CorrectionDialog({
  state,
  currentEventId,
  onClose,
}: {
  state: DialogState;
  currentEventId: number;
  onClose: () => void;
}) {
  const { m } = useI18n();
  const search = useFetcher<typeof searchLoader>();
  const submit = useFetcher<typeof action>();
  const [picked, setPicked] = useState<EventMatch | null>(null);
  const [newTitle, setNewTitle] = useState("");

  const isMove = state?.kind === "move";
  const busy = submit.state !== "idle";
  const matches = (search.data?.events ?? []).filter((e) => e.id !== currentEventId);
  const canSubmit = Boolean(picked) || (isMove && newTitle.trim().length > 0);

  function confirm() {
    if (!state) return;
    const form = new FormData();
    form.set("intent", state.kind);
    if (picked) form.set("eventId", String(picked.id));
    else form.set("newEventTitle", newTitle.trim());
    if (state.kind === "move") form.set("articleId", String(state.articleId));
    submit.submit(form, { method: "post" });
  }

  return (
    <Dialog
      open={state !== null}
      onOpenChange={(open) => {
        if (!open && !busy) onClose();
      }}
      title={isMove ? m.admin.moveTitle : m.admin.mergeTitle}
      description={isMove ? m.admin.moveDescription : m.admin.mergeDescription}
      actions={
        <>
          <Button variant="text" onClick={onClose} disabled={busy}>
            {m.admin.cancel}
          </Button>
          <Button
            variant="filled"
            onClick={confirm}
            disabled={!canSubmit || busy}
            focusableWhenDisabled
          >
            {isMove ? m.admin.move : m.admin.merge}
          </Button>
        </>
      }
    >
      {state?.kind === "move" && (
        <p className="md-body-medium" lang={CONTENT_LANG}>
          {state.headline}
        </p>
      )}
      <TextField
        label={m.admin.searchEvents}
        name="q"
        type="search"
        autoComplete="off"
        supportingText={m.admin.searchHelp}
        onChange={(e) => {
          setPicked(null);
          search.load(`/admin/event-search?q=${encodeURIComponent(e.currentTarget.value)}`);
        }}
      />
      {search.data && matches.length === 0 && (
        <p className={`${styles.secondary} md-body-medium`}>{m.admin.noMatches}</p>
      )}
      {matches.length > 0 && (
        <fieldset className={styles.choices}>
          <legend className="visually-hidden">{m.admin.searchEvents}</legend>
          {matches.map((match) => (
            <label key={match.id} className={`${styles.choice} md-body-medium`}>
              <input
                type="radio"
                name="target"
                checked={picked?.id === match.id}
                onChange={() => {
                  setPicked(match);
                  setNewTitle("");
                }}
              />
              <span>
                {match.title}
                <span className={styles.secondary}>
                  #{match.id} · {m.admin.articles(match.articleCount)}
                </span>
              </span>
            </label>
          ))}
        </fieldset>
      )}
      {isMove && (
        <>
          <p className="md-title-small">{m.admin.orNewEvent}</p>
          <TextField
            label={m.admin.newEventTitle}
            name="newEventTitle"
            value={newTitle}
            supportingText={m.admin.newEventHelp}
            onChange={(e) => {
              setNewTitle(e.currentTarget.value);
              setPicked(null);
            }}
          />
        </>
      )}
      {submit.data && "error" in submit.data && (
        <p className={`${styles.errorText} md-body-medium`} role="alert">
          {submit.data.error}
        </p>
      )}
    </Dialog>
  );
}
