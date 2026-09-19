// Date formatting. Everything is shown in Taiwan time and in the UI locale,
// never the browser's, so the server and the client render the same text.
import type { Messages } from "~/i18n/messages/en";

const TIME_ZONE = "Asia/Taipei";
const DAY_MS = 24 * 60 * 60 * 1000;

export type Formatters = ReturnType<typeof createFormatters>;

export function createFormatters(intlLocale: string, m: Messages) {
  const dateTime = new Intl.DateTimeFormat(intlLocale, {
    timeZone: TIME_ZONE,
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  });
  const date = new Intl.DateTimeFormat(intlLocale, {
    timeZone: TIME_ZONE,
    year: "numeric",
    month: "short",
    day: "numeric",
  });
  const yearMonth = new Intl.DateTimeFormat(intlLocale, {
    timeZone: TIME_ZONE,
    year: "numeric",
    month: "short",
  });
  // Calendar dates carry no time zone; format them as UTC so they never shift.
  const calendarDate = new Intl.DateTimeFormat(intlLocale, {
    timeZone: "UTC",
    year: "numeric",
    month: "short",
    day: "numeric",
  });

  return {
    dateTime: (iso: string) => dateTime.format(new Date(iso)),
    date: (iso: string) => date.format(new Date(iso)),
    yearMonth: (iso: string) => yearMonth.format(new Date(iso)),

    /** Formats a calendar date (YYYY-MM-DD). */
    calendarDate: (ymd: string) => {
      const day = calendarDayNumber(ymd);
      return day === null ? ymd : calendarDate.format(new Date(day * DAY_MS));
    },

    /**
     * Describes a long quiet gap between two timeline steps, e.g. "3 months later".
     * Returns null when the gap is under 30 days and not worth calling out.
     */
    gap: (fromYmd: string, toYmd: string): string | null => {
      const from = calendarDayNumber(fromYmd);
      const to = calendarDayNumber(toYmd);
      if (from === null || to === null) return null;
      const days = to - from;
      if (days < 30) return null;
      if (days < 365) return m.timeline.gapMonths(Math.round(days / 30));
      const years = Math.floor(days / 365);
      return m.timeline.gapYears(years, Math.round((days - years * 365) / 30));
    },
  };
}

function calendarDayNumber(ymd: string): number | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(ymd);
  if (!match) return null;
  return Math.floor(Date.UTC(Number(match[1]), Number(match[2]) - 1, Number(match[3])) / DAY_MS);
}

/** True when the event started long enough before its last update to be worth noting. */
export function isLongRunning(firstSeenIso: string, updatedIso: string): boolean {
  return new Date(updatedIso).getTime() - new Date(firstSeenIso).getTime() > 30 * DAY_MS;
}
