import { describe, expect, it } from "vitest";
import { createI18n } from "~/i18n/i18n";
import { isLongRunning } from "./format";

const en = createI18n("en").fmt;
const zh = createI18n("zh-TW").fmt;

describe("formatters", () => {
  it("shows times in Taiwan time whatever the server's zone", () => {
    // 17:30 UTC is 01:30 the next day in Taipei.
    expect(en.dateTime("2026-09-02T17:30:00Z")).toContain("3 Sept 2026");
    expect(en.dateTime("2026-09-02T17:30:00Z")).toContain("01:30");
  });

  it("never shifts a calendar date across time zones", () => {
    expect(en.calendarDate("2026-03-13")).toBe("13 Mar 2026");
    expect(zh.calendarDate("2026-03-13")).toBe("2026年3月13日");
    expect(en.calendarDate("not-a-date")).toBe("not-a-date");
  });

  it("describes only long gaps between timeline steps", () => {
    expect(en.gap("2026-03-01", "2026-03-20")).toBeNull();
    expect(en.gap("2026-03-15", "2026-06-16")).toBe("3 months later");
    expect(en.gap("2026-03-15", "2026-04-16")).toBe("1 month later");
    expect(en.gap("2024-01-01", "2026-03-01")).toBe("2 years and 2 months later");
    expect(zh.gap("2026-03-15", "2026-06-16")).toBe("3 個月後");
  });

  it("flags events that started long before their last update", () => {
    expect(isLongRunning("2026-03-01T00:00:00Z", "2026-09-01T00:00:00Z")).toBe(true);
    expect(isLongRunning("2026-08-25T00:00:00Z", "2026-09-01T00:00:00Z")).toBe(false);
  });
});
