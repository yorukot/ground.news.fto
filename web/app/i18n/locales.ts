import { en, type Messages } from "./messages/en";
import { zhTW } from "./messages/zh-TW";

// To add a language: write messages/<locale>.ts against the Messages type and
// register it here. English is the default and the source of the key set.
export const LOCALES = {
  en: { messages: en, label: "English", htmlLang: "en", intl: "en-GB" },
  "zh-TW": { messages: zhTW, label: "繁體中文", htmlLang: "zh-Hant-TW", intl: "zh-TW" },
} as const satisfies Record<
  string,
  { messages: Messages; label: string; htmlLang: string; intl: string }
>;

export type Locale = keyof typeof LOCALES;

export const DEFAULT_LOCALE: Locale = "en";

export function isLocale(value: unknown): value is Locale {
  return typeof value === "string" && value in LOCALES;
}

/** Original headlines and outlet names are Chinese whatever the UI language. */
export const CONTENT_LANG = "zh-Hant-TW";
