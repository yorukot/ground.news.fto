// Saves the appearance and language choices as cookies. Submitted by the
// settings menu through a fetcher; the loaders revalidate afterwards, so a
// language change re-renders the page in the new language.
import { data } from "react-router";
import { isLocale } from "~/i18n/locales";
import { isThemeChoice, localeCookie, themeCookie } from "~/lib/preferences";
import type { Route } from "./+types/preferences";

export async function action({ request }: Route.ActionArgs) {
  const form = await request.formData();
  const headers = new Headers();

  const theme = form.get("theme");
  if (isThemeChoice(theme)) headers.append("Set-Cookie", themeCookie(theme));

  const locale = form.get("locale");
  if (isLocale(locale)) headers.append("Set-Cookie", localeCookie(locale));

  return data({ ok: true }, { headers });
}
