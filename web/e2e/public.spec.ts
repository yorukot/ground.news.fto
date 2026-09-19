import { expect, test } from "@playwright/test";
import { findEventLink } from "./helpers";

const SAMPLE_EVENT = "[Sample] Sample City Council reviews next year's budget";

test("home lists events and leads to an event page", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1, name: "Current events" })).toBeVisible();

  await (await findEventLink(page, /Sample City Council reviews/)).click();
  await expect(page.getByRole("heading", { level: 1, name: SAMPLE_EVENT })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Timeline" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Coverage" })).toBeVisible();

  // An approximate date is marked in text.
  await expect(page.getByText(/^c\. \d/)).toBeVisible();

  // The wire story appears once, with the other outlets folded under it.
  const reprints = page.getByRole("button", { name: "Also run by 3 other outlets" });
  await expect(reprints).toBeVisible();
  await reprints.click();
  await expect(page.getByRole("link", { name: /聯合新聞網 \(opens in a new tab\)/ })).toBeVisible();

  // Originals open in a new tab and never hand over the opener.
  const original = page.getByRole("link", { name: /Read the original/ }).first();
  await expect(original).toHaveAttribute("target", "_blank");
  await expect(original).toHaveAttribute("rel", /noopener/);
});

test("a timeline chip jumps to that outlet's article", async ({ page }) => {
  await page.goto("/");
  await (await findEventLink(page, /Sample City Council reviews/)).click();
  await page
    .getByRole("list", { name: "Outlets that reported this development" })
    .first()
    .getByRole("link")
    .first()
    .click();
  await expect(page).toHaveURL(/#article-\d+$/);
});

test("event pages are server-rendered with link-preview tags", async ({ request }) => {
  const home = await (await request.get("/")).text();
  const id = home.match(/href="\/events\/(\d+)"/)?.[1];
  expect(id).toBeTruthy();

  // No browser here: this is what a crawler or a link preview sees.
  const html = await (await request.get(`/events/${id}`)).text();
  expect(html).toContain('property="og:title"');
  expect(html).toContain("Read the original");
  expect((await request.get("/events/99999999")).status()).toBe(404);
});

test("the language can be switched and is remembered", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator("html")).toHaveAttribute("lang", "en");

  await page.getByRole("button", { name: "Settings" }).click();
  await page.getByRole("menuitemradio", { name: "繁體中文" }).click();
  await expect(page.locator("html")).toHaveAttribute("lang", "zh-Hant-TW");
  await expect(page.getByRole("heading", { level: 1, name: "目前的事件" })).toBeVisible();

  await page.reload();
  await expect(page.getByRole("heading", { level: 1, name: "目前的事件" })).toBeVisible();
});

test("the dark theme choice is rendered by the server", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Settings" }).click();
  await page.getByRole("menuitemradio", { name: "Dark" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");

  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});
