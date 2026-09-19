// Helpers for route `meta` exports, which run outside React and so can't use
// the i18n context: the locale comes from the root loader's data instead.
import { DEFAULT_LOCALE, isLocale, LOCALES } from "~/i18n/locales";
import type { Messages } from "~/i18n/messages/en";

type Match = { id: string; loaderData?: unknown } | undefined;

export function messagesFor(matches: readonly Match[]): Messages {
  const root = matches.find((match) => match?.id === "root")?.loaderData;
  const locale =
    root && typeof root === "object" && "locale" in root && isLocale(root.locale)
      ? root.locale
      : DEFAULT_LOCALE;
  return LOCALES[locale].messages;
}

export function pageMeta(m: Messages, title: string | null, description: string) {
  const fullTitle = title ? `${title} · ${m.site.name}` : `${m.site.name} · ${m.site.tagline}`;
  return [
    { title: fullTitle },
    { name: "description", content: description },
    { property: "og:title", content: title ?? m.site.name },
    { property: "og:description", content: description },
    { property: "og:site_name", content: m.site.name },
    { property: "og:type", content: title ? "article" : "website" },
    { name: "twitter:card", content: "summary" },
  ];
}
