import { createContext, type ReactNode, useContext, useMemo } from "react";
import { createFormatters, type Formatters } from "~/lib/format";
import { DEFAULT_LOCALE, LOCALES, type Locale } from "./locales";
import type { Messages } from "./messages/en";

type I18n = {
  locale: Locale;
  /** The message catalogue for the current locale. */
  m: Messages;
  /** Date formatters for the current locale, always in Taiwan time. */
  fmt: Formatters;
};

export function createI18n(locale: Locale): I18n {
  const entry = LOCALES[locale];
  return { locale, m: entry.messages, fmt: createFormatters(entry.intl, entry.messages) };
}

const I18nContext = createContext<I18n>(createI18n(DEFAULT_LOCALE));

export function I18nProvider({ locale, children }: { locale: Locale; children: ReactNode }) {
  const value = useMemo(() => createI18n(locale), [locale]);
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18n {
  return useContext(I18nContext);
}
