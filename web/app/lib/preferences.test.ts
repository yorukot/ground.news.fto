import { describe, expect, it } from "vitest";
import { readPreferences } from "./preferences";

describe("readPreferences", () => {
  it("defaults to the system theme and English", () => {
    expect(readPreferences(null)).toEqual({ theme: "system", locale: "en" });
  });

  it("reads valid choices and ignores anything else", () => {
    expect(readPreferences("a=1; theme=dark; locale=zh-TW")).toEqual({
      theme: "dark",
      locale: "zh-TW",
    });
    expect(readPreferences("theme=neon; locale=xx")).toEqual({ theme: "system", locale: "en" });
  });
});
