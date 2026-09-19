// Reader preferences kept in cookies, so the server renders the right theme
// and language and there is no flash. "system" means no explicit theme choice.
import { DEFAULT_LOCALE, isLocale, type Locale } from "~/i18n/locales";

export type ThemeChoice = "system" | "light" | "dark";

export function isThemeChoice(value: unknown): value is ThemeChoice {
  return value === "system" || value === "light" || value === "dark";
}

function readCookie(header: string | null, name: string): string | null {
  const match = header?.match(new RegExp(`(?:^|;\\s*)${name}=([^;]*)`));
  return match?.[1] ? decodeURIComponent(match[1]) : null;
}

export function readPreferences(cookieHeader: string | null): {
  theme: ThemeChoice;
  locale: Locale;
} {
  const theme = readCookie(cookieHeader, "theme");
  const locale = readCookie(cookieHeader, "locale");
  return {
    theme: isThemeChoice(theme) ? theme : "system",
    locale: isLocale(locale) ? locale : DEFAULT_LOCALE,
  };
}

const YEAR = 60 * 60 * 24 * 365;

export function themeCookie(choice: ThemeChoice): string {
  return choice === "system"
    ? "theme=; Path=/; SameSite=Lax; Max-Age=0"
    : `theme=${choice}; Path=/; SameSite=Lax; Max-Age=${YEAR}`;
}

export function localeCookie(locale: Locale): string {
  return `locale=${encodeURIComponent(locale)}; Path=/; SameSite=Lax; Max-Age=${YEAR}`;
}
