import type { Locator, Page } from "@playwright/test";

/**
 * Finds an event card on the homepage, loading more pages until it shows up.
 * On a database that also holds crawled events, the seeded samples are not
 * on the first page.
 */
export async function findEventLink(page: Page, name: RegExp): Promise<Locator> {
  const link = page.getByRole("link", { name });
  const more = page.getByRole("button", { name: "Load more" });
  for (let i = 0; i < 30; i++) {
    if (await link.count()) break;
    if (!(await more.count())) break;
    const before = await page.getByRole("article").count();
    await more.click();
    await page.waitForFunction(
      (n) =>
        document.querySelectorAll("article").length > n || !document.querySelector("form button"),
      before,
    );
  }
  return link.first();
}
